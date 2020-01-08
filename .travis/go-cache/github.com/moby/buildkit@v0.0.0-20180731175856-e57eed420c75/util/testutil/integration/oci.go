package integration

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/shlex"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func init() {
	register(&oci{})

	// the rootless uid is defined in hack/dockerfiles/test.Dockerfile
	if s := os.Getenv("BUILDKIT_INTEGRATION_ROOTLESS_IDPAIR"); s != "" {
		var uid, gid int
		if _, err := fmt.Sscanf(s, "%d:%d", &uid, &gid); err != nil {
			logrus.Fatalf("unexpected BUILDKIT_INTEGRATION_ROOTLESS_IDPAIR: %q", s)
		}
		if rootlessSupported(uid) {
			register(&oci{uid: uid, gid: gid})
		}
	}
}

type oci struct {
	uid int
	gid int
}

func (s *oci) Name() string {
	if s.uid != 0 {
		return "oci-rootless"
	}
	return "oci"
}

func (s *oci) New() (Sandbox, func() error, error) {
	if err := lookupBinary("buildkitd"); err != nil {
		return nil, nil, err
	}
	if err := requireRoot(); err != nil {
		return nil, nil, err
	}
	logs := map[string]*bytes.Buffer{}
	buildkitdArgs := []string{"buildkitd", "--oci-worker=true", "--containerd-worker=false"}
	if s.uid != 0 {
		if s.gid == 0 {
			return nil, nil, errors.Errorf("unsupported id pair: uid=%d, gid=%d", s.uid, s.gid)
		}
		// TODO: make sure the user exists and subuid/subgid are configured.
		buildkitdArgs = append([]string{"sudo", "-u", fmt.Sprintf("#%d", s.uid), "-i", "--", "rootlesskit"}, buildkitdArgs...)
	}
	buildkitdSock, stop, err := runBuildkitd(buildkitdArgs, logs, s.uid, s.gid)
	if err != nil {
		return nil, nil, err
	}

	deferF := &multiCloser{}
	deferF.append(stop)

	return &sandbox{address: buildkitdSock, logs: logs, cleanup: deferF, rootless: s.uid != 0}, deferF.F(), nil
}

type sandbox struct {
	address  string
	logs     map[string]*bytes.Buffer
	cleanup  *multiCloser
	rootless bool
}

func (sb *sandbox) Address() string {
	return sb.address
}

func (sb *sandbox) PrintLogs(t *testing.T) {
	for name, l := range sb.logs {
		t.Log(name)
		s := bufio.NewScanner(l)
		for s.Scan() {
			t.Log(s.Text())
		}
	}
}

func (sb *sandbox) NewRegistry() (string, error) {
	url, cl, err := newRegistry()
	if err != nil {
		return "", err
	}
	sb.cleanup.append(cl)
	return url, nil
}

func (sb *sandbox) Cmd(args ...string) *exec.Cmd {
	if len(args) == 1 {
		if split, err := shlex.Split(args[0]); err == nil {
			args = split
		}
	}
	cmd := exec.Command("buildctl", args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "BUILDKIT_HOST="+sb.Address())
	return cmd
}

func (sb *sandbox) Rootless() bool {
	return sb.rootless
}

func runBuildkitd(args []string, logs map[string]*bytes.Buffer, uid, gid int) (address string, cl func() error, err error) {
	deferF := &multiCloser{}
	cl = deferF.F()

	defer func() {
		if err != nil {
			deferF.F()()
			cl = nil
		}
	}()

	tmpdir, err := ioutil.TempDir("", "bktest_buildkitd")
	if err != nil {
		return "", nil, err
	}
	if err := os.Chown(tmpdir, uid, gid); err != nil {
		return "", nil, err
	}
	deferF.append(func() error { return os.RemoveAll(tmpdir) })

	address = "unix://" + filepath.Join(tmpdir, "buildkitd.sock")
	if runtime.GOOS == "windows" {
		address = "//./pipe/buildkitd-" + filepath.Base(tmpdir)
	}

	args = append(args, "--root", tmpdir, "--addr", address, "--debug")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "BUILDKIT_DEBUG_EXEC_OUTPUT=1")

	if stop, err := startCmd(cmd, logs); err != nil {
		return "", nil, err
	} else {
		deferF.append(stop)
	}

	if err := waitUnix(address, 5*time.Second); err != nil {
		return "", nil, err
	}

	return
}

func rootlessSupported(uid int) bool {
	cmd := exec.Command("sudo", "-u", fmt.Sprintf("#%d", uid), "-i", "--", "unshare", "-U", "true")
	b, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("rootless mode is not supported on this host: %v (%s)", err, string(b))
		return false
	}
	return true
}
