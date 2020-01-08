# amicontained

[![Travis CI](https://img.shields.io/travis/genuinetools/amicontained.svg?style=for-the-badge)](https://travis-ci.org/genuinetools/amicontained)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/genuinetools/amicontained)
[![Github All Releases](https://img.shields.io/github/downloads/genuinetools/amicontained/total.svg?style=for-the-badge)](https://github.com/genuinetools/amicontained/releases)

Container introspection tool. Find out what container runtime is being used as
well as features available.

- [Installation](#installation)
    + [Binaries](#binaries)
    + [Via Go](#via-go)
- [Usage](#usage)
- [Examples](#examples)
    + [docker](#docker)
    + [lxc](#lxc)
    + [systemd-nspawn](#systemd-nspawn)
    + [rkt](#rkt)
    + [unshare](#unshare)

## Installation

#### Binaries

For installation instructions from binaries please visit the [Releases Page](https://github.com/genuinetools/amicontained/releases).

- **linux** [386](https://github.com/genuinetools/amicontained/releases/download/v0.4.3/amicontained-linux-386) / [amd64](https://github.com/genuinetools/amicontained/releases/download/v0.4.3/amicontained-linux-amd64) / [arm](https://github.com/genuinetools/amicontained/releases/download/v0.4.3/amicontained-linux-arm) / [arm64](https://github.com/genuinetools/amicontained/releases/download/v0.4.3/amicontained-linux-arm64)

#### Via Go

```bash
$ go get github.com/genuinetools/amicontained
```

## Usage

```console
$ amicontained -h
amicontained -  A container introspection tool.

Usage: amicontained <command>

Flags:

  -d  enable debug logging (default: false)

Commands:

  version  Show the version information.
```

## Examples

#### docker

```console
$ docker run --rm -it r.j3ss.co/amicontained
Container Runtime: docker
Has Namespaces:
        pid: true
        user: true
User Namespace Mappings:
	Container -> 0
	Host -> 886432
	Range -> 65536
AppArmor Profile: docker-default (enforce)
Capabilities:
	BOUNDING -> chown dac_override fowner fsetid kill setgid setuid setpcap net_bind_service net_raw sys_chroot mknod audit_write setfcap
Chroot (not pivot_root): false

$ docker run --rm -it --pid host r.j3ss.co/amicontained
Container Runtime: docker
Has Namespaces:
        pid: false
        user: false
AppArmor Profile: docker-default (enforce)
Capabilities:
	BOUNDING -> chown dac_override fowner fsetid kill setgid setuid setpcap net_bind_service net_raw sys_chroot mknod audit_write setfcap
Chroot (not pivot_root): false

$ docker run --rm -it --security-opt "apparmor=unconfined" r.j3ss.co/amicontained
Container Runtime: docker
Has Namespaces:
        pid: true
        user: false
AppArmor Profile: unconfined
Capabilities:
	BOUNDING -> chown dac_override fowner fsetid kill setgid setuid setpcap net_bind_service net_raw sys_chroot mknod audit_write setfcap
Chroot (not pivot_root): false
```

#### lxc

```console
$ lxc-attach -n xenial
root@xenial:/# amicontained
Container Runtime: lxc
Has Namespaces:
        pid: true
        user: true
User Namespace Mappings:
	Container -> 0	Host -> 100000	Range -> 65536
AppArmor Profile: none
Capabilities:
	BOUNDING -> chown dac_override dac_read_search fowner fsetid kill setgid setuid setpcap linux_immutable net_bind_service net_broadcast net_admin net_raw ipc_lock ipc_owner sys_chroot sys_ptrace sys_pacct sys_admin sys_boot sys_nice sys_resource sys_tty_config mknod lease audit_write audit_control setfcap syslog wake_alarm block_suspend audit_read
Chroot (not pivot_root): false

$ lxc-execute -n xenial -- /bin/amicontained
Container Runtime: lxc
Has Namespaces:
        pid: true
        user: true
User Namespace Mappings:
	Container -> 0	Host -> 100000	Range -> 65536
AppArmor Profile: none
Capabilities:
	BOUNDING -> chown dac_override dac_read_search fowner fsetid kill setgid setuid setpcap linux_immutable net_bind_service net_broadcast net_admin net_raw ipc_lock ipc_owner sys_chroot sys_ptrace sys_pacct sys_admin sys_boot sys_nice sys_resource sys_tty_config mknod lease audit_write audit_control setfcap syslog wake_alarm block_suspend audit_read
Chroot (not pivot_root): false
```

#### systemd-nspawn

```console
$ sudo systemd-nspawn --machine amicontained --directory nspawn-amicontained /usr/bin/amicontained
Spawning container amicontained on /home/genuinetools/nspawn-amicontained.
Press ^] three times within 1s to kill container.
Timezone UTC does not exist in container, not updating container timezone.
Container Runtime: systemd-nspawn
Has Namespaces:
        pid: true
        user: false
AppArmor Profile: none
Capabilities:
	BOUNDING -> chown dac_override dac_read_search fowner fsetid kill setgid setuid setpcap linux_immutable net_bind_service net_broadcast net_raw ipc_owner sys_chroot sys_ptrace sys_admin sys_boot sys_nice sys_resource sys_tty_config mknod lease audit_write audit_control setfcap
Chroot (not pivot_root): false
Container amicontained exited successfully.
```

#### rkt

```console
$ sudo rkt --insecure-options=image run docker://r.j3ss.co/amicontained
[  631.522121] amicontained[5]: Container Runtime: rkt
[  631.522471] amicontained[5]: Host PID Namespace: false
[  631.522617] amicontained[5]: AppArmor Profile: none
[  631.522768] amicontained[5]: User Namespace: false
[  631.522922] amicontained[5]: Capabilities:
[  631.523075] amicontained[5]: 	BOUNDING -> chown dac_override fowner fsetid kill setgid setuid setpcap net_bind_service net_raw sys_chroot mknod audit_write setfcap
[  631.523213] amicontained[5]: Chroot (not pivot_root): false

$ sudo rkt --insecure-options=image run  --private-users=true --no-overlay docker://r.j3ss.co/amicontained
[  785.547050] amicontained[5]: Container Runtime: rkt
[  785.547360] amicontained[5]: Host PID Namespace: false
[  785.547567] amicontained[5]: AppArmor Profile: none
[  785.547717] amicontained[5]: User Namespace: true
[  785.547856] amicontained[5]: User Namespace Mappings:
[  785.548064] amicontained[5]: 	Container -> 0	Host -> 229834752	Range -> 65536
[  785.548335] amicontained[5]: Capabilities:
[  785.548537] amicontained[5]: 	BOUNDING -> chown dac_override fowner fsetid kill setgid setuid setpcap net_bind_service net_raw sys_chroot mknod audit_write setfcap
[  785.548679] amicontained[5]: Chroot (not pivot_root): false
```

#### unshare

```console
$ sudo unshare --user -r
root@coreos:/home/jessie/.go/src/github.com/genuinetools/amicontained# ./amicontained
Container Runtime: not-found
Has Namespaces:
        pid: false
        user: true
User Namespace Mappings:
	Container -> 0
	Host -> 0
	Range -> 1
AppArmor Profile: unconfined
Capabilities:
	BOUNDING -> chown dac_override dac_read_search fowner fsetid kill setgid setuid setpcap linux_immutable net_bind_service net_broadcast net_admin net_raw ipc_lock ipc_owner sys_module sys_rawio sys_chroot sys_ptrace sys_pacct sys_admin sys_boot sys_nice sys_resource sys_time sys_tty_config mknod lease audit_write audit_control setfcap mac_override mac_admin syslog wake_alarm block_suspend audit_read
Chroot (not pivot_root): false
```
