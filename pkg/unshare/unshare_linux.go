//go:build linux
// +build linux

package unshare

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"

	"github.com/GoogleContainerTools/kaniko/pkg/idtools"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/pkg/reexec"
)

const (
	unshareReexecKey = "unshare-reexec"
	continuePipeKey  = "_kaniko_continue_pipe"
	pidPipeKey       = "_kaniko_pid_pipe"
	// insideUnshareCommandKey will be set to every command that is run by unshare.Command.
	// this will make sure, that childWait() will be executed in child.
	insideUnshareCommandKey = "_kaniko_unshare_command"
)

type Cmd struct {
	*exec.Cmd
}

// Command will create a new Cmd with a reexec.Command set to args.
//
// Also set SysProcAttr.UnshareFlags to unshareFlags.
// Use 0 if you don't want to create any namespaces.
//
// Make sure that reexec.Init() will be called in your program.
func Command(unshareFlags int, args ...string) *Cmd {
	c := &Cmd{
		Cmd: reexec.Command(args...),
	}
	// SysProcAttr will always be created from reexec.Command()
	// so don't worry about nil pointer
	c.SysProcAttr.Unshareflags = uintptr(unshareFlags)
	return c
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) Start() error {
	// Create a pipe for getting the pid of child.
	// Use this method instead of checking in the parent, because we wouldn't
	// know when the child is ready
	pidReader, pidWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creating pid pipes: %w", err)
	}
	defer pidReader.Close()
	defer func() {
		if pidWriter != nil {
			pidWriter.Close()
		}
	}()

	// Create a pipe signaling the child to continue
	// Child will wait until something is sent over this pipe
	continueReader, continueWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creating pid pipes: %w", err)
	}
	defer func() {
		if continueReader != nil {
			continueReader.Close()
		}
	}()
	defer continueWriter.Close()

	// create env before appending files, because len(ExtraFIles) would be wrong otherwise
	c.Env = append(c.Env, fmt.Sprintf("%s=%d", pidPipeKey, len(c.ExtraFiles)+3))
	c.ExtraFiles = append(c.ExtraFiles, pidWriter)
	c.Env = append(c.Env, fmt.Sprintf("%s=%d", continuePipeKey, len(c.ExtraFiles)+3))
	c.ExtraFiles = append(c.ExtraFiles, continueReader)

	// set insideUnshareCommandKey to signal child that it needs to execute childWait()
	c.Env = append(c.Env, fmt.Sprintf("%s=%d", insideUnshareCommandKey, 1))

	// Start the new process.
	err = c.Cmd.Start()
	if err != nil {
		return err
	}

	// Close the ends of the pipes that the parent doesn't need.
	continueReader.Close()
	continueReader = nil
	pidWriter.Close()
	pidWriter = nil

	pidbuf := make([]byte, 8)

	n, err := pidReader.Read(pidbuf)
	if err != nil {
		err = fmt.Errorf("reading pid from child pipe: %w", err)
		fmt.Fprint(continueWriter, err)
		return err
	}

	pid, err := strconv.Atoi(string(pidbuf[:n]))
	if err != nil {
		err = fmt.Errorf("converting pid from child to integer: %w", err)
		fmt.Fprint(continueWriter, err)
		return err
	}

	// only create additional mappings if creating user namespace
	if c.SysProcAttr.Unshareflags&syscall.CLONE_NEWUSER != 0 {
		uid := os.Getuid()
		gid := os.Getgid()

		uidmap, gidmap := []idtools.Mapping{}, []idtools.Mapping{}
		if uid != 0 {
			u, err := util.LookupUser("/", fmt.Sprint(uid))
			if err != nil {
				return fmt.Errorf("lookup user for %v: %w", uid, err)
			}

			group, err := util.LookupGroup("/", fmt.Sprint(gid))
			if err != nil {
				return fmt.Errorf("lookup group for %v: %w", gid, err)
			}

			uidmap, gidmap, err = idtools.GetSubIDMappings(uint32(uid), uint32(gid), u.Username, group.Name)
			if err != nil {
				return fmt.Errorf("getting subid mappings: %w", err)
			}

			// Map our UID and GID, then the subuid and subgid ranges,
			// consecutively, starting at 0, to get the mappings to use for
			// a copy of ourselves.
			uidmap = append([]idtools.Mapping{{HostID: uint32(uid), ContainerID: 0, Size: 1}}, uidmap...)
			gidmap = append([]idtools.Mapping{{HostID: uint32(gid), ContainerID: 0, Size: 1}}, gidmap...)
		} else {
			// Read the set of ID mappings that we're currently using.
			uidmap, gidmap, err = idtools.GetHostIDMappings("")
			if err != nil {
				return fmt.Errorf("getting hostID mappings: %w", err)
			}
		}

		if err = idtools.SetUidMap(pid, uidmap); err != nil {
			err = fmt.Errorf("apply subuid mappings: %w", err)
			fmt.Fprint(continueWriter, err)
			return err
		}

		// disable ability of process pid to call setgroups() syscall
		if err = writeToSetGroups(pid, "deny"); err != nil {
			err = fmt.Errorf("write deny to setgroups: %w", err)
			fmt.Fprint(continueWriter, err)
			return err
		}

		if err = idtools.SetGidMap(pid, gidmap); err != nil {
			err = fmt.Errorf("apply subgid mappings: %w", err)
			fmt.Fprint(continueWriter, err)
			return err
		}
	}

	// nothing went wrong, so lets continue child
	_, err = fmt.Fprint(continueWriter, "continue")
	if err != nil {
		return fmt.Errorf("writing to child continue pipe: %w", err)
	}
	return nil
}

// childWait will be executed before any unshare Command is run.
func childWait() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// early return if we were not called from unshare.Command()
	if os.Getenv(insideUnshareCommandKey) == "" {
		return
	}

	pidStr := fmt.Sprint(os.Getpid())
	pidPipe, err := getPipeFromKey(pidPipeKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting pid pipe: %v\n", err)
		os.Exit(1)
	}
	defer pidPipe.Close()

	_, err = io.WriteString(pidPipe, pidStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing pid to pidpipe: %v\n", err)
		os.Exit(1)
	}

	err = waitForContinue()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing pid to pidpipe: %v\n", err)
		os.Exit(1)
	}
}

// writeSetGroup will write val to /proc/PID/setgroups
//
// Since Linux 3.19 unprivileged writing of /proc/self/gid_map
// has been disabled unless /proc/self/setgroups is written
// first to permanently disable the ability to call setgroups
// in that user namespace.
func writeToSetGroups(pid int, val string) error {
	path := fmt.Sprintf("/proc/%d/setgroups", pid)
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(val))
	if err != nil {
		return fmt.Errorf("writing %v to %v: %w", val, path, err)
	}
	return nil
}

// waitForContinue will block until we read something from the continue pipe.
// This pipe will be used by the parent if it errors or child can continue execution
func waitForContinue() error {
	continuePipe, err := getPipeFromKey(continuePipeKey)
	if err != nil {
		return fmt.Errorf("creating continue pipe: %w", err)
	}
	defer continuePipe.Close()
	buf := make([]byte, 1024)
	// use read instead of readall because pipe wont send EOF
	_, err = continuePipe.Read(buf)
	if err != nil {
		return fmt.Errorf("reading from continue pipe: %w", err)
	}
	return nil
}

func getPipeFromKey(key string) (*os.File, error) {
	fdStr := os.Getenv(key)
	if fdStr == "" {
		return nil, fmt.Errorf("%v is not set, can't create pipe", key)
	}
	fd, err := strconv.Atoi(fdStr)
	if err != nil {
		return nil, fmt.Errorf("converting %v to integer: %w", fdStr, err)
	}
	return os.NewFile(uintptr(fd), key), nil
}

func init() {
	// childWait will always be executed on startup
	childWait()
}
