# kaniko

kaniko is a tool to build container images from a Dockerfile without a Docker daemon. This enables building container images in unpriviliged environments, which can't easily or securely run a Docker daemon, such as a standard Kubernetes cluster. 

The majority of Dockerfile commands can be executed with kaniko, but we're still working on supporting the following commands:
    * ADD
    * SHELL
    * HEALTHCHECK
    * STOPSIGNAL
    * ONBUILD
    * ARG
    * VOLUME

We're currently in the process of building kaniko, so as of now it isn't production ready. Please let us know if you have any feature requests or find any bugs!

## Running kaniko in a Kubernetes cluster

kaniko runs as an image, which is responsible for building the final image from a Dockerfile and pushing it to a GCR registry.

`make images`

The image takes in three arguments: a path to a Dockerfile, a path to a build context, and the GCR registry the final image should be pushed to (in the form gcr.io/$PROJECT/$IMAGE:$TAG)


## Comparison with Other Tools

Similar tools include:
    * [img](https://github.com/genuinetools/img)
    * [orca-build](https://github.com/cyphar/orca-build)
    * [buildah](https://github.com/projectatomic/buildah)

All of these tools build container images; however, the way in which they accomplish this differs from kaniko. Both kaniko and img build unprivileged images, but they interpret “unprivileged” differently. img builds as a non root user from within the container, while kaniko is run in an unprivileged environment with root access inside the container. 

Unlike orca-build, kaniko doesn't use runC to build images. Instead, it runs as a root user within the container.

buildah requires the same root privilges as a Docker daemon does to run, while kaniko runs without any special privileges or permissions.  
