//go:build linux
// +build linux

package unshare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/idtools"
	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/docker/docker/pkg/reexec"
)

const (
	reportReexecKey = "report-reexec"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func init() {
	reexec.Register(reportReexecKey, reportMain)
}

func TestUnshareNamespaces(t *testing.T) {
	for name, flag := range namespaces {
		// always create user namespace because we might not be running as root
		c := Command(syscall.CLONE_NEWUSER|flag, reportReexecKey)
		buf := new(bytes.Buffer)
		c.Stderr, c.Stdout = buf, buf

		t.Run(name, func(t *testing.T) {
			err := c.Run()
			if err != nil {
				t.Fatalf("run %q: %v: %s", name, err, buf.String())
			}
			// our namespace links
			ns, err := getNamespaceLinks()
			if err != nil {
				t.Fatalf("getting namespace links: %v", err)
			}
			report := getReport(t, buf.Bytes())
			if report.Namespaces[name] == ns[name] {
				t.Errorf("unshare didn't create a new %v namespace", name)
			}
		})
	}
}

func TestUnshareIDMappings(t *testing.T) {
	tests := []struct {
		name         string
		unshareFlags int
		want         report
	}{
		{
			name: "no new namespace",
			want: func() report {
				var r report
				var err error
				r.Uidmap, r.Gidmap, err = idtools.GetHostIDMappings("")
				if err != nil {
					t.Fatalf("getting hostid mappings: %v", err)
				}
				r.Uid = uint32(os.Getuid())
				return r
			}(),
		},
		{
			name:         "user namespace",
			unshareFlags: syscall.CLONE_NEWUSER,
			want: func() report {
				var r report
				r.Uidmap, r.Gidmap = expectedRootlessMappings(t)
				// when using user namespace we want to be root inside there
				r.Uid = 0
				return r
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Command(tt.unshareFlags, reportReexecKey)
			buf := new(bytes.Buffer)
			c.Stderr, c.Stdout = buf, buf

			err := c.Run()
			if err != nil {
				t.Fatalf("run %q: %v: %s", tt.name, err, buf.String())
			}
			report := getReport(t, buf.Bytes())
			testutil.CheckDeepEqual(t, tt.want.Gidmap, report.Gidmap)
			testutil.CheckDeepEqual(t, tt.want.Uidmap, report.Uidmap)
			testutil.CheckDeepEqual(t, tt.want.Uid, report.Uid)
		})
	}
}

func getReport(t *testing.T, data []byte) report {
	var report report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("error parsing results: %v", err)
	}
	return report
}

// check which namespaces we are in
func getNamespaceLinks() (map[string]string, error) {
	found := map[string]string{}
	for name := range namespaces {
		linkTarget, err := os.Readlink("/proc/self/ns/" + name)
		if err != nil {
			return nil, fmt.Errorf("reading link /proc/self/ns/%s: %w", name, err)
		}
		found[name] = linkTarget
	}
	return found, nil
}

func expectedRootlessMappings(t *testing.T) ([]idtools.Mapping, []idtools.Mapping) {
	u := testutil.GetCurrentUser(t)
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		t.Errorf("converting uid to int: %v", err)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		t.Errorf("converting gid to int: %v", err)
	}

	uidmap, gidmap, err := idtools.GetSubIDMappings(uint32(uid), uint32(gid), u.Username, u.PrimaryGroup)
	if err != nil {
		t.Errorf("getting subuid mappings: %v", err)
	}
	uidmap = append([]idtools.Mapping{{HostID: uint32(uid), ContainerID: 0, Size: 1}}, uidmap...)
	gidmap = append([]idtools.Mapping{{HostID: uint32(gid), ContainerID: 0, Size: 1}}, gidmap...)

	return uidmap, gidmap
}

type report struct {
	Uidmap     []idtools.Mapping
	Gidmap     []idtools.Mapping
	Uid        uint32
	Namespaces map[string]string
}

var (
	namespaces = map[string]int{
		"ipc":  syscall.CLONE_NEWIPC,
		"net":  syscall.CLONE_NEWNET,
		"mnt":  syscall.CLONE_NEWNS,
		"user": syscall.CLONE_NEWUSER,
		"uts":  syscall.CLONE_NEWUTS,
	}
)

// reportMain will collect information about the unshared environment
// and write a report into a pipe for later use.
func reportMain() {
	uidmap, gidmap, err := idtools.GetHostIDMappings("")
	if err != nil {
		fmt.Printf("error getting hostIDMappings: %v", err)
		os.Exit(1)
	}

	ns, err := getNamespaceLinks()
	if err != nil {
		fmt.Printf("error getting namespace links: %v", err)
		os.Exit(1)
	}

	r := report{
		Uidmap:     uidmap,
		Gidmap:     gidmap,
		Uid:        uint32(os.Getuid()),
		Namespaces: ns,
	}

	err = json.NewEncoder(os.Stdout).Encode(r)
	if err != nil {
		fmt.Printf("error writing reportData to pipe: %v", err)
		os.Exit(1)
	}
}
