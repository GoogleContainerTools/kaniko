# Rootless mode (Experimental)

Requirements:
- runc `ecd55a4135e0a26de884ce436442914f945b1e76` (May 30, 2018) or later
- Some distros such as Debian (excluding Ubuntu) and Arch Linux require `echo 1 > /proc/sys/kernel/unprivileged_userns_clone`
- `newuidmap` and `newgidmap` need to be installed on the host. These commands are provided by the `uidmap` package.
- `/etc/subuid` and `/etc/subgid` should contain >= 65536 sub-IDs. e.g. `penguin:231072:65536`.
- To run in a Docker container with non-root `USER`, `docker run --privileged` is still required. See also Jessie's blog: https://blog.jessfraz.com/post/building-container-images-securely-on-kubernetes/


## Set up

Setting up rootless mode also requires some bothersome steps as follows, but you can also use [`rootlesskit`](https://github.com/rootless-containers/rootlesskit) for automating these steps.

### Terminal 1:

```
$ unshare -U -m
unshared$ echo $$ > /tmp/pid
```

Unsharing mountns (and userns) is required for mounting filesystems without real root privileges.

### Terminal 2:

```
$ id -u
1001
$ grep $(whoami) /etc/subuid
penguin:231072:65536
$ grep $(whoami) /etc/subgid
penguin:231072:65536
$ newuidmap $(cat /tmp/pid) 0 1001 1 1 231072 65536
$ newgidmap $(cat /tmp/pid) 0 1001 1 1 231072 65536
```

### Terminal 1:

```
unshared# buildkitd
```

* The data dir will be set to `/home/penguin/.local/share/buildkit`
* The address will be set to `unix:///run/user/1001/buildkit/buildkitd.sock`
* `overlayfs` snapshotter is not supported except Ubuntu-flavored kernel: http://kernel.ubuntu.com/git/ubuntu/ubuntu-artful.git/commit/fs/overlayfs?h=Ubuntu-4.13.0-25.29&id=0a414bdc3d01f3b61ed86cfe3ce8b63a9240eba7
* containerd worker is not supported ( pending PR: https://github.com/containerd/containerd/pull/2006 )
* Network namespace is not used at the moment.

### Terminal 2:

```
$ go get ./examples/build-using-dockerfile
$ build-using-dockerfile --buildkit-addr unix:///run/user/1001/buildkit/buildkitd.sock -t foo /path/to/somewhere
```

## Set up (using a container)

```
$ docker build -t buildkit-rootless --target rootless -f hack/dockerfiles/test.Dockerfile .
$ docker run --name buildkitd -d --privileged -p 1234:1234  buildkit-rootless --addr tcp://0.0.0.0:1234
```

`docker run` requires `--privileged` but the BuildKit daemon is executed as a normal user.
See [`moby/moby#36597`](https://github.com/moby/moby/issues/36597, [`kubernetes/community#1934`](https://github.com/kubernetes/community/pull/1934) and [Jess's blog](https://blog.jessfraz.com/post/building-container-images-securely-on-kubernetes/) for the ongoing work to remove this requirement

```
$ docker exec buildkitd id
uid=1000(user) gid=1000(user)
$ docker exec buildkitd ps aux
PID   USER     TIME   COMMAND
    1 user       0:00 rootlesskit buildkitd --addr tcp://0.0.0.0:1234
   13 user       0:00 /proc/self/exe buildkitd --addr tcp://0.0.0.0:1234
   21 user       0:00 buildkitd --addr tcp://0.0.0.0:1234
   29 user       0:00 ps aux
```

```
$ go get ./examples/build-using-dockerfile
$ build-using-dockerfile --buildkit-addr tcp://127.0.0.1:1234 -t foo /path/to/somewhere
```
