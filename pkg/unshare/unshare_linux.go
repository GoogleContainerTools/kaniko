package unshare

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/GoogleContainerTools/kaniko/pkg/idtools"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
)

const (
	unshareReexecKey = "unshare-reexec"
	continuePipeKey  = "kaniko_continue_pipe"
	pidPipeKey       = "kaniko_pid_pipe"
)

type Cmd struct {
	*exec.Cmd
}

func Command(args ...string) *Cmd {
	return &Cmd{
		Cmd: reexec.Command(args...),
	}
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

	uid := os.Getuid()
	gid := os.Getgid()
	uidmap, gidmap := []idtools.Mapping{}, []idtools.Mapping{}
	if uid != 0 {
		// only create additional mappings if running rootless
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

	err = idtools.SetUidMap(pid, uidmap)
	if err != nil {
		err = fmt.Errorf("apply subuid mappings: %w", err)
		fmt.Fprint(continueWriter, err)
		return err
	}

	err = idtools.SetGidMap(pid, gidmap)
	if err != nil {
		err = fmt.Errorf("apply subgid mappings: %w", err)
		fmt.Fprint(continueWriter, err)
		return err
	}

	// nothing went wrong, so lets continue child
	_, err = fmt.Fprint(continueWriter, "continue")
	if err != nil {
		return fmt.Errorf("writing to child continue pipe: %w", err)
	}
	return nil
}

// ChildWait must be used in any child that is created with Command
func ChildWait() {
	pidStr := fmt.Sprint(os.Getpid())
	logrus.Infof("child pid is %v", pidStr)
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

	logrus.Infof("child process is running with uid %d", os.Getuid())

}

// waitForContinue will block until we read something from the continue pipe.
// This pipe will be used by the parent if it errors or child can continue execution
func waitForContinue() error {
	continuePipe, err := getPipeFromKey(continuePipeKey)
	if err != nil {
		return fmt.Errorf("creating continue pipe: %w", err)
	}
	buf := make([]byte, 1024)
	_, err = continuePipe.Read(buf)
	if err != nil {
		return fmt.Errorf("reading from continue pipe: %w", err)
	}
	logrus.Info("recieved from continue pipe, continue")
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
	logrus.Infof("getting pipe from fd %d", fd)
	return os.NewFile(uintptr(fd), key), nil
}
