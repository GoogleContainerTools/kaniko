// +build seccomp

/*
Copyright 2018 Google LLC

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

package seccomp

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/opencontainers/runc/libcontainer/specconv"

	"github.com/sirupsen/logrus"

	"github.com/containerd/containerd/namespaces"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"

	sc "github.com/containerd/containerd/contrib/seccomp"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/seccomp"
)

var DefaultProfile *configs.Seccomp

func init() {

	// In order to generate the default profile, we have to create a fake containerd object,
	// namespace and spec to pass through to the Default profile method.
	c := containers.Container{
		ID: "",
	}
	ctx := namespaces.WithNamespace(context.Background(), "unused")
	s, err := oci.GenerateSpec(ctx, nil, &c)
	if err != nil {
		panic(err)
	}
	cfg := sc.DefaultProfile(s)
	if DefaultProfile, err = specconv.SetupSeccomp(cfg); err != nil {
		panic(err)
	}
}

func InitSeccomp() error {
	// Make sure the binary was compiled with support for seccomp, and the process isn't already running with seccomp.
	if seccomp.IsEnabled() && !IsProcessSeccommped() {
		return seccomp.InitSeccomp(DefaultProfile)
	}
	logrus.Infof("Not enabling seccomp because it isn't supported or is already enabled.")
	return nil
}

func IsProcessSeccommped() bool {
	// There will be a line in proc/self/status like:
	// Seccomp: 0|1|2
	// http://man7.org/linux/man-pages/man5/proc.5.html
	b, err := ioutil.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		split := strings.SplitN(line, "\t", 2)
		if split[0] == "Seccomp:" {
			return split[1] != "0"
		}
	}
	return false
}
