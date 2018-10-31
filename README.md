# kaniko - Build Images In Kubernetes

[![Build Status](https://travis-ci.org/GoogleContainerTools/kaniko.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/kaniko)

![kaniko logo](logo/Kaniko-Logo.png)

kaniko is a tool to build container images from a Dockerfile, inside a container or Kubernetes cluster.

kaniko doesn't depend on a Docker daemon and executes each command within a Dockerfile completely in userspace.
This enables building container images in environments that can't easily or securely run a Docker daemon, such as a standard Kubernetes cluster.

kaniko is meant to be run as an image, `gcr.io/kaniko-project/executor`.
We do **not** recommend running the kaniko executor binary in another image, as it might not work.

- [Kaniko](#kaniko)
  - [How does kaniko work?](#how-does-kaniko-work)
  - [Known Issues](#known-issues)
- [Demo](#demo)
- [Using kaniko](#using-kaniko)
  - [kaniko Build Contexts](#kaniko-build-contexts)
  - [Running kaniko](#running-kaniko)
    - [Running kaniko in a Kubernetes cluster](#running-kaniko-in-a-kubernetes-cluster)
    - [Running kaniko in gVisor](#running-kaniko-in-gvisor)
    - [Running kaniko in Google Cloud Build](#running-kaniko-in-google-cloud-build)
    - [Running kaniko in Docker](#running-kaniko-in-Docker)
  - [Caching](#caching)
  - [Pushing to Different Registries](#pushing-to-different-registries)
  - [Additional Flags](#additional-flags)
  - [Debug Image](#debug-image)
- [Security](#security)
- [Comparison with Other Tools](#comparison-with-other-tools)
- [Community](#community)
- [Limitations](#limitations)

_If you are interested in contributing to kaniko, see [DEVELOPMENT.md](DEVELOPMENT.md) and [CONTRIBUTING.md](CONTRIBUTING.md)._

### How does kaniko work?

The kaniko executor image is responsible for building an image from a Dockerfile and pushing it to a registry.
Within the executor image, we extract the filesystem of the base image (the FROM image in the Dockerfile).
We then execute the commands in the Dockerfile, snapshotting the filesystem in userspace after each one.
After each command, we append a layer of changed files to the base image (if there are any) and update image metadata.

### Known Issues
kaniko does not support building Windows containers.

## Demo

![Demo](/docs/demo.gif)

## Using kaniko

To use kaniko to build and push an image for you, you will need:

1. A [build context](#kaniko-build-contexts), aka something to build
2. A [running instance of kaniko](#running-kaniko)

### kaniko Build Contexts

kaniko's build context is very similar to the build context you would send your Docker daemon for an image build; it represents a directory containing a Dockerfile which kaniko will use to build your image.
For example, a `COPY` command in your Dockerfile should refer to a file in the build context.

You will need to store your build context in a place that kaniko can access. 
Right now, kaniko supports these storage solutions:
- GCS Bucket
- S3 Bucket
- Local Directory

_Note: the local directory option refers to a directory within the kaniko container.
If you wish to use this option, you will need to mount in your build context into the container as a directory._

If using a GCS or S3 bucket, you will first need to create a compressed tar of your build context and upload it to your bucket. 
Once running, kaniko will then download and unpack the compressed tar of the build context before starting the image build.

To create a compressed tar, you can run:
```shell
tar -C <path to build context> -zcvf context.tar.gz .
```
Then, copy over the compressed tar into your bucket. 
For example, we can copy over the compressed tar to a GCS bucket with gsutil:
```
gsutil cp context.tar.gz gs://<bucket name>
```

When running kaniko, use the `--context` flag with the appropriate prefix to specify the location of your build context:

|  Source | Prefix  |
|---------|---------|
| Local Directory  | dir://[path to a directory in the kaniko container]  |
| GCS Bucket       | gs://[bucket name]/[path to .tar.gz]     | 
| S3 Bucket        | s3://[bucket name]/[path to .tar.gz]     |

If you don't specify a prefix, kaniko will assume a local directory.
For example, to use a GCS bucket called `kaniko-bucket`, you would pass in `--context=gs://kaniko-bucket/path/to/context.tar.gz`. 

### Running kaniko

There are several different ways to deploy and run kaniko:

- [In a Kubernetes cluster](#running-kaniko-in-a-kubernetes-cluster)
- [In gVisor](#running-kaniko-in-gvisor)
- [In Google Cloud Build](#running-kaniko-in-google-cloud-build)
- [In Docker](#running-kaniko-in-docker)

#### Running kaniko in a Kubernetes cluster

Requirements:

- Standard Kubernetes cluster (e.g. using [GKE](https://cloud.google.com/kubernetes-engine/))
- [Kubernetes Secret](#kubernetes-secret)
- A [build context](#kaniko-build-contexts)

##### Kubernetes secret

To run kaniko in a Kubernetes cluster, you will need a standard running Kubernetes cluster and a Kubernetes secret, which contains the auth required to push the final image.

To create a secret to authenticate to Google Cloud Registry, follow these steps:
1. Create a service account in the Google Cloud Console project you want to push the final image to with `Storage Admin` permissions.
2. Download a JSON key for this service account
3. Rename the key to `kaniko-secret.json`
4. To create the secret, run:

```shell
kubectl create secret generic kaniko-secret --from-file=<path to kaniko-secret.json>
```

_Note: If using a GCS bucket in the same GCP project as a build context, this service account should now also have permissions to read from that bucket._ 

The Kubernetes Pod spec should look similar to this, with the args parameters filled in:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kaniko
spec:
  containers:
  - name: kaniko
    image: gcr.io/kaniko-project/executor:latest
    args: ["--dockerfile=<path to Dockerfile within the build context>",
            "--context=gs://<GCS bucket>/<path to .tar.gz>",
            "--destination=<gcr.io/$PROJECT/$IMAGE:$TAG>"]
    volumeMounts:
      - name: kaniko-secret
        mountPath: /secret
    env:
      - name: GOOGLE_APPLICATION_CREDENTIALS
        value: /secret/kaniko-secret.json
  restartPolicy: Never
  volumes:
    - name: kaniko-secret
      secret:
        secretName: kaniko-secret
```

This example pulls the build context from a GCS bucket.
To use a local directory build context, you could consider using configMaps to mount in small build contexts.

#### Running kaniko in gVisor

Running kaniko in [gVisor](https://github.com/google/gvisor) provides an additional security boundary.
You will need to add the `--force` flag to run kaniko in gVisor, since currently there isn't a way to determine whether or not a container is running in gVisor.

```shell
docker run --runtime=runsc -v $(pwd):/workspace -v ~/.config:/root/.config \
gcr.io/kaniko-project/executor:latest \
--dockerfile=<path to Dockerfile> --context=/workspace \
--destination=gcr.io/my-repo/my-image --force
```

We pass in `--runtime=runsc` to use gVisor.
This example mounts the current directory to `/workspace` for the build context and the `~/.config` directory for GCR credentials.

#### Running kaniko in Google Cloud Build

Requirements:
- A [build context](#kaniko-build-contexts)

To run kaniko in GCB, add it to your build config as a build step:

```yaml
steps:
  - name: gcr.io/kaniko-project/executor:latest
    args: ["--dockerfile=<path to Dockerfile within the build context>",
           "--context=dir://<path to build context>",
           "--destination=<gcr.io/$PROJECT/$IMAGE:$TAG>"]
```

kaniko will build and push the final image in this build step.

#### Running kaniko in Docker

Requirements:

- [Docker](https://docs.docker.com/install/)
- [gcloud](https://cloud.google.com/sdk/install)

We can run the kaniko executor image locally in a Docker daemon to build and push an image from a Dockerfile.

1. Load the executor image into the Docker daemon by running:

  ```shell
  make images
  ```

2. Run kaniko in Docker using [`run_in_docker.sh`](./run_in_docker.sh):

  ```shell
  ./run_in_docker.sh <path to Dockerfile> <path to build context> <destination of final image>
  ```

### Caching

#### Caching Layers
kaniko currently can cache layers created by `RUN` commands in a remote repository. 
Before executing a command, kaniko checks the cache for the layer.
If it exists, kaniko will pull and extract the cached layer instead of executing the command.
If not, kaniko will execute the command and then push the newly created layer to the cache.

Users can opt in to caching by setting the `--cache=true` flag.
A remote repository for storing cached layers can be provided via the `--cache-repo` flag.
If this flag isn't provided, a cached repo will be inferred from the `--destination` provided.  

#### Caching Base Images
kaniko can cache images in a local directory that can be volume mounted into the kaniko image. 
To do so, the cache must first be populated, as it is read-only. We provide a kaniko cache warming 
image at `gcr.io/kaniko-project/warmer`:

```shell
docker run -v $(pwd):/workspace gcr.io/kaniko-project/warmer:latest --cache-dir=/workspace/cache --image=<image to cache> --image=<another image to cache>
```

`--image` can be specified for any number of desired images. 
This command will cache those images by digest in a local directory named `cache`.
Once the cache is populated, caching is opted into with the same `--cache=true` flag as above. 
The location of the local cache is provided via the `--cache-dir` flag, defaulting at `/cache` as with the cache warmer. 
See the `examples` directory for how to use with kubernetes clusters and persistent cache volumes.

### Pushing to Different Registries

kaniko uses Docker credential helpers to push images to a registry.

kaniko comes with support for GCR and Amazon ECR, but configuring another credential helper should allow pushing to a different registry.

#### Pushing to Amazon ECR

The Amazon ECR [credential helper](https://github.com/awslabs/amazon-ecr-credential-helper) is built in to the kaniko executor image.
To configure credentials, you will need to do the following:

1. Update the `credHelpers` section of [config.json](https://github.com/GoogleContainerTools/kaniko/blob/master/files/config.json) with the specific URI of your ECR registry:

  ```json
  {
    "credHelpers": {
      "aws_account_id.dkr.ecr.region.amazonaws.com": "ecr-login"
    }
  }
  ```

  You can mount in the new config as a configMap:

  ```shell
  kubectl create configmap docker-config --from-file=<path to config.json>
  ```

2. Create a Kubernetes secret for your `~/.aws/credentials` file so that credentials can be accessed within the cluster.

  To create the secret, run:

  ```shell
  kubectl create secret generic aws-secret --from-file=<path to .aws/credentials>
  ```

  The Kubernetes Pod spec should look similar to this, with the args parameters filled in:

  ```yaml
  apiVersion: v1
  kind: Pod
  metadata:
    name: kaniko
  spec:
    containers:
    - name: kaniko
      image: gcr.io/kaniko-project/executor:latest
      args: ["--dockerfile=<path to Dockerfile within the build context>",
              "--context=s3://<bucket name>/<path to .tar.gz>",
              "--destination=<aws_account_id.dkr.ecr.region.amazonaws.com/my-repository:my-tag>"]
      volumeMounts:
        - name: aws-secret
          mountPath: /root/.aws/
        - name: docker-config
          mountPath: /kaniko/.docker/
    restartPolicy: Never
    volumes:
      - name: aws-secret
        secret:
          secretName: aws-secret
      - name: docker-config
        configMap:
          name: docker-config
  ```

### Additional Flags

#### --snapshotMode

You can set the `--snapshotMode=<full (default), time>` flag to set how kaniko will snapshot the filesystem.
If `--snapshotMode=time` is set, only file mtime will be considered when snapshotting (see
[limitations related to mtime](#mtime-and-snapshotting)).

#### --build-arg

This flag allows you to pass in ARG values at build time, similarly to Docker.
You can set it multiple times for multiple arguments.

#### --single-snapshot

This flag takes a single snapshot of the filesystem at the end of the build, so only one layer will be appended to the base image.

#### --reproducible

Set this flag to strip timestamps out of the built image and make it reproducible.

#### --tarPath

Set this flag as `--tarPath=<path>` to save the image as a tarball at path instead of pushing the image.

#### --target

Set this flag to indicate which build stage is the target build stage.

#### --no-push

Set this flag if you only want to build the image, without pushing to a registry.

#### --insecure

Set this flag if you want to push images to a plain HTTP registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --skip-tls-verify

Set this flag to skip TLS certificate validation when pushing images to a registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --insecure-pull

Set this flag if you want to pull images from a plain HTTP registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --skip-tls-verify-pull

Set this flag to skip TLS certificate validation when pulling images from a registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --cache

Set this flag as `--cache=true` to opt in to caching with kaniko.

#### --cache-repo

Set this flag to specify a remote repository which will be used to store cached layers.

If this flag is not provided, a cache repo will be inferred from the `--destination` flag.
If `--destination=gcr.io/kaniko-project/test`, then cached layers will be stored in `gcr.io/kaniko-project/test/cache`.

_This flag must be used in conjunction with the `--cache=true` flag._

#### --cache-dir

Set this flag to specify a local directory cache for base images. Defaults to `/cache`.

_This flag must be used in conjunction with the `--cache=true` flag._

#### --cleanup

Set this flag to cleanup the filesystem at the end, leaving a clean kaniko container (if you want to build multiple images in the same container, using the debug kaniko image)

### Debug Image

The kaniko executor image is based off of scratch and doesn't contain a shell.
We provide `gcr.io/kaniko-project/executor:debug`, a debug image which consists of the kaniko executor image along with a busybox shell to enter.

You can launch the debug image with a shell entrypoint:

```shell
docker run -it --entrypoint=/busybox/sh gcr.io/kaniko-project/executor:debug
```

## Security

kaniko by itself **does not** make it safe to run untrusted builds inside your cluster, or anywhere else.

kaniko relies on the security features of your container runtime to provide build security.

The minimum permissions kaniko needs inside your container are governed by a few things:

* The permissions required to unpack your base image into it's container
* The permissions required to execute the RUN commands inside the container

If you have a minimal base image (SCRATCH or similar) that doesn't require
permissions to unpack, and your Dockerfile doesn't execute any commands as the
root user, you can run Kaniko without root permissions. It should be noted that
Docker runs as root by default, so you still require (in a sense) privileges to
use Kaniko.

You may be able to achieve the same default seccomp profile that Docker uses in your Pod by setting [seccomp](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#seccomp) profiles with annotations on a [PodSecurityPolicy](https://cloud.google.com/kubernetes-engine/docs/how-to/pod-security-policies) to create or update security policies on your cluster.

## Comparison with Other Tools

Similar tools include:

- [img](https://github.com/genuinetools/img)
- [orca-build](https://github.com/cyphar/orca-build)
- [umoci](https://github.com/openSUSE/umoci)
- [buildah](https://github.com/projectatomic/buildah)
- [FTL](https://github.com/GoogleCloudPlatform/runtimes-common/tree/master/ftl)
- [Bazel rules_docker](https://github.com/bazelbuild/rules_docker)

All of these tools build container images with different approaches.

`img` can perform as a non root user from within a container, but requires that
the `img` container has `RawProc` access to create nested containers.  `kaniko`
does not actually create nested containers, so it does not require `RawProc`
access.

`orca-build` depends on `runc` to build images from Dockerfiles, which can not
run inside a container (for similar reasons to `img` above). `kaniko` doesn't
use `runc` so it doesn't require the use of kernel namespacing techniques.
However, `orca-build` does not require Docker or any privileged daemon (so
builds can be done entirely without privilege).

`umoci` works without any privileges, and also has no restrictions on the root
filesystem being extracted (though it requires additional handling if your
filesystem is sufficiently complicated). However it has no `Dockerfile`-like
build tooling (it's a slightly lower-level tool that can be used to build such
builders -- such as `orca-build`).

`buildah` requires the same privileges as a Docker daemon does to run, while
`kaniko` runs without any special privileges or permissions.

`FTL` and `Bazel` aim to achieve the fastest possible creation of Docker images
for a subset of images.  These can be thought of as a special-case "fast path"
that can be used in conjunction with the support for general Dockerfiles kaniko
provides.

## Community

[kaniko-users](https://groups.google.com/forum/#!forum/kaniko-users) Google group

To Contribute to kaniko, see [DEVELOPMENT.md](DEVELOPMENT.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

## Limitations

### mtime and snapshotting

When taking a snapshot, kaniko's hashing algorithms include (or in the case of
[`--snapshotMode=time`](#--snapshotmode), only use) a file's
[`mtime`](https://en.wikipedia.org/wiki/Inode#POSIX_inode_description) to determine
if the file has changed. Unfortunately there is a delay between when changes to a
file are made and when the `mtime` is updated. This means:

* With the time-only snapshot mode (`--snapshotMode=time`), kaniko may miss changes
  introduced by `RUN` commands entirely.
* With the default snapshot mode (`--snapshotMode=full`), whether or not kaniko will
  add a layer in the case where a `RUN` command modifies a file **but the contents do
  not** change is theoretically non-deterministic. This _does not affect the contents_
  which will still be correct, but it does affect the number of layers.

_Note that these issues are currently theoretical only. If you see this issue occur, please
[open an issue](https://github.com/GoogleContainerTools/kaniko/issues)._
