package integration

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func startCmd(cmd *exec.Cmd, logs map[string]*bytes.Buffer) (func() error, error) {
	if logs != nil {
		b := new(bytes.Buffer)
		logs["stdout: "+cmd.Path] = b
		cmd.Stdout = b
		b = new(bytes.Buffer)
		logs["stderr: "+cmd.Path] = b
		cmd.Stderr = b

	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	eg, ctx := errgroup.WithContext(context.TODO())

	stopped := make(chan struct{})
	stop := make(chan struct{})
	eg.Go(func() error {
		_, err := cmd.Process.Wait()
		close(stopped)
		select {
		case <-stop:
			return nil
		default:
			return err
		}
	})

	eg.Go(func() error {
		select {
		case <-ctx.Done():
		case <-stopped:
		case <-stop:
			cmd.Process.Signal(syscall.SIGTERM)
			go func() {
				select {
				case <-stopped:
				case <-time.After(20 * time.Second):
					cmd.Process.Kill()
				}
			}()
		}
		return nil
	})

	return func() error {
		close(stop)
		return eg.Wait()
	}, nil
}

func waitUnix(address string, d time.Duration) error {
	address = strings.TrimPrefix(address, "unix://")
	addr, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		return err
	}

	step := 50 * time.Millisecond
	i := 0
	for {
		if conn, err := net.DialUnix("unix", nil, addr); err == nil {
			conn.Close()
			break
		}
		i++
		if time.Duration(i)*step > d {
			return errors.Errorf("failed dialing: %s", address)
		}
		time.Sleep(step)
	}
	return nil
}

type multiCloser struct {
	fns []func() error
}

func (mc *multiCloser) F() func() error {
	return func() error {
		var err error
		for i := range mc.fns {
			if err1 := mc.fns[len(mc.fns)-1-i](); err == nil {
				err = err1
			}
		}
		mc.fns = nil
		return err
	}
}

func (mc *multiCloser) append(f func() error) {
	mc.fns = append(mc.fns, f)
}

var ErrorRequirements = errors.Errorf("missing requirements")

func lookupBinary(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return errors.Wrapf(ErrorRequirements, "failed to lookup %s binary", name)
	}
	return nil
}

func requireRoot() error {
	if os.Getuid() != 0 {
		return errors.Wrap(ErrorRequirements, "requires root")
	}
	return nil
}
