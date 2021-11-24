// Package proc provides tools for inspecting proc.
package proc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/syndtr/gocapability/capability"
	"golang.org/x/sys/unix"
)

// ContainerRuntime is the type for the various container runtime strings.
type ContainerRuntime string

// SeccompMode is the type for the various seccomp mode strings.
type SeccompMode string

const (
	// RuntimeDocker is the string for the docker runtime.
	RuntimeDocker ContainerRuntime = "docker"
	// RuntimeRkt is the string for the rkt runtime.
	RuntimeRkt ContainerRuntime = "rkt"
	// RuntimeNspawn is the string for the systemd-nspawn runtime.
	RuntimeNspawn ContainerRuntime = "systemd-nspawn"
	// RuntimeLXC is the string for the lxc runtime.
	RuntimeLXC ContainerRuntime = "lxc"
	// RuntimeLXCLibvirt is the string for the lxc-libvirt runtime.
	RuntimeLXCLibvirt ContainerRuntime = "lxc-libvirt"
	// RuntimeOpenVZ is the string for the openvz runtime.
	RuntimeOpenVZ ContainerRuntime = "openvz"
	// RuntimeKubernetes is the string for the kubernetes runtime.
	RuntimeKubernetes ContainerRuntime = "kube"
	// RuntimeGarden is the string for the garden runtime.
	RuntimeGarden ContainerRuntime = "garden"
	// RuntimePodman is the string for the podman runtime.
	RuntimePodman ContainerRuntime = "podman"
	// RuntimeGVisor is the string for the gVisor (runsc) runtime.
	RuntimeGVisor ContainerRuntime = "gvisor"
	// RuntimeFirejail is the string for the firejail runtime.
	RuntimeFirejail ContainerRuntime = "firejail"
	// RuntimeWSL is the string for the Windows Subsystem for Linux runtime.
	RuntimeWSL ContainerRuntime = "wsl"
	// RuntimeNotFound is the string for when no container runtime is found.
	RuntimeNotFound ContainerRuntime = "not-found"

	// SeccompModeDisabled is equivalent to "0" in the /proc/{pid}/status file.
	SeccompModeDisabled SeccompMode = "disabled"
	// SeccompModeStrict is equivalent to "1" in the /proc/{pid}/status file.
	SeccompModeStrict SeccompMode = "strict"
	// SeccompModeFiltering is equivalent to "2" in the /proc/{pid}/status file.
	SeccompModeFiltering SeccompMode = "filtering"

	apparmorUnconfined = "unconfined"

	uint32Max = 4294967295

	cgroupContainerID = ":(/docker/|/kube.*/.*/|/kube.*/.*/.*/.*/|/system.slice/docker-|/machine.slice/machine-rkt-|/machine.slice/machine-|/lxc/|/lxc-libvirt/|/garden/|/podman/)([[:alnum:]\\-]{1,64})(.scope|$)"
	statusFileValue   = ":(.*)"
)

var (
	// ContainerRuntimes contains all the container runtimes.
	ContainerRuntimes = []ContainerRuntime{
		RuntimeDocker,
		RuntimeRkt,
		RuntimeNspawn,
		RuntimeLXC,
		RuntimeLXCLibvirt,
		RuntimeOpenVZ,
		RuntimeKubernetes,
		RuntimeGarden,
		RuntimePodman,
		RuntimeGVisor,
		RuntimeFirejail,
		RuntimeWSL,
	}

	seccompModes = map[string]SeccompMode{
		"0": SeccompModeDisabled,
		"1": SeccompModeStrict,
		"2": SeccompModeFiltering,
	}

	cgroupContainerIDRegex = regexp.MustCompile(cgroupContainerID)
	statusFileValueRegex   = regexp.MustCompile(statusFileValue)
)

// GetContainerRuntime returns the container runtime the process is running in.
// If pid is less than one, it returns the runtime for "self".
func GetContainerRuntime(tgid, pid int) ContainerRuntime {
	file := "/proc/self/cgroup"
	if pid > 0 {
		if tgid > 0 {
			file = fmt.Sprintf("/proc/%d/task/%d/cgroup", tgid, pid)
		} else {
			file = fmt.Sprintf("/proc/%d/cgroup", pid)
		}
	}

	// read the cgroups file
	a := readFileString(file)
	runtime := getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// /proc/vz exists in container and outside of the container, /proc/bc only outside of the container.
	if fileExists("/proc/vz") && !fileExists("/proc/bc") {
		return RuntimeOpenVZ
	}

	// /__runsc_containers__ directory is present in gVisor containers.
	if fileExists("/__runsc_containers__") {
		return RuntimeGVisor
	}

	// firejail runs with `firejail` as pid 1.
	// As firejail binary cannot be run with argv[0] != "firejail"
	// it's okay to rely on cmdline.
	a = readFileString("/proc/1/cmdline")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// WSL has /proc/version_signature starting with "Microsoft".
	a = readFileString("/proc/version_signature")
	if strings.HasPrefix(a, "Microsoft") {
		return RuntimeWSL
	}

	a = os.Getenv("container")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// PID 1 might have dropped this information into a file in /run.
	// Read from /run/systemd/container since it is better than accessing /proc/1/environ,
	// which needs CAP_SYS_PTRACE
	a = readFileString("/run/systemd/container")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	return RuntimeNotFound
}

func getContainerRuntime(input string) ContainerRuntime {
	if len(strings.TrimSpace(input)) < 1 {
		return RuntimeNotFound
	}

	for _, runtime := range ContainerRuntimes {
		if strings.Contains(input, string(runtime)) {
			return runtime
		}
	}

	return RuntimeNotFound
}

// GetContainerID returns the container ID for a process if it's running in a container.
// If pid is less than one, it returns the container ID for "self".
func GetContainerID(tgid, pid int) string {
	file := "/proc/self/cgroup"
	if pid > 0 {
		if tgid > 0 {
			file = fmt.Sprintf("/proc/%d/task/%d/cgroup", tgid, pid)
		} else {
			file = fmt.Sprintf("/proc/%d/cgroup", pid)
		}
	}

	return getContainerID(readFileString(file))
}

func getContainerID(input string) string {
	if len(strings.TrimSpace(input)) < 1 {
		return ""
	}

	// rkt encodes the dashes as ascii, replace them.
	input = strings.Replace(input, `\x2d`, "-", -1)

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		matches := cgroupContainerIDRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			return matches[2]
		}
	}

	return ""
}

// GetAppArmorProfile determines the AppArmor profile for a process.
// If pid is less than one, it returns the AppArmor profile for "self".
func GetAppArmorProfile(pid int) string {
	file := "/proc/self/attr/current"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/attr/current", pid)
	}

	f := readFileString(file)
	if f == "" {
		return apparmorUnconfined
	}
	return f
}

// UserMapping holds the values for a {uid,gid}_map.
type UserMapping struct {
	ContainerID int64
	HostID      int64
	Range       int64
}

// GetUserNamespaceInfo determines if the process is running in a UserNamespace
// and returns the mappings if true.
// If pid is less than one, it returns the user namespace info for "self".
func GetUserNamespaceInfo(pid int) (bool, []UserMapping) {
	file := "/proc/self/uid_map"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/uid_map", pid)
	}

	f := readFileString(file)
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

// GetCapabilities returns the allowed capabilities for the process.
// If pid is less than one, it returns the capabilities for "self".
func GetCapabilities(pid int) (map[string][]string, error) {
	allCaps := capability.List()

	caps, err := capability.NewPid(pid)
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

// GetUIDGID returns the uid and gid for a process.
// If pid is less than one, it returns the seccomp enforcing mode for "self".
func GetUIDGID(tgid, pid int) (uint32, uint32, error) {
	file := "/proc/self/status"
	if pid > 0 {
		if tgid > 0 {
			file = fmt.Sprintf("/proc/%d/task/%d/status", tgid, pid)
		} else {
			file = fmt.Sprintf("/proc/%d/status", pid)
		}
	}

	return getUIDGID(readFileString(file))
}

func getUIDGID(input string) (uint32, uint32, error) {
	// Split status file string by line
	statusMappings := strings.Split(input, "\n")
	statusMappings = deleteEmpty(statusMappings)

	var uid, gid string
	for _, line := range statusMappings {
		if strings.Contains(line, "Uid:") {
			matches := statusFileValueRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				uid = matches[1]
				continue
			}
		}
		if strings.Contains(line, "Gid:") {
			matches := statusFileValueRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				gid = matches[1]
				continue
			}
		}
		if len(uid) > 0 && len(gid) > 0 {
			break
		}
	}

	if len(uid) < 1 && len(gid) < 1 {
		return 0, 0, nil
	}

	u, err := strconv.Atoi(strings.Split(strings.Split(strings.TrimSpace(uid), " ")[0], "\t")[0])
	if err != nil {
		return 0, 0, err
	}
	g, err := strconv.Atoi(strings.Split(strings.Split(strings.TrimSpace(gid), " ")[0], "\t")[0])
	if err != nil {
		return 0, 0, err
	}

	return uint32(u), uint32(g), nil
}

// GetSeccompEnforcingMode returns the seccomp enforcing level (disabled, filtering, strict)
// for a process.
// If pid is less than one, it returns the seccomp enforcing mode for "self".
func GetSeccompEnforcingMode(pid int) SeccompMode {
	file := "/proc/self/status"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/status", pid)
	}

	return getSeccompEnforcingMode(readFileString(file))
}

func getSeccompEnforcingMode(input string) SeccompMode {
	mode := getStatusEntry(input, "Seccomp:")
	sm, ok := seccompModes[mode]
	if ok {
		return sm
	}

	// Pre linux 3.8, check if Seccomp is supported, via CONFIG_SECCOMP.
	if err := unix.Prctl(unix.PR_GET_SECCOMP, 0, 0, 0, 0); err != unix.EINVAL {
		// Make sure the kernel has CONFIG_SECCOMP_FILTER.
		if err := unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, 0, 0, 0); err != unix.EINVAL {
			return SeccompModeStrict
		}
	}

	return SeccompModeDisabled
}

// GetNoNewPrivileges returns if no_new_privileges is set
// for a process.
// If pid is less than one, it returns if set for "self".
func GetNoNewPrivileges(pid int) bool {
	file := "/proc/self/status"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/status", pid)
	}

	return getNoNewPrivileges(readFileString(file))
}

func getNoNewPrivileges(input string) bool {
	return getStatusEntry(input, "NoNewPrivs:") == "1"
}

// GetCmdline returns the cmdline for a process.
// If pid is less than one, it returns the cmdline for "self".
func GetCmdline(pid int) []string {
	file := "/proc/self/cmdline"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/cmdline", pid)
	}

	return parseProcFile(readFile(file))
}

// GetEnviron returns the environ for a process.
// If pid is less than one, it returns the environ for "self".
func GetEnviron(pid int) []string {
	file := "/proc/self/environ"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/environ", pid)
	}

	return parseProcFile(readFile(file))
}

// GetCwd returns the current working directory for the process.
// If pid is less than one, it returns the current working directory for "self".
func GetCwd(pid int) string {
	file := "/proc/self/cwd"
	if pid > 0 {
		file = fmt.Sprintf("/proc/%d/cwd", pid)
	}

	cwd, err := os.Readlink(file)
	if err != nil {
		if os.IsPermission(err) {
			// Ignore the permission errors or the logs are noisy.
			return ""
		}
		// Ignore errors in general.
		return ""
	}

	return cwd
}

// TODO: make this function more efficient and read the file line by line.
func getStatusEntry(input, find string) string {
	// Split status file string by line
	statusMappings := strings.Split(input, "\n")
	statusMappings = deleteEmpty(statusMappings)

	for _, line := range statusMappings {
		if strings.Contains(line, find) {
			matches := statusFileValueRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				return strings.TrimSpace(matches[1])
			}
		}
	}

	return ""
}

func fileExists(file string) bool {
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true
	}
	return false
}

func readFile(file string) []byte {
	if !fileExists(file) {
		return nil
	}

	b, _ := ioutil.ReadFile(file)
	return b
}

func readFileString(file string) string {
	b := readFile(file)
	if b == nil {
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

func parseProcFile(data []byte) []string {
	if len(data) < 1 {
		return nil
	}
	if data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	parts := bytes.Split(data, []byte{0})
	var strParts []string
	for _, p := range parts {
		strParts = append(strParts, string(p))
	}

	return strParts
}

// IsValidContainerRuntime checks if a string is a valid container runtime.
func IsValidContainerRuntime(s string) bool {
	for _, b := range ContainerRuntimes {
		if string(b) == s {
			return true
		}
	}
	return false
}

// HasNamespace determines if a container is using a particular namespace or the
// host namespace.
// The device number of an unnamespaced /proc/1/ns/{ns} is 4 and anything else is
// higher.
// Only works from inside a container.
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
