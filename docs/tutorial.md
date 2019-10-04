# Getting Started Tutorial

This tutorial is for beginners who want to start using kaniko and aims to establish a quick start test case.

## Table of Content

1. [Prerequisities](#Prerequisities)
2. [Prepare config files for kaniko](#Prepare-config-files-for-kaniko)
3. [Prepare the local mounted directory](#Prepare-the-local-mounted-directory)
4. [Create a Secret that holds your authorization token](#Create-a-Secret-that-holds-your-authorization-token)
5. [Create resources in kubernetes](#Create-resources-in-kubernetes)
6. [Pull the image and test](#Pull-the-image-and-test)

## Prerequisities

- A Kubernetes Cluster. You could use [Minikube](https://kubernetes.io/docs/setup/minikube/) to deploy kubernetes locally, or use kubernetes service from cloud provider like [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/).
- A [dockerhub](https://hub.docker.com/) account to push built image public.

## Prepare config files for kaniko

Prepare several config files to create resources in kubernetes, which are:

- [pod.yaml](../examples/pod.yaml) is for starting a kaniko container to build the example image.
- [volume.yaml](../examples/volume.yaml) is for creating a persistent volume used as kaniko build context.
- [volume-claim.yaml](../examples/volume-claim.yaml) is for creating a persistent volume claim which will mounted in the kaniko container.

## Prepare the local mounted directory

SSH into the cluster, and create a local directory which will be mounted in kaniko container as build context. Create a simple dockerfile there.

> Note: To ssh into cluster, if you use minikube, you could use `minikube ssh` command. If you use cloud service, please refer to official doc, such as [Azure Kubernetes Service](https://docs.microsoft.com/en-us/azure/aks/ssh#code-try-0).

```shell
$ mkdir kaniko && cd kaniko
$ echo 'FROM ubuntu' >> dockerfile
$ echo 'ENTRYPOINT ["/bin/bash", "-c", "echo hello"]' >> dockerfile
$ cat dockerfile
FROM ubuntu
ENTRYPOINT ["/bin/bash", "-c", "echo hello"]
$ pwd
/home/<user-name>/kaniko # copy this path in volume.yaml file
```

> Note: It is important to notice that the `hostPath` in the volume.yaml need to be replaced with the local directory you created.

## Create a Secret that holds your authorization token

A Kubernetes cluster uses the Secret of docker-registry type to authenticate with a docker registry to push an image.

Create this Secret, naming it regcred:

```shell
kubectl create secret docker-registry regcred --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email>
```

- `<your-registry-server>` is your Private Docker Registry FQDN. (https://index.docker.io/v1/ for DockerHub)
- `<your-name>` is your Docker username.
- `<your-pword>` is your Docker password.
- `<your-email>` is your Docker email.

This secret will be used in pod.yaml config.

## Create resources in kubernetes

```shell
# create persistent volume
$ kubectl create -f volume.yaml
persistentvolume/dockerfile created

# create persistent volume claim
$ kubectl create -f volume-claim.yaml
persistentvolumeclaim/dockerfile-claim created

# check whether the volume mounted correctly
$ kubectl get pv dockerfile
NAME         CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                      STORAGECLASS    REASON   AGE
dockerfile   10Gi       RWO            Retain           Bound    default/dockerfile-claim   local-storage            1m

# create pod
$ kubectl create -f pod.yaml
pod/kaniko created
$ kubectl get pods
NAME     READY   STATUS              RESTARTS   AGE
kaniko   0/1     ContainerCreating   0          7s

# check whether the build complete and show the build logs
$ kubectl get pods
NAME     READY   STATUS      RESTARTS   AGE
kaniko   0/1     Completed   0          34s
$ kubectl logs kaniko
➜ kubectl logs kaniko
INFO[0000] Resolved base name ubuntu to ubuntu
INFO[0000] Resolved base name ubuntu to ubuntu
INFO[0000] Downloading base image ubuntu
INFO[0000] Error while retrieving image from cache: getting file info: stat /cache/sha256:1bbdea4846231d91cce6c7ff3907d26fca444fd6b7e3c282b90c7fe4251f9f86: no such file or directory
INFO[0000] Downloading base image ubuntu
INFO[0001] Built cross stage deps: map[]
INFO[0001] Downloading base image ubuntu
INFO[0001] Error while retrieving image from cache: getting file info: stat /cache/sha256:1bbdea4846231d91cce6c7ff3907d26fca444fd6b7e3c282b90c7fe4251f9f86: no such file or directory
INFO[0001] Downloading base image ubuntu
INFO[0001] Skipping unpacking as no commands require it.
INFO[0001] Taking snapshot of full filesystem...
INFO[0001] ENTRYPOINT ["/bin/bash", "-c", "echo hello"]
```

> Note: It is important to notice that the `destination` in the pod.yaml need to be replaced with your own.

## Pull the image and test

If as expected, the kaniko will build image and push to dockerhub successfully. Pull the image to local and run it to test:

```shell
$ sudo docker run -it <user-name>/<repo-name>
Unable to find image 'debuggy/helloworld:latest' locally
latest: Pulling from debuggy/helloworld
5667fdb72017: Pull complete
d83811f270d5: Pull complete
ee671aafb583: Pull complete
7fc152dfb3a6: Pull complete
Digest: sha256:2707d17754ea99ce0cf15d84a7282ae746a44ff90928c2064755ee3b35c1057b
Status: Downloaded newer image for debuggy/helloworld:latest
hello
```

Congratulation! You have gone through the hello world successfully, please refer to project for more details.
