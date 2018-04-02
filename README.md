# kaniko

kaniko is a tool to build unpriviliged container images from a Dockerfile. It doesn't depend on a Docker daemon, which enables building container images in environments that can't easily or securely run a Docker daemon, such as a standard Kubernetes cluster. 

The majority of Dockerfile commands can be executed with kaniko, but we're still working on supporting the following commands:
    * ADD
    * VOLUME
    * SHELL
    * HEALTHCHECK
    * STOPSIGNAL
    * ONBUILD
    * ARG

We're currently in the process of building kaniko, so as of now it isn't production ready. Please let us know if you have any feature requests or find any bugs!

## How does it work?

The kaniko executor image is responsible for building the final image from a Dockerfile and pushing it to a registry. Within the executor image, we extract the filesystem of the base image (the FROM image in the Dockerfile). We then execute the commands in the Dockerfile, snapshotting the filesystem in userspace after each one. After each command, we append a layer of changed files to the base image (if there are any) and update image metadata.

## kaniko Build Context
kaniko supports local directories and GCS buckets as build contexts. To specify a local directory, pass in the `--context=<path to build context>` flag as an argument to the executor image. To specify a GCS bucket, pass in the `--bucket=<GCS bucket name>` flag. The GCS bucket should contain a compressed tar of the build context called `context.tar.gz`, which kaniko will unpack and use as the build context. 

To easily create `context.tar.gz`, we can use [skaffold](https://github.com/GoogleCloudPlatform/skaffold).

Running `skaffold docker context` will create `context.tar.gz`, which will contain the Dockerfile and any files it depends on.

We can copy over the compressed tar with gsutil:
`gsutil cp context.tar.gz gs://<bucket name>`

## Running kaniko locally

Requirements:
    * Docker
    * gcloud

We can run the kaniko executor image locally in a Docker daemon to build and push an image from a Dockerfile.

First, to build the executor image locally, run `make images`.  This will load the executor image into your Docker daemon.

To run kaniko in Docker, run the following command:
`./run_in_docker.sh <path to build context> <destination of final image in the form gcr.io/$PROJECT/$IMAGE:$TAG>`

## Running kaniko in a Kubernetes cluster

Requirements:
    * Standard Kubernetes cluster
    * Kubernetes Secret
    
To run kaniko in a Kubernetes cluster, you will need a standard running Kubernetes cluster and a Kubernetes secret, which contains the auth required to push the final image. 

To create the secret, first you will need to create a service account in the Pantheon project you want to push the final image to, with `Storage Admin` permissions. You can download a JSON key for this service account, and rename it `kaniko-secret.json`. To create the secret, run:

`kubectl create secret generic kaniko-secret --from-file=<path to kaniko-secret.json>`

The Kubernetes job.yaml should look similar to this, with the args parameters filled in:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kaniko
spec:
  template:
    spec:
      containers:
      - name: kaniko
        image: gcr.io/kaniko-project/executor:latest
        args: ["--dockerfile=<path to Dockerfile>", "--bucket=<GCS bucket where context.tar.gz lives>", "--destination=<gcr.io/$PROJECT/$IMAGE:$TAG>"]
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

This example pulls the build context from a GCS bucket. To use a local directory build context, you could consider using configMaps to mount in small build context.

## Comparison with Other Tools/Solutions

Similar tools include:
    * [img](https://github.com/genuinetools/img)
    * [orca-build](https://github.com/cyphar/orca-build)
    * [buildah](https://github.com/projectatomic/buildah)
    * [Bazel](https://github.com/bazelbuild/rules_docker)/[FTL](https://github.com/GoogleCloudPlatform/runtimes-common/tree/master/ftl)

All of these tools build container images; however, the way in which they accomplish this differs from kaniko. Both kaniko and img build unprivileged images, but they interpret “unprivileged” differently. img builds as a non root user from within the container, while kaniko is run in an unprivileged environment with root access inside the container. 

Unlike orca-build, kaniko doesn't use runC to build images. Instead, it runs as a root user within the container.

buildah requires the same root privilges as a Docker daemon does to run, while kaniko runs without any special privileges or permissions.  

Bazel/FTL aim to improve DevEx by achieving the fastest possible creation of Docker images, at the expense of build compatibility. By restricting the set of allowed builds to an optimizable subset, we get the nice side effect of being able to run without privileges inside an arbitrary cluster.

These approaches can be thought of as special-case "fast paths" that can be used in conjunction with the support for general Dockerfile kaniko provides.
