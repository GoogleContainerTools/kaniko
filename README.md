# kaniko - Build Images In Kubernetes

`NOTE: Kaniko is not an officially supported Google product`

[![Build Status](https://travis-ci.org/GoogleContainerTools/kaniko.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/kaniko) [![Go Report Card](https://goreportcard.com/badge/github.com/GoogleContainerTools/kaniko)](https://goreportcard.com/report/github.com/GoogleContainerTools/kaniko)

![kaniko logo](logo/Kaniko-Logo.png)

kaniko is a tool to build container images from a Dockerfile, inside a container or Kubernetes cluster.

kaniko doesn't depend on a Docker daemon and executes each command within a Dockerfile completely in userspace.
This enables building container images in environments that can't easily or securely run a Docker daemon, such as a standard Kubernetes cluster.

kaniko is meant to be run as an image: `gcr.io/kaniko-project/executor`. We do **not** recommend running the kaniko executor binary in another image, as it might not work.

We'd love to hear from you!  Join us on [#kaniko Kubernetes Slack](https://kubernetes.slack.com/messages/CQDCHGX7Y/)

:mega: **Please fill out our [quick 5-question survey](https://forms.gle/HhZGEM33x4FUz9Qa6)** so that we can learn how satisfied you are with Kaniko, and what improvements we should make. Thank you! :dancers:


_If you are interested in contributing to kaniko, see [DEVELOPMENT.md](DEVELOPMENT.md) and [CONTRIBUTING.md](CONTRIBUTING.md)._


<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Community](#community)
- [How does kaniko work?](#how-does-kaniko-work)
- [Known Issues](#known-issues)
- [Demo](#demo)
- [Tutorial](#tutorial)
- [Using kaniko](#using-kaniko)
  - [kaniko Build Contexts](#kaniko-build-contexts)
  - [Using Azure Blob Storage](#using-azure-blob-storage)
  - [Using Private Git Repository](#using-private-git-repository)
  - [Running kaniko](#running-kaniko)
    - [Running kaniko in a Kubernetes cluster](#running-kaniko-in-a-kubernetes-cluster)
      - [Kubernetes secret](#kubernetes-secret)
    - [Running kaniko in gVisor](#running-kaniko-in-gvisor)
    - [Running kaniko in Google Cloud Build](#running-kaniko-in-google-cloud-build)
    - [Running kaniko in Docker](#running-kaniko-in-docker)
  - [Caching](#caching)
    - [Caching Layers](#caching-layers)
    - [Caching Base Images](#caching-base-images)
  - [Pushing to Different Registries](#pushing-to-different-registries)
    - [Pushing to Docker Hub](#pushing-to-docker-hub)
    - [Pushing to Amazon ECR](#pushing-to-amazon-ecr)
  - [Additional Flags](#additional-flags)
    - [--build-arg](#--build-arg)
    - [--cache](#--cache)
    - [--cache-dir](#--cache-dir)
    - [--cache-repo](#--cache-repo)
    - [--context-sub-path](#context-sub-path)
    - [--digest-file](#--digest-file)
    - [--oci-layout-path](#--oci-layout-path)
    - [--insecure-registry](#--insecure-registry)
    - [--skip-tls-verify-registry](#--skip-tls-verify-registry)
    - [--cleanup](#--cleanup)
    - [--insecure](#--insecure)
    - [--insecure-pull](#--insecure-pull)
    - [--log-format](#--log-format)
    - [--log-timestamp](#--log-timestamp)
    - [--no-push](#--no-push)
    - [--registry-certificate](#--registry-certificate)
    - [--registry-mirror](#--registry-mirror)
    - [--reproducible](#--reproducible)
    - [--single-snapshot](#--single-snapshot)
    - [--skip-tls-verify](#--skip-tls-verify)
    - [--skip-tls-verify-pull](#--skip-tls-verify-pull)
    - [--snapshotMode](#--snapshotmode)
    - [--target](#--target)
    - [--tarPath](#--tarpath)
    - [--verbosity](#--verbosity)
    - [--whitelist-var-run](#--whitelist-var-run)
    - [--label](#--label)
    - [--skip-unused-stages](#skip-unused-stages)
  - [Debug Image](#debug-image)
- [Security](#security)
- [Comparison with Other Tools](#comparison-with-other-tools)
- [Community](#community-1)
- [Limitations](#limitations)
  - [mtime and snapshotting](#mtime-and-snapshotting)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Community
We'd love to hear from you! Join [#kaniko on Kubernetes Slack](https://kubernetes.slack.com/messages/CQDCHGX7Y/)

## How does kaniko work?

The kaniko executor image is responsible for building an image from a Dockerfile and pushing it to a registry.
Within the executor image, we extract the filesystem of the base image (the FROM image in the Dockerfile).
We then execute the commands in the Dockerfile, snapshotting the filesystem in userspace after each one.
After each command, we append a layer of changed files to the base image (if there are any) and update image metadata.

## Known Issues

* kaniko does not support building Windows containers.
* Running kaniko in any Docker image other than the official kaniko image is not supported (ie YMMV).
  * This includes copying the kaniko executables from the official image into another image.
* kaniko does not support the v1 Registry API ([Registry v1 API Deprecation](https://engineering.docker.com/2019/03/registry-v1-api-deprecation/))

## Demo

![Demo](/docs/demo.gif)

## Tutorial

For a detailed example of kaniko with local storage, please refer to a [getting started tutorial](./docs/tutorial.md).

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
- Azure Blob Storage
- Local Directory
- Local Tar
- Standard Input
- Git Repository

_Note about Local Directory: this option refers to a directory within the kaniko container.
If you wish to use this option, you will need to mount in your build context into the container as a directory._

_Note about Local Tar: this option refers to a tar gz  file within the kaniko container.
If you wish to use this option, you will need to mount in your build context into the container as a file._

_Note about Standard Input: the only Standard Input allowed by kaniko is in `.tar.gz` format._

If using a GCS or S3 bucket, you will first need to create a compressed tar of your build context and upload it to your bucket.
Once running, kaniko will then download and unpack the compressed tar of the build context before starting the image build.

To create a compressed tar, you can run:

```shell
tar -C <path to build context> -zcvf context.tar.gz .
```
Then, copy over the compressed tar into your bucket.
For example, we can copy over the compressed tar to a GCS bucket with gsutil:

```shell
gsutil cp context.tar.gz gs://<bucket name>
```

When running kaniko, use the `--context` flag with the appropriate prefix to specify the location of your build context:

|  Source | Prefix  | Example |
|---------|---------|---------|
| Local Directory   | dir://[path to a directory in the kaniko container]             | `dir:///workspace`                                            |
| Local Tar Gz      | tar://[path to a .tar.gz in the kaniko container]               | `tar://path/to/context.tar.gz`                                            |
| Standard Input    | tar://[stdin]                                                   | `tar://stdin`                                                 |
| GCS Bucket        | gs://[bucket name]/[path to .tar.gz]                            | `gs://kaniko-bucket/path/to/context.tar.gz`                   |
| S3 Bucket         | s3://[bucket name]/[path to .tar.gz]                            | `s3://kaniko-bucket/path/to/context.tar.gz`                   |
| Azure Blob Storage| https://[account].[azureblobhostsuffix]/[container]/[path to .tar.gz] | `https://myaccount.blob.core.windows.net/container/path/to/context.tar.gz` |
| Git Repository    | git://[repository url][#reference]                              | `git://github.com/acme/myproject.git#refs/heads/mybranch`     |

If you don't specify a prefix, kaniko will assume a local directory.
For example, to use a GCS bucket called `kaniko-bucket`, you would pass in `--context=gs://kaniko-bucket/path/to/context.tar.gz`.

### Using Azure Blob Storage
If you are using Azure Blob Storage for context file, you will need to pass [Azure Storage Account Access Key](https://docs.microsoft.com/en-us/azure/storage/common/storage-configure-connection-string?toc=%2fazure%2fstorage%2fblobs%2ftoc.json) as an environment variable named `AZURE_STORAGE_ACCESS_KEY` through Kubernetes Secrets

### Using Private Git Repository
You can use `Personal Access Tokens` for Build Contexts from Private Repositories from [GitHub](https://blog.github.com/2012-09-21-easier-builds-and-deployments-using-git-over-https-and-oauth/).

You can either pass this in as part of the git URL (e.g., `git://TOKEN@github.com/acme/myproject.git#refs/heads/mybranch`)
or using the environment variable `GIT_USERNAME`.

### Using Standard Input
If running kaniko and using Standard Input build context, you will need to add the docker or kubernetes `-i, --interactive` flag.
Once running, kaniko will then get the data from `STDIN` and create the build context as a compressed tar.
It will then unpack the compressed tar of the build context before starting the image build.
If no data is piped during the interactive run, you will need to send the EOF signal by yourself by pressing `Ctrl+D`.

Complete example of how to interactively run kaniko with `.tar.gz` Standard Input data, using docker:
```shell
echo -e 'FROM alpine \nRUN echo "created from standard input"' > Dockerfile | tar -cf - Dockerfile | gzip -9 | docker run \
  --interactive -v $(pwd):/workspace gcr.io/kaniko-project/executor:latest \
  --context tar://stdin \
  --destination=<gcr.io/$project/$image:$tag>
```

Complete example of how to interactively run kaniko with `.tar.gz` Standard Input data, using Kubernetes command line with a temporary container and completely dockerless:
```shell
echo -e 'FROM alpine \nRUN echo "created from standard input"' > Dockerfile | tar -cf - Dockerfile | gzip -9 | kubectl run kaniko \
--rm --stdin=true \
--image=gcr.io/kaniko-project/executor:latest --restart=Never \
--overrides='{
  "apiVersion": "v1",
  "spec": {
    "containers": [
      {
        "name": "kaniko",
        "image": "gcr.io/kaniko-project/executor:latest",
        "stdin": true,
        "stdinOnce": true,
        "args": [
          "--dockerfile=Dockerfile",
          "--context=tar://stdin",
          "--destination=gcr.io/my-repo/my-image"
        ],
        "volumeMounts": [
          {
            "name": "cabundle",
            "mountPath": "/kaniko/ssl/certs/"
          },
          {
            "name": "docker-config",
            "mountPath": "/kaniko/.docker/"
          }
        ]
      }
    ],
    "volumes": [
      {
        "name": "cabundle",
        "configMap": {
          "name": "cabundle"
        }
      },
      {
        "name": "docker-config",
        "configMap": {
          "name": "docker-config"
        }
      }
    ]
  }
}'
```

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
    args:
    - "--dockerfile=<path to Dockerfile within the build context>"
    - "--context=gs://<GCS bucket>/<path to .tar.gz>"
    - "--destination=<gcr.io/$PROJECT/$IMAGE:$TAG>"
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

We can run the kaniko executor image locally in a Docker daemon to build and push an image from a Dockerfile.

For example, when using gcloud and GCR you could run Kaniko as follows:
```shell
docker run \
    -v "$HOME"/.config/gcloud:/root/.config/gcloud \
    -v /path/to/context:/workspace \
    gcr.io/kaniko-project/executor:latest \
    --dockerfile /workspace/Dockerfile \
    --destination "gcr.io/$PROJECT_ID/$IMAGE_NAME:$TAG" \
    --context dir:///workspace/
```

There is also a utility script [`run_in_docker.sh`](./run_in_docker.sh) that can be used as follows:
```shell
./run_in_docker.sh <path to Dockerfile> <path to build context> <destination of final image>
```

_NOTE: `run_in_docker.sh` expects a path to a
Dockerfile relative to the absolute path of the build context._

An example run, specifying the Dockerfile in the container directory `/workspace`, the build
context in the local directory `/home/user/kaniko-project`, and a Google Container Registry
as a remote image destination:

```shell
./run_in_docker.sh /workspace/Dockerfile /home/user/kaniko-project gcr.io/$PROJECT_ID/$TAG
```

### Caching

#### Caching Layers
kaniko can cache layers created by `RUN` commands in a remote repository.
Before executing a command, kaniko checks the cache for the layer.
If it exists, kaniko will pull and extract the cached layer instead of executing the command.
If not, kaniko will execute the command and then push the newly created layer to the cache.

Users can opt into caching by setting the `--cache=true` flag.
A remote repository for storing cached layers can be provided via the `--cache-repo` flag.
If this flag isn't provided, a cached repo will be inferred from the `--destination` provided.

#### Caching Base Images

kaniko can cache images in a local directory that can be volume mounted into the kaniko pod.
To do so, the cache must first be populated, as it is read-only. We provide a kaniko cache warming
image at `gcr.io/kaniko-project/warmer`:

```shell
docker run -v $(pwd):/workspace gcr.io/kaniko-project/warmer:latest --cache-dir=/workspace/cache --image=<image to cache> --image=<another image to cache>
```

`--image` can be specified for any number of desired images.
This command will cache those images by digest in a local directory named `cache`.
Once the cache is populated, caching is opted into with the same `--cache=true` flag as above.
The location of the local cache is provided via the `--cache-dir` flag, defaulting to `/cache` as with the cache warmer.
See the `examples` directory for how to use with kubernetes clusters and persistent cache volumes.

### Pushing to Different Registries

kaniko uses Docker credential helpers to push images to a registry.

kaniko comes with support for GCR, Docker `config.json` and Amazon ECR, but configuring another credential helper should allow pushing to a different registry.

#### Pushing to Docker Hub

Get your docker registry user and password encoded in base64

    echo -n USER:PASSWORD | base64

Create a `config.json` file with your Docker registry url and the previous generated base64 string

```
{
	"auths": {
		"https://index.docker.io/v2/": {
			"auth": "xxxxxxxxxxxxxxx"
		}
	}
}
```

Run kaniko with the `config.json` inside `/kaniko/.docker/config.json`

    docker run -ti --rm -v `pwd`:/workspace -v `pwd`/config.json:/kaniko/.docker/config.json:ro gcr.io/kaniko-project/executor:latest --dockerfile=Dockerfile --destination=yourimagename

#### Pushing to Amazon ECR

The Amazon ECR [credential helper](https://github.com/awslabs/amazon-ecr-credential-helper) is built into the kaniko executor image.
To configure credentials, you will need to do the following:

1. Update the `credsStore` section of [config.json](https://github.com/awslabs/amazon-ecr-credential-helper#configuration):

  ```json
  { "credsStore": "ecr-login" }
  ```

  You can mount in the new config as a configMap:

  ```shell
  kubectl create configmap docker-config --from-file=<path to config.json>
  ```

2. Configure credentials

    1. You can use instance roles when pushing to ECR from a EC2 instance or from EKS, by [configuring the instance role permissions](https://docs.aws.amazon.com/AmazonECR/latest/userguide/ECR_on_EKS.html).

    2. Or you can create a Kubernetes secret for your `~/.aws/credentials` file so that credentials can be accessed within the cluster.
    To create the secret, run:
        ```shell
        kubectl create secret generic aws-secret --from-file=<path to .aws/credentials>
        ```

The Kubernetes Pod spec should look similar to this, with the args parameters filled in.
Note that `aws-secret` volume mount and volume are only needed when using AWS credentials from a secret, not when using instance roles.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kaniko
spec:
  containers:
  - name: kaniko
    image: gcr.io/kaniko-project/executor:latest
    args:
    - "--dockerfile=<path to Dockerfile within the build context>"
    - "--context=s3://<bucket name>/<path to .tar.gz>"
    - "--destination=<aws_account_id.dkr.ecr.region.amazonaws.com/my-repository:my-tag>"
    volumeMounts:
    - name: docker-config
      mountPath: /kaniko/.docker/
    # when not using instance role
    - name: aws-secret
      mountPath: /root/.aws/
  restartPolicy: Never
  volumes:
  - name: docker-config
    configMap:
      name: docker-config
  # when not using instance role
  - name: aws-secret
    secret:
      secretName: aws-secret
```

### Additional Flags

#### --build-arg

This flag allows you to pass in ARG values at build time, similarly to Docker.
You can set it multiple times for multiple arguments.

#### --cache

Set this flag as `--cache=true` to opt into caching with kaniko.

#### --cache-dir

Set this flag to specify a local directory cache for base images. Defaults to `/cache`.

_This flag must be used in conjunction with the `--cache=true` flag._

#### --cache-repo

Set this flag to specify a remote repository that will be used to store cached layers.

If this flag is not provided, a cache repo will be inferred from the `--destination` flag.
If `--destination=gcr.io/kaniko-project/test`, then cached layers will be stored in `gcr.io/kaniko-project/test/cache`.

_This flag must be used in conjunction with the `--cache=true` flag._

#### --context-sub-path

Set a sub path within the given `--context`.

Its particularly useful when your context is, for example, a git repository,
and you want to build one of its subfolders instead of the root folder.

#### --digest-file

Set this flag to specify a file in the container. This file will
receive the digest of a built image. This can be used to
automatically track the exact image built by Kaniko.

For example, setting the flag to `--digest-file=/dev/termination-log`
will write the digest to that file, which is picked up by
Kubernetes automatically as the `{{.state.terminated.message}}`
of the container.

#### --oci-layout-path

Set this flag to specify a directory in the container where the OCI image
layout of a built image will be placed. This can be used to automatically
track the exact image built by Kaniko.

For example, to surface the image digest built in a
[Tekton task](https://github.com/tektoncd/pipeline/blob/v0.6.0/docs/resources.md#surfacing-the-image-digest-built-in-a-task),
this flag should be set to match the image resource `outputImageDir`.

_Note: Depending on the built image, the media type of the image manifest might be either
`application/vnd.oci.image.manifest.v1+json` or `application/vnd.docker.distribution.manifest.v2+json`._

#### --insecure-registry

Set this flag to use plain HTTP requests when accessing a registry. It is supposed to be used for testing purposes only and should not be used in production!
You can set it multiple times for multiple registries.

#### --skip-tls-verify-registry

Set this flag to skip TLS certificate validation when accessing a registry. It is supposed to be used for testing purposes only and should not be used in production!
You can set it multiple times for multiple registries.

#### --cleanup

Set this flag to clean the filesystem at the end of the build.

#### --insecure

Set this flag if you want to push images to a plain HTTP registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --insecure-pull

Set this flag if you want to pull images from a plain HTTP registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --no-push

Set this flag if you only want to build the image, without pushing to a registry.

#### --registry-certificate

Set this flag to provide a certificate for TLS communication with a given registry.

Expected format is `my.registry.url=/path/to/the/certificate.cert`

#### --registry-mirror

Set this flag if you want to use a registry mirror instead of default `index.docker.io`.

#### --reproducible

Set this flag to strip timestamps out of the built image and make it reproducible.

#### --single-snapshot

This flag takes a single snapshot of the filesystem at the end of the build, so only one layer will be appended to the base image.

#### --skip-tls-verify

Set this flag to skip TLS certificate validation when pushing to a registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --skip-tls-verify-pull

Set this flag to skip TLS certificate validation when pulling from a registry. It is supposed to be used for testing purposes only and should not be used in production!

#### --snapshotMode

You can set the `--snapshotMode=<full (default), time>` flag to set how kaniko will snapshot the filesystem.
If `--snapshotMode=time` is set, only file mtime will be considered when snapshotting (see
[limitations related to mtime](#mtime-and-snapshotting)).

#### --target

Set this flag to indicate which build stage is the target build stage.

#### --tarPath

Set this flag as `--tarPath=<path>` to save the image as a tarball at path instead of pushing the image.
You need to set `--destination` as well (for example `--destination=image`).

#### --verbosity

Set this flag as `--verbosity=<panic|fatal|error|warn|info|debug|trace>` to set the logging level. Defaults to `info`.

#### --log-format

Set this flag as `--log-format=<text|color|json>` to set the log format. Defaults to `color`.

#### --log-timestamp

Set this flag as `--log-timestamp=<true|false>` to add timestamps to `<text|color>` log format. Defaults to `false`.

#### --whitelist-var-run

Ignore /var/run when taking image snapshot. Set it to false to preserve /var/run/* in destination image. (Default true).

#### --label

Set this flag as `--label key=value` to set some metadata to the final image. This is equivalent as using the `LABEL` within the Dockerfile.

#### --skip-unused-stages

This flag builds only used stages if defined to `true`.
Otherwise it builds by default all stages, even the unnecessaries ones until it reaches the target stage / end of Dockerfile

### Debug Image

The kaniko executor image is based on scratch and doesn't contain a shell.
We provide `gcr.io/kaniko-project/executor:debug`, a debug image which consists of the kaniko executor image along with a busybox shell to enter.

You can launch the debug image with a shell entrypoint:

```shell
docker run -it --entrypoint=/busybox/sh gcr.io/kaniko-project/executor:debug
```

## Security

kaniko by itself **does not** make it safe to run untrusted builds inside your cluster, or anywhere else.

kaniko relies on the security features of your container runtime to provide build security.

The minimum permissions kaniko needs inside your container are governed by a few things:

* The permissions required to unpack your base image into its container
* The permissions required to execute the RUN commands inside the container

If you have a minimal base image (SCRATCH or similar) that doesn't require
permissions to unpack, and your Dockerfile doesn't execute any commands as the
root user, you can run Kaniko without root permissions. It should be noted that
Docker runs as root by default, so you still require (in a sense) privileges to
use Kaniko.

You may be able to achieve the same default seccomp profile that Docker uses in your Pod by setting [seccomp](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#seccomp) profiles with annotations on a [PodSecurityPolicy](https://cloud.google.com/kubernetes-engine/docs/how-to/pod-security-policies) to create or update security policies on your cluster.

## Comparison with Other Tools

Similar tools include:

- [BuildKit](https://github.com/moby/buildkit)
- [img](https://github.com/genuinetools/img)
- [orca-build](https://github.com/cyphar/orca-build)
- [umoci](https://github.com/openSUSE/umoci)
- [buildah](https://github.com/containers/buildah)
- [FTL](https://github.com/GoogleCloudPlatform/runtimes-common/tree/master/ftl)
- [Bazel rules_docker](https://github.com/bazelbuild/rules_docker)

All of these tools build container images with different approaches.

BuildKit (and `img`) can perform as a non-root user from within a container but requires
seccomp and AppArmor to be disabled to create nested containers.  `kaniko`
does not actually create nested containers, so it does not require seccomp and AppArmor
to be disabled.

`orca-build` depends on `runc` to build images from Dockerfiles, which can not
run inside a container (for similar reasons to `img` above). `kaniko` doesn't
use `runc` so it doesn't require the use of kernel namespacing techniques.
However, `orca-build` does not require Docker or any privileged daemon (so
builds can be done entirely without privilege).

`umoci` works without any privileges, and also has no restrictions on the root
filesystem being extracted (though it requires additional handling if your
filesystem is sufficiently complicated). However, it has no `Dockerfile`-like
build tooling (it's a slightly lower-level tool that can be used to build such
builders -- such as `orca-build`).

`Buildah` specializes in building OCI images.  Buildah's commands replicate all
of the commands that are found in a Dockerfile.  This allows building images
with and without Dockerfiles while not requiring any root privileges.
Buildah’s ultimate goal is to provide a lower-level coreutils interface to
build images.  The flexibility of building images without Dockerfiles allows
for the integration of other scripting languages into the build process.
Buildah follows a simple fork-exec model and does not run as a daemon
but it is based on a comprehensive API in golang, which can be vendored
into other tools.

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
if the file has changed. Unfortunately, there is a delay between when changes to a
file are made and when the `mtime` is updated. This means:

* With the time-only snapshot mode (`--snapshotMode=time`), kaniko may miss changes
  introduced by `RUN` commands entirely.
* With the default snapshot mode (`--snapshotMode=full`), whether or not kaniko will
  add a layer in the case where a `RUN` command modifies a file **but the contents do
  not** change is theoretically non-deterministic. This _does not affect the contents_
  which will still be correct, but it does affect the number of layers.

_Note that these issues are currently theoretical only. If you see this issue occur, please
[open an issue](https://github.com/GoogleContainerTools/kaniko/issues)._
