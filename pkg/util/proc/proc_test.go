/*
Copyright 2022 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proc

import (
	"testing"
)

func TestGetContainerRuntime(t *testing.T) {
	testcases := map[string]struct {
		expectedRuntime ContainerRuntime
		input           string
	}{
		"empty": {
			expectedRuntime: RuntimeNotFound,
		},
		"typical docker": {
			expectedRuntime: RuntimeDocker,
			input: `11:pids:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
10:devices:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
9:freezer:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
8:net_cls,net_prio:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
7:perf_event:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
6:cpuset:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
5:memory:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
4:blkio:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
3:cpu,cpuacct:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
2:hugetlb:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
1:name=systemd:/docker/68fad1f9e0985989408aff30e7b83e7dada1d235ff46a22c5465ca193ddf0fac
0::/system.slice/containerd.service`,
		},
		"uncontainerized process": {
			expectedRuntime: RuntimeNotFound,
			input: `11:pids:/system.slice/ssh.service
10:devices:/system.slice/ssh.service
9:freezer:/
8:net_cls,net_prio:/
7:perf_event:/
6:cpuset:/
5:memory:/system.slice/ssh.service
4:blkio:/system.slice/ssh.service
3:cpu,cpuacct:/system.slice/ssh.service
2:hugetlb:/
1:name=systemd:/system.slice/ssh.service
0::/system.slice/ssh.service`,
		},
		"kubernetes": {
			expectedRuntime: RuntimeKubernetes,
			input: `12:perf_event:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
11:freezer:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
10:pids:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
9:net_cls,net_prio:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
8:memory:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
7:cpuset:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
6:devices:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
5:blkio:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
4:rdma:/
3:hugetlb:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
2:cpu,cpuacct:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47
1:name=systemd:/kubepods/burstable/pod98051bd2-a5fa-11e8-9bb9-0a58ac1f31f2/74998d19bd3c7423744214c344c6e814d19b7908a92f165f9d58243073a27a47`,
		},
		"lxc": {
			expectedRuntime: RuntimeLXC, // this is usually in $container for lxc
			input: `10:cpuset:/lxc/debian2
9:pids:/lxc/debian2
8:devices:/lxc/debian2
7:net_cls,net_prio:/lxc/debian2
6:freezer:/lxc/debian2
5:blkio:/lxc/debian2
4:memory:/lxc/debian2
3:cpu,cpuacct:/lxc/debian2
2:perf_event:/lxc/debian2
1:name=systemd:/lxc/debian2`,
		},
		"nspawn": {
			expectedRuntime: RuntimeNotFound, // since this variable is in $container
			input: `10:cpuset:/
9:pids:/machine.slice/machine-nspawntest.scope
8:devices:/machine.slice/machine-nspawntest.scope
7:net_cls,net_prio:/
6:freezer:/user/root/0
5:blkio:/machine.slice/machine-nspawntest.scope
4:memory:/machine.slice/machine-nspawntest.scope
3:cpu,cpuacct:/machine.slice/machine-nspawntest.scope
2:perf_event:/
1:name=systemd:/machine.slice/machine-nspawntest.scope`,
		},
		"rkt": {
			expectedRuntime: RuntimeRkt,
			input: `10:cpuset:/
9:pids:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice/debian.service
8:devices:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice/debian.service
7:net_cls,net_prio:/
6:freezer:/user/root/0
5:blkio:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice
4:memory:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice
3:cpu,cpuacct:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice
2:perf_event:/
1:name=systemd:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice/debian.service`,
		},
		"rkt host": {
			expectedRuntime: RuntimeRkt,
			input: `10:cpuset:/
9:pids:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice/debian.service
8:devices:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice/debian.service
7:net_cls,net_prio:/
6:freezer:/user/root/0
5:blkio:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice
4:memory:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice
3:cpu,cpuacct:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice
2:perf_event:/
1:name=systemd:/machine.slice/machine-rkt\x2dbfb7d57e\x2d80ff\x2d4ef8\x2db602\x2d9b907b3f3a38.scope/system.slice/debian.service`,
		},
	}

	for key, tc := range testcases {
		runtime := getContainerRuntime(tc.input)
		if runtime != tc.expectedRuntime {
			t.Errorf("[%s]: expected runtime %q, got %q", key, tc.expectedRuntime, runtime)
		}
	}
}
