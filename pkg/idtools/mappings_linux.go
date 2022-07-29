//go:build linux
// +build linux

package idtools

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/syndtr/gocapability/capability"
)

func hasSetID(path string, modeid os.FileMode, capid capability.Cap) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	mode := info.Mode()
	if mode&modeid == modeid {
		return true, nil
	}
	cap, err := capability.NewFile2(path)
	if err != nil {
		return false, err
	}
	if err := cap.Load(); err != nil {
		return false, err
	}
	return cap.Get(capability.EFFECTIVE, capid), nil
}

// SetUidMap executes newuidmap with mapping defined in uidmap
func SetUidMap(pid int, uidmap []Mapping) error {
	path, err := exec.LookPath("newuidmap")
	if err != nil {
		return fmt.Errorf("finding newgidmap: %w", err)
	}
	err = runNewIDMap(
		path,
		fmt.Sprintf("%d", pid),
		uidmap,
	)
	if err != nil {
		ok, err := hasSetID(path, os.ModeSetuid, capability.CAP_SETUID)
		if err != nil {
			return fmt.Errorf("determining if %v has setuid cap: %w", path, err)
		}
		if !ok {
			return fmt.Errorf("%v failed because setuid was not set on the file nor had the capabiltity", path)
		}
	}
	return nil
}

// SetUidMap executes newgidmap with mapping defined in gidmap
func SetGidMap(pid int, gidmap []Mapping) error {
	path, err := exec.LookPath("newgidmap")
	if err != nil {
		return fmt.Errorf("finding newgidmap: %w", err)
	}
	err = runNewIDMap(
		path,
		fmt.Sprintf("%d", pid),
		gidmap,
	)
	if err != nil {
		ok, err := hasSetID(path, os.ModeSetgid, capability.CAP_SETGID)
		if err != nil {
			return fmt.Errorf("determining if %v has Setgid cap: %w", path, err)
		}
		if !ok {
			return fmt.Errorf("%v failed because Setgid was not set on the file", path)
		}
	}
	return nil
}

func runNewIDMap(path, pid string, mappings []Mapping) error {
	// newuidmap and newgidmap are only allowed once per process
	mappingBuffer := new(bytes.Buffer)
	for _, m := range mappings {
		mStr := fmt.Sprintf("%d %d %d ", m.ContainerID, m.HostID, m.Size)
		logrus.Infof("args for %v: %s %s", path, pid, mStr)
		fmt.Fprint(mappingBuffer, mStr)
	}
	args := []string{
		pid,
	}
	args = append(args, strings.Fields(mappingBuffer.String())...)
	cmd := exec.Command(path, args...)

	output := new(bytes.Buffer)
	cmd.Stdout, cmd.Stderr = output, output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v failed: %s %w", path, output.String(), err)
	}
	return nil
}

// getHostIDMappings reads mappings from the named node under /proc.
func getHostIDMappings(path string) ([]Mapping, error) {
	var mappings []Mapping
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading ID mappings from %q: %w", path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("line %q from %q has %d fields, not 3", line, path, len(fields))
		}
		cid, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing container ID value %q from line %q in %q: %w", fields[0], line, path, err)
		}
		hid, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing host ID value %q from line %q in %q: %w", fields[1], line, path, err)
		}
		size, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing size value %q from line %q in %q: %w", fields[2], line, path, err)
		}
		mappings = append(mappings, Mapping{ContainerID: uint32(cid), HostID: uint32(hid), Size: uint32(size)})
	}
	return mappings, nil
}

// GetHostIDMappings reads mappings for the specified process (or the current
// process if pid is "self" or an empty string) from the kernel.
func GetHostIDMappings(pid string) ([]Mapping, []Mapping, error) {
	if pid == "" {
		pid = "self"
	}
	uidmapPath := fmt.Sprintf("/proc/%v/uid_map", pid)
	uidmap, err := getHostIDMappings(uidmapPath)
	if err != nil {
		return nil, nil, err
	}
	gidmapPath := fmt.Sprintf("/proc/%v/gid_map", pid)
	gidmap, err := getHostIDMappings(gidmapPath)
	if err != nil {
		return nil, nil, err
	}
	return uidmap, gidmap, nil
}

// GetSubIDMappings reads mappings from /etc/subuid and /etc/subgid.
func GetSubIDMappings(uid, gid uint32, user, group string) ([]Mapping, []Mapping, error) {
	return newIDMappings(uid, gid, user, group)
}

func newIDMappings(uid, gid uint32, username, group string) (uidmap []Mapping, gidmap []Mapping, err error) {
	uidFile, err := os.Open(subgidFile)
	if err != nil {
		return uidmap, gidmap, err
	}
	defer uidFile.Close()
	uidmap, err = getMappingFromSubFile(uid, username, uidFile)
	if err != nil {
		return uidmap, gidmap, fmt.Errorf("get mapping from %v for user %v: %w", subuidFile, username, err)
	}

	gidFile, err := os.Open(subgidFile)
	if err != nil {
		return uidmap, gidmap, err
	}
	defer gidFile.Close()
	gidmap, err = getMappingFromSubFile(gid, group, gidFile)
	if err != nil {
		return uidmap, gidmap, fmt.Errorf("get mapping from %v for user %v: %w", subuidFile, username, err)
	}
	return
}

func getMappingFromSubFile(uidOrGid uint32, userOrGroup string, r io.Reader) ([]Mapping, error) {
	// /etc/sub{uid,gid} is of the following format
	// USERNAME_OR_GROUP:START_UID_IN_USERNAMESPACE:SIZE
	scanner := bufio.NewScanner(r)
	maps := []Mapping{}
	for {
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if line == "" {
			// skip empty lines
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			return nil, errors.New("content of reader is in wrong format")
		}
		if parts[0] == userOrGroup || userOrGroup == "ALL" {
			containerID, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}
			size, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}
			maps = append(maps, Mapping{
				HostID:      uint32(containerID),
				ContainerID: uidOrGid,
				Size:        uint32(size),
			})
		}
	}
	return maps, nil
}
