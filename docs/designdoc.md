# Kaniko Design Doc 

_<span style="color:#666666;">Authors: [priyawadhwa@google.com](mailto:priyawadhwa@google.com)<span style="color:#666666;">, [dlorenc@google.com](mailto:dlorenc@google.com)</span></span>_


[TOC]



## Objective {#objective}

Kaniko aims to build container images, with enough Dockerfile support to be useful, without a dependency on a Docker daemon. There is nothing preventing us from adding full support other than the up-front engineering burden, but we'll know we're done when our customers are happy to use this in place of Docker.

This will enable building container images in environments that cannot run a Docker daemon, such as a Kubernetes cluster (where kernel privileges are unappealing). 


## Background {#background}

Currently, building a container image in a cluster can't be done safely with Docker because the Docker daemon requires privileged access. Despite no solution existing for arbitrary container images and arbitrary clusters, some workarounds exist today if users don't need full Dockerfile support or the ability to run in an arbitrary cluster with no extra privileges allowed.

<span style="color:#000000;">This Kubernetes [issue](https://github.com/kubernetes/kubernetes/issues/1806)<span style="color:#000000;"> outlines many problems and other use-cases.</span></span>


## Design Overview {#design-overview}

Kaniko is an open source tool to build container images from a Dockerfile in a Kubernetes cluster without using Docker. The user provides a Dockerfile, source context, and destination for the built image, and kaniko will build the image and push it to the desired location. This will be accomplished as follows:



1.  The user will provide a Dockerfile, source context and destination for the image via a command line tool or API
1.  The builder executable will run inside a container with access to the Dockerfile and source context supplied by the user (either as an Container Builder build step or Kubernetes Job)
1.  The builder executable will parse the Dockerfile, and extract the filesystem of the base image (the image in the FROM line of the Dockerfile) to root. It will then execute each command in the Dockerfile, snapshotting the filesystem after each one. Snapshots will be saved as tarballs, and then appended to the base image as layers to build the final image and push it to a registry.


## Detailed Design {#detailed-design}


### Builder executable {#builder-executable}

The builder executable is responsible for parsing the Dockerfile, executing the commands within the Dockerfile, and building the final image. It first extracts the base image filesystem to root (the base image is the image declared in the FROM line of the Dockerfile). Next, it parses the Dockerfile, and executes each command in the Dockerfile in sequence. After executing each command, the executable will snapshot the filesystem, storing any changes in a tarball, which will be a layer in the final image. Once every command has been executed, the executable will append the new layers to the base image, and push the new image to the specified destination. 


### Snapshotting {#snapshotting}

Snapshotting the filesystem can be done by checksumming every file in the image before command execution (including permission, mode and timestamp bits), then comparing these to the checksums after the command is executed. Files that have different checksums after (including files that have been deleted) need to be added to that layer's differential tarball.

This system mimics the behavior of `overlay` or `snapshotting` filesystems by moving the diffing operation into user-space. This will obviously result in lower performance than a real snapshotting filesystem, but some benchmarks show that this overhead is negligible when compared to the commands executed in a typical build. A snapshot of the entire Debian filesystem takes around .5s, unoptimized.

The tradeoff here is portability - a userspace snapshotter can execute in any storage driver or container runtime.

The following directories and files will be excluded from snapshotting. `workspace` is created by kaniko to store the builder executable and the Dockerfile. The other directories are injected by the Docker daemon or Kubernetes and can be ignored.



*   `/workspace`
*   `/dev`
*   `/sys`
*   `/proc`
*   `/var/run/secrets`
*   `/etc/hostname, /etc/hosts, /etc/mtab, /etc/resolv.conf`
*   `/.dockerenv`

These directories and files can be dynamically discovered via introspection of the /proc/self/mountinfo file, allowing the build to run in more environments without manual whitelisting of directories.


#### Whitelisting from /proc/self/mountinfo

Documentation for /proc/self/mountinfo can be found [here](https://www.kernel.org/doc/Documentation/filesystems/proc.txt), in section 3.5. This file contains information about mounted directories, which we want to ignore when snapshotting. Each line in the file is in the form:


```
36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
(1)(2)(3)  (4)   (5)   (6)        (7)     (8)(9)  (10)                    (11)
```


Where (5) is the mount point relative to the process's root. We can parse the file line by line, storing the relative mount point in a whitelist array as we go. 


#### Ptrace Optimizations

The above discussed snapshotting can be thought of as a "naive" approach, examining the contents of every file after every command. Several optimizations to this are possible, including using the [ptrace(2)](http://man7.org/linux/man-pages/man2/ptrace.2.html) syscall to monitor what files are actually manipulated by the command.

A rough proof of concept of this approach is available at github.com/dlorenc/ptracer.

It works as follows:



1.  Setup ptrace to intercept syscalls from specified RUN commands.
1.  Intercept syscalls that open file descriptors ([open(2), openat(2), creat(2)](http://man7.org/linux/man-pages/man2/open.2.html))
1.  Parse the syscall arguments out of the registers using PTRACE_GETREGS, PTRACE_PEEK_DATA/TEXT and the x86_64 syscall tables.
1.  Resolve the paths/FDs to filenames using the process's working directory/state

This can help limit the number of files that need to be examined for changes after each command.


### Dockerfile Commands {#dockerfile-commands}

There are approximately [18 supported Dockerfile commands](https://docs.docker.com/engine/reference/builder/#from), many of which only operate on container metadata. The full set of commands is outlined below, but only "interesting" ones are detailed more thoroughly.


<table>
  <tr>
   <td>Command
   </td>
   <td>Description
   </td>
  </tr>
  <tr>
   <td>FROM
   </td>
   <td>Used to unpack the initial (and subsequent base images for the mutli-step case).
   </td>
  </tr>
  <tr>
   <td>RUN
   </td>
   <td>The command will be run, with shell prepending, and the FS snapshotted after each command.
   </td>
  </tr>
  <tr>
   <td>CMD, ENTRYPOINT, LABEL, MAINTAINER, EXPOSE, VOLUME, STOPSIGNAL, HEALTHCHECK
   </td>
   <td>Metadata-only operations.
   </td>
  </tr>
  <tr>
   <td>ENV, WORKDIR, USER, SHELL
   </td>
   <td>Mostly metadata only - Variables must also be applied to the build environment at the proper time for use in subsequent RUN commands.
   </td>
  </tr>
  <tr>
   <td>ADD/COPY
   </td>
   <td>Files will be copied out of the build context and into the build environment. For multi-step builds, filesu will be saved from the previous build environment so they can be copied into later steps.
   </td>
  </tr>
  <tr>
   <td>ARG
   </td>
   <td>ARG variables will be supplied to the build executable and substituted in appropriately during parsing.
   </td>
  </tr>
</table>



### Source Context {#source-context}

The source context can be provided as a local directory, which will be uploaded to a GCS bucket.  Like [skaffold](https://github.com/GoogleContainerTools/skaffold), we could potentially parse the Dockerfile for dependencies and upload only the files we need to save time. Other possible source contexts include Git repositories or pre-packaged tarball contexts.


### Permissions in the Kubernetes cluster {#permissions-in-the-kubernetes-cluster}

The Kubernetes cluster will require permission to push images to the desired registry, which will be accomplished with Secrets. 


## User Experience {#user-experience}

Kaniko will initially be packaged in a few formats, detailed below:



*   An Container Builder build step container
*   A command line tool for use with Kubernetes clusters


### Container Builder Build Step Container {#Container Builder-build-step-container}

This container will surface an API as similar to the standard "docker build" step as possible, with changes required for the implicit "docker push". This container will expect the context to already be unpacked, and take an optional path to a Dockerfile. Authorization will be provided by Container Builder's spoofed metadata server, which provides the Container builder robot's credentials.

The surface will be roughly:


```
steps:
- name: 'gcr.io/some-project/kaniko
  args: ['build-and-push', '-t', 'NAME_OF_IMAGE', '-f', 'PATH_TO_DOCKERFILE']
# no images: stanza, image was pushed by kaniko
```


### Command Line Tool {#command-line-tool}


```
$ kaniko --dockerfile=<path to dockerfile> --context=<path to context>
         --name=<name to push to in registry>
```


The --context parameter will default to the directory of the specified dockerfile.

Using a command line tool, the user will provide a Dockerfile from which the resulting image will be built, a source context, and the full name of the registry the image should be pushed to. The command line tool will upload the source context to a GCS bucket, so that it can be accessed later on (for example, when carrying out ADD/COPY command in the Dockerfile). 


#### Kubernetes Job {#kubernetes-job}

The command line tool will then generate a Kubernetes `v1/batch Job` from the information the user has provided. The spec is shown below: 


```
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: kaniko-build
    spec:
      template:
        spec:
          containers:
          - name: init-static
            image: gcr.io/priya-wadhwa/kaniko:latest
            volumeMounts:
            - name: dockerfile
              mountPath: /dockerfile
            command: ["/work-dir/main"]
          restartPolicy: Never
          volumes:
          - name: dockerfile
    	 configMap:
            - name: build-dockerfile
              items:
    	     key: dockerfile
             data: "dockerfile contents" 
```


The Job specifies one container to run, called the init-static container. This container is based off of the distroless base image, and contains a Go builder executable and some additional files necessary for authentication. The entrypoint of the image is the builder executable, which allows the executable to examine the filesystem of the image and execute commands in this mount namespace. 


## Alternative Solutions {#alternative-solutions}


### Docker Socket Mounting

This solution involves volume mounting the host Docker socket into the pod, essentially giving the pod full root access on the machine. This gives containers in the Pod the ability to create, delete and modify other containers running on the same node. See below for an example configuration that uses hostPath to mount the Docker socket into a pod:


```
        apiVersion: v1
        kind: Pod
        metadata:
          name: docker
        spec:
          containers:
          - image: docker
            command: ["docker", "ps"]
            name: docker
            volumeMounts:
            - name: dockersock
              mountPath: /var/run/
          volumes:
          - name: dockersock
            hostPath:
              path: /var/run/
```


This is similar to how Cloud Build exposes a Docker daemon to users.

This is a security problem and a leaky abstraction - containers in a cluster should **not** know about the container runtime of the node they are running on. This would prevent a Pod from being able to run on a node backed by an alternate CRI implementation, or even a virtual kubelet.


### Docker-in-Docker {#docker-in-docker}

See [jpetazzo/dind](https://github.com/jpetazzo/dind) for instructions and a more thorough explanation of this approach. This approach runs a second Docker daemon inside a container, running inside the node's Docker daemon.

Docker and cgroups don't handle nesting particularly well, which can lead to bugs and strange behavior. This approach also requires the `--privileged` flag on the outer container. While this is supported in Kubernetes, it must be explicitly allowed by each node's configuration before a node will accept privileged containers.


### FTL/Bazel {#ftl-bazel}

These solutions aim to improve DevEx by achieving the fastest possible creation of Docker images, at the expense of build compatibility. By restricting the set of allowed builds to an optimizable subset, we get the nice side effect of being able to run without privileges inside an arbitrary cluster.

These approaches can be thought of as special-case "fast paths" that can be used in conjunction with the support for general Dockerfile builds outlined above.
