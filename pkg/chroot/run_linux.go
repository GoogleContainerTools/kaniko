//go:build linux
// +build linux

package chroot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/unshare"
	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	parentProcess = "chroot-parent-process"
	childProcess  = "chroot-child-process"
	confPipeKey   = "kaniko_conf_pipe"
)

func init() {
	reexec.Register(parentProcess, runParentProcessMain)
	reexec.Register(childProcess, runChildProcessMain)
	// when a reexec main was invoked, exit immediately
	if reexec.Init() {
		os.Exit(0)
	}
}

// cmd is exec.Cmd without io.Reader and io.Writer fields
// cmd is exec.Cmd without io.Reader and io.Writer fields
type cmd struct {
	Path    string               `json:"path,omitempty"`
	Args    []string             `json:"args,omitempty"`
	Env     []string             `json:"env,omitempty"`
	SysAttr *syscall.SysProcAttr `json:"sys_attr,omitempty"`
	Dir     string               `json:"dir,omitempty"`
}

func execCmdToCmd(execCmd *exec.Cmd) *cmd {
	return &cmd{
		Path:    execCmd.Path,
		Args:    execCmd.Args,
		Env:     execCmd.Env,
		SysAttr: execCmd.SysProcAttr,
		Dir:     execCmd.Dir,
	}
}

func cmdToExecCmd(cmd *cmd) *exec.Cmd {
	return &exec.Cmd{
		Path:        cmd.Path,
		Args:        cmd.Args,
		Env:         cmd.Env,
		SysProcAttr: cmd.SysAttr,
		Dir:         cmd.Dir,
		// set std{in,out,err} to os versions because they didn't get marshaled
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

type config struct {
	Cmd     *cmd   `json:"cmd,omitempty"`
	NewRoot string `json:"new_root,omitempty"`
}

// Run will execute the cmd inside a chrooted and newly created namespace environment
func Run(cmd *exec.Cmd, newRoot string) error {

	// lockOSThread because changing the thread would kick us out of the namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Create a pipe for passing configuration down to the next process.
	confReader, confWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creating configuration pipes: %w", err)
	}
	defer confReader.Close()
	defer confWriter.Close()

	// marshal config for communication with subprocess
	c := config{
		Cmd:     execCmdToCmd(cmd),
		NewRoot: newRoot,
	}

	unshareCmd := unshare.Command(parentProcess)

	unshareCmd.Stderr, unshareCmd.Stdout, unshareCmd.Stdin = os.Stderr, os.Stdout, os.Stdin
	sysProcAttr := unshareCmd.SysProcAttr
	if sysProcAttr == nil {
		sysProcAttr = &syscall.SysProcAttr{}
	}
	sysProcAttr.Pdeathsig = syscall.SIGKILL
	sysProcAttr.Cloneflags = syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS

	err = copyConfigIntoPipeAndStartChild(unshareCmd, &c, confReader, confWriter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running unshare cmd: %v\n", err)
		os.Exit(1)
	}
	err = unshareCmd.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error waiting for unshare cmd: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// runParentProcessMain will create all needed mounts, pivot_root and execute the child
func runParentProcessMain() {
	// lockOSThread because changing the thread would kick us out of the namespaces
	// don't unlock because this function will only be called in a new process
	runtime.LockOSThread()
	// wait for child to become ready
	unshare.ChildWait()

	c, err := unmarshalConfigFromPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error unmarshal config from pipe: %v\n", err)
		os.Exit(1)
	}
	// create mounts for pivot_root
	undo, err := prepareMounts(c.NewRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating mounts: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		logrus.Debug("undo mounting of chroot isolation")
		undoErr := undo()
		if undoErr != nil {
			fmt.Fprintf(os.Stderr, "error undo mounting: %s\n", undoErr)
			os.Exit(1)
		}
	}()

	err = pivotRoot(c.NewRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Create a pipe for passing configuration down to the next process.
	confReader, confWriter, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating configuration pipe: %v\n", err)
		os.Exit(1)
	}
	defer confWriter.Close()
	defer confReader.Close()

	childCmd := unshare.Command(childProcess)

	childCmd.Stderr, childCmd.Stdout, childCmd.Stdin = os.Stderr, os.Stdout, os.Stdin
	sysProcAttr := childCmd.SysProcAttr
	if sysProcAttr == nil {
		sysProcAttr = &syscall.SysProcAttr{}
	}
	sysProcAttr.Pdeathsig = syscall.SIGKILL
	// delay pid namespace until here, because pid would be wrong otherwise
	sysProcAttr.Cloneflags = syscall.CLONE_NEWPID

	err = copyConfigIntoPipeAndStartChild(childCmd, &c, confReader, confWriter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running child: %v\n", err)
		os.Exit(1)
	}
	childCmd.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error waiting for child: %v\n", err)
		os.Exit(1)
	}

}

// runChildProcess will set capabilities and execute the initial cmd
// TODO: add apparmor and seccomp profiles
func runChildProcessMain() {
	runtime.LockOSThread()
	unshare.ChildWait()

	c, err := unmarshalConfigFromPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error unmarshal config from pipe: %v\n", err)
		os.Exit(1)
	}

	err = setCapabilities()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting capabilities: %v\n", err)
		os.Exit(1)
	}
	cmd := cmdToExecCmd(c.Cmd)
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running original command: %v\n", err)
		os.Exit(1)
	}
}

// copyConfigIntoPipeAndStartChild will marshal the config into a pipe which will be passed into child.
// After that, the child will start, but not wait for it.
func copyConfigIntoPipeAndStartChild(child *unshare.Cmd, conf *config, confReader, confWriter *os.File) error {
	// marshal config for communication with subprocess
	confData, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("marshaling configuration: %w", err)
	}

	child.Env = append(child.Env, fmt.Sprintf("%s=%d", confPipeKey, len(child.ExtraFiles)+3))
	child.ExtraFiles = append(child.ExtraFiles, confReader)

	err = child.Start()
	if err != nil {
		return fmt.Errorf("starting child process: %w", err)
	}
	_, err = io.Copy(confWriter, bytes.NewReader(confData))
	if err != nil {
		return fmt.Errorf("copy configuration to pipe: %w", err)
	}
	return nil
}

func unmarshalConfigFromPipe() (config, error) {
	fdStr := os.Getenv(confPipeKey)
	if fdStr == "" {
		return config{}, fmt.Errorf("%v is not set, can't create pipe", confPipeKey)
	}
	fd, err := strconv.Atoi(fdStr)
	if err != nil {
		return config{}, fmt.Errorf("converting %v to integer: %w", fdStr, err)
	}
	confPipe := os.NewFile(uintptr(fd), confPipeKey)
	defer confPipe.Close()
	var c config
	err = json.NewDecoder(confPipe).Decode(&c)
	if err != nil {
		return c, fmt.Errorf("decoding cmd config: %v", err)
	}
	return c, nil
}

func pivotRoot(newRoot string) error {
	err := unix.Chdir(newRoot)
	if err != nil {
		return fmt.Errorf("chdir to newRoot: %w", err)
	}
	err = unix.PivotRoot(newRoot, newRoot)
	if err != nil {
		return fmt.Errorf("syscall pivot_root: %w", err)
	}
	err = unmount(".")
	if err != nil {
		return fmt.Errorf("unmounting newRoot after pivot_root: %w", err)
	}
	return nil
}

func prepareMounts(newRoot string, additionalMounts ...string) (undoMount func() error, err error) {
	bindFlags := uintptr(unix.MS_BIND | unix.MS_REC | unix.MS_PRIVATE)
	devFlags := bindFlags | unix.MS_NOEXEC | unix.MS_NOSUID | unix.MS_RDONLY
	sysFlags := devFlags | unix.MS_NODEV
	type mountOpts struct {
		flags     uintptr
		mountType string
	}
	mounts := map[string]mountOpts{
		"/etc/resolv.conf": {flags: unix.MS_RDONLY | bindFlags},
		"/etc/hostname":    {flags: unix.MS_RDONLY | bindFlags},
		"/etc/hosts":       {flags: unix.MS_RDONLY | bindFlags},
		"/dev":             {flags: devFlags},
		"/sys":             {flags: sysFlags},
	}
	for _, add := range additionalMounts {
		mounts[add] = mountOpts{flags: bindFlags}
	}

	for src, opts := range mounts {
		srcinfo, err := os.Lstat(src)
		if err != nil {
			return nil, fmt.Errorf("src %v for mount doesn't exist: %w", src, err)
		}
		dest := filepath.Join(newRoot, src)
		err = createDest(srcinfo, dest)
		if err != nil {
			return nil, fmt.Errorf("creating dest %v: %w", dest, err)
		}
		err = mount(src, dest, opts.mountType, opts.flags)
		if err != nil {
			return nil, err
		}
	}
	// self mount newRoot for pivot_root
	// unmount will happen after pivot_root is called
	err = mount(newRoot, newRoot, "", bindFlags)
	if err != nil {
		return nil, err
	}

	undoMount = func() error {
		for src := range mounts {
			logrus.Debugf("unmounting %v", src)
			err := unmount(src)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return undoMount, nil
}

func unmount(dest string) error {
	// perform lazy detaching if bind mount
	err := unix.Unmount(dest, unix.MNT_DETACH)
	if err != nil {
		retries := 0
		for (err == unix.EBUSY || err == unix.EAGAIN) && retries < 50 {
			time.Sleep(50 * time.Millisecond)
			err = unix.Unmount(dest, unix.MNT_DETACH)
			retries++
		}
		if err != nil {
			return fmt.Errorf("unmounting %q (retried %d times): %v", dest, retries, err)
		}
	}
	return nil
}

func mount(src, dest, mountType string, flags uintptr) error {
	logrus.Debugf("mounting %v to %v", src, dest)
	err := unix.Mount(src, dest, mountType, uintptr(flags), "")
	if err != nil {
		return fmt.Errorf("mounting %v to %v: %w", src, dest, err)
	}
	return nil
}

func createDest(srcinfo fs.FileInfo, dest string) error {
	// Check if target is a symlink
	_, err := os.Lstat(dest)
	if err != nil {
		// If the target can't be stat()ted, check the error.
		if !os.IsNotExist(err) {
			return fmt.Errorf("error examining %q for mounting: %w", dest, err)
		}
		// The target isn't there yet, so create it.
		if srcinfo.IsDir() {
			if err = os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("error creating mountpoint %q in mount namespace: %w", dest, err)
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return fmt.Errorf("error ensuring parent of mountpoint %q (%q) is present in new root: %w", dest, filepath.Dir(dest), err)
			}
			var file *os.File
			if file, err = os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
				return fmt.Errorf("error creating mountpoint %q: %w", dest, err)
			}
			file.Close()
		}
	}
	return nil
}
