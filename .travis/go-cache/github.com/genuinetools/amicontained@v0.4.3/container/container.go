// Package container provides tools for introspecting containers.
package container

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/syndtr/gocapability/capability"
	"golang.org/x/sys/unix"
)

const (
	// RuntimeDocker is the string for the docker runtime.
	RuntimeDocker = "docker"
	// RuntimeRkt is the string for the rkt runtime.
	RuntimeRkt = "rkt"
	// RuntimeNspawn is the string for the systemd-nspawn runtime.
	RuntimeNspawn = "systemd-nspawn"
	// RuntimeLXC is the string for the lxc runtime.
	RuntimeLXC = "lxc"
	// RuntimeLXCLibvirt is the string for the lxc-libvirt runtime.
	RuntimeLXCLibvirt = "lxc-libvirt"
	// RuntimeOpenVZ is the string for the openvz runtime.
	RuntimeOpenVZ = "openvz"
	// RuntimeKubernetes is the string for the kubernetes runtime.
	RuntimeKubernetes = "kube"
	// RuntimeGarden is the string for the garden runtime.
	RuntimeGarden = "garden"

	uint32Max = 4294967295
)

var (
	// ErrContainerRuntimeNotFound describes when a container runtime could not be found.
	ErrContainerRuntimeNotFound = errors.New("container runtime could not be found")

	runtimes = []string{RuntimeDocker, RuntimeRkt, RuntimeNspawn, RuntimeLXC, RuntimeLXCLibvirt, RuntimeOpenVZ, RuntimeKubernetes, RuntimeGarden}
)

// DetectRuntime returns the container runtime the process is running in.
func DetectRuntime() (string, error) {
	// read the cgroups file
	cgroups := readFile("/proc/self/cgroup")
	if len(cgroups) > 0 {
		for _, runtime := range runtimes {
			if strings.Contains(cgroups, runtime) {
				return runtime, nil
			}
		}
	}

	// /proc/vz exists in container and outside of the container, /proc/bc only outside of the container.
	if fileExists("/proc/vz") && !fileExists("/proc/bc") {
		return RuntimeOpenVZ, nil
	}

	ctrenv := os.Getenv("container")
	if ctrenv != "" {
		for _, runtime := range runtimes {
			if ctrenv == runtime {
				return runtime, nil
			}
		}
	}

	// PID 1 might have dropped this information into a file in /run.
	// Read from /run/systemd/container since it is better than accessing /proc/1/environ,
	// which needs CAP_SYS_PTRACE
	f := readFile("/run/systemd/container")
	if len(f) > 0 {
		for _, runtime := range runtimes {
			if f == runtime {
				return runtime, nil
			}
		}
	}

	return "not-found", ErrContainerRuntimeNotFound
}

// HasNamespace determines if the container is using a particular namespace or the
// host namespace.
// The device number of an unnamespaced /proc/1/ns/{ns} is 4 and anything else is
// higher.
func HasNamespace(ns string) (bool, error) {
	file := fmt.Sprintf("/proc/1/ns/%s", ns)

	// Use Lstat to not follow the symlink.
	var info syscall.Stat_t
	if err := syscall.Lstat(file, &info); err != nil {
		return false, &os.PathError{Op: "lstat", Path: file, Err: err}
	}

	// Get the device number. If it is higher than 4 it is in a namespace.
	if info.Dev > 4 {
		return true, nil
	}

	return false, nil
}

// AppArmorProfile determines the apparmor profile for a container.
func AppArmorProfile() string {
	f := readFile("/proc/self/attr/current")
	if f == "" {
		return "none"
	}
	return f
}

// UserMapping holds the values for a {uid,gid}_map.
type UserMapping struct {
	ContainerID int64
	HostID      int64
	Range       int64
}

// UserNamespace determines if the container is running in a UserNamespace and returns the mappings if so.
func UserNamespace() (bool, []UserMapping) {
	f := readFile("/proc/self/uid_map")
	if len(f) < 0 {
		// user namespace is uninitialized
		return true, nil
	}

	userNs, mappings, err := readUserMappings(f)
	if err != nil {
		return false, nil
	}

	return userNs, mappings
}

func readUserMappings(f string) (iuserNS bool, mappings []UserMapping, err error) {
	parts := strings.Split(f, " ")
	parts = deleteEmpty(parts)
	if len(parts) < 3 {
		return false, nil, nil
	}

	for i := 0; i < len(parts); i += 3 {
		nsu, hu, r := parts[i], parts[i+1], parts[i+2]
		mapping := UserMapping{}

		mapping.ContainerID, err = strconv.ParseInt(nsu, 10, 0)
		if err != nil {
			return false, nil, nil
		}
		mapping.HostID, err = strconv.ParseInt(hu, 10, 0)
		if err != nil {
			return false, nil, nil
		}
		mapping.Range, err = strconv.ParseInt(r, 10, 0)
		if err != nil {
			return false, nil, nil
		}

		if mapping.ContainerID == 0 && mapping.HostID == 0 && mapping.Range == uint32Max {
			return false, nil, nil
		}

		mappings = append(mappings, mapping)
	}

	return true, mappings, nil
}

// Capabilities returns the allowed capabilities in the container.
func Capabilities() (map[string][]string, error) {
	allCaps := capability.List()

	caps, err := capability.NewPid(0)
	if err != nil {
		return nil, err
	}

	allowedCaps := map[string][]string{}
	allowedCaps["EFFECTIVE | PERMITTED | INHERITABLE"] = []string{}
	allowedCaps["BOUNDING"] = []string{}
	allowedCaps["AMBIENT"] = []string{}

	for _, cap := range allCaps {
		if caps.Get(capability.CAPS, cap) {
			allowedCaps["EFFECTIVE | PERMITTED | INHERITABLE"] = append(allowedCaps["EFFECTIVE | PERMITTED | INHERITABLE"], cap.String())
		}
		if caps.Get(capability.BOUNDING, cap) {
			allowedCaps["BOUNDING"] = append(allowedCaps["BOUNDING"], cap.String())
		}
		if caps.Get(capability.AMBIENT, cap) {
			allowedCaps["AMBIENT"] = append(allowedCaps["AMBIENT"], cap.String())
		}
	}

	return allowedCaps, nil
}

// Chroot detects if we are running in a chroot or a pivot_root.
// Currently, we can not distinguish between the two.
func Chroot() (bool, error) {
	var a, b syscall.Stat_t

	if err := syscall.Lstat("/proc/1/root", &a); err != nil {
		return false, err
	}

	if err := syscall.Lstat("/", &b); err != nil {
		return false, err
	}

	return a.Ino == b.Ino && a.Dev == b.Dev, nil
}

// SeccompEnforcingMode returns the seccomp enforcing level (disabled, filtering, strict)
func SeccompEnforcingMode() (string, error) {
	// Read from /proc/self/status Linux 3.8+
	s := readFile("/proc/self/status")

	// Pre linux 3.8
	if !strings.Contains(s, "Seccomp") {
		// Check if Seccomp is supported, via CONFIG_SECCOMP.
		if err := unix.Prctl(unix.PR_GET_SECCOMP, 0, 0, 0, 0); err != unix.EINVAL {
			// Make sure the kernel has CONFIG_SECCOMP_FILTER.
			if err := unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, 0, 0, 0); err != unix.EINVAL {
				return "strict", nil
			}
		}
		return "disabled", nil
	}

	// Split status file string by line
	statusMappings := strings.Split(s, "\n")
	statusMappings = deleteEmpty(statusMappings)

	mode := "-1"
	for _, line := range statusMappings {
		if strings.Contains(line, "Seccomp:") {
			mode = string(line[len(line)-1])
		}
	}

	seccompModes := map[string]string{
		"0": "disabled",
		"1": "strict",
		"2": "filtering",
	}

	seccompMode, ok := seccompModes[mode]
	if !ok {
		return "", errors.New("could not retrieve seccomp filtering status")
	}

	return seccompMode, nil
}

func fileExists(file string) bool {
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true
	}
	return false
}

func readFile(file string) string {
	if !fileExists(file) {
		return ""
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if strings.TrimSpace(str) != "" {
			r = append(r, strings.TrimSpace(str))
		}
	}
	return r
}
