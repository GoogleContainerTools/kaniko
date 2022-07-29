//go:build linux
// +build linux

package chroot

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/syndtr/gocapability/capability"
)

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	stdin, stdout, stderr := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
	type args struct {
		cmd     *exec.Cmd
		newRoot string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple ls -al",
			args: args{
				cmd: &exec.Cmd{
					Path:   "/bin/ls",
					Args:   []string{"ls", "-al"},
					Stdin:  stdin,
					Stdout: stdout,
					Stderr: stderr,
				},
				newRoot: tempDir,
			},
			wantErr: false,
		},
		{
			name: "mount syscall should be denied",
			args: args{
				cmd: &exec.Cmd{
					Path:   "/bin/mount",
					Args:   []string{"mount", "--bind", "/tmp", "/bin"},
					Stdin:  stdin,
					Stdout: stdout,
					Stderr: stderr,
				},
				newRoot: tempDir,
			},
			wantErr: true,
		},
	}

	t.Logf("setup %v", tempDir)
	err := setupNewRoot(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.newRoot = tempDir
			if err = Run(tt.args.cmd, tt.args.newRoot); (err != nil) != tt.wantErr {
				output, errRead := io.ReadAll(stderr)
				if errRead != nil {
					t.Fatalf("can't read stderr: %v", errRead)
				}
				t.Logf("stderr output: %s\n", output)
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			output, err := io.ReadAll(stderr)
			if err != nil {
				t.Fatalf("can't read stderr: %v", err)
			}
			t.Logf("stdout output: %s\n", output)
		})
	}
}

// setupNewRoot will unTar a debian:bullseye-slim filesystem on newRoot
func setupNewRoot(newRoot string) error {
	debianTar := filepath.Join("testdata", "debian.tar")
	// use unix tar because our tar implementation restores file permissions and needs root
	cmd := exec.Command("tar", "xf", debianTar, "-C", newRoot)
	return cmd.Run()
}

func Test_setCapabilities(t *testing.T) {
	test := struct {
		name    string
		wanted  map[capability.CapType][]capability.Cap
		wantErr bool
	}{
		name: "default applied capabilities",
		wanted: map[capability.CapType][]capability.Cap{
			capability.BOUNDING:    defaultCapabilities,
			capability.EFFECTIVE:   defaultCapabilities,
			capability.INHERITABLE: {},
			capability.PERMITTED:   defaultCapabilities,
		},
		wantErr: false,
	}
	if os.Getuid() != 0 {
		t.Skip("calling user is not root, so can't load caps")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := setCapabilities(); (err != nil) != test.wantErr {
		t.Fatalf("setCapabilities() error = %v, wantErr %v", err, test.wantErr)
	}
	// load the current caps
	caps, err := capability.NewPid2(0)
	if err != nil {
		t.Fatal(err)
	}
	err = caps.Load()
	if err != nil {
		t.Fatal(err)
	}
	for capType, capList := range test.wanted {
		for _, cap := range capList {
			if !caps.Get(capType, cap) {
				t.Errorf("cap %s on capType %s is not set but wanted", cap, capType)
			}
		}
	}
	t.Logf(caps.String())
}
