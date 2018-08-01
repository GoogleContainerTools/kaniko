/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RunCommand struct {
	cmd *instructions.RunCommand
}

func (r *RunCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	var newCommand []string
	if r.cmd.PrependShell {
		// This is the default shell on Linux
		var shell []string
		if len(config.Shell) > 0 {
			shell = config.Shell
		} else {
			shell = append(shell, "/bin/sh", "-c")
		}

		newCommand = append(shell, strings.Join(r.cmd.CmdLine, " "))
	} else {
		newCommand = r.cmd.CmdLine
	}

	logrus.Infof("cmd: %s", newCommand[0])
	logrus.Infof("args: %s", newCommand[1:])

	cmd := exec.Command(newCommand[0], newCommand[1:]...)
	cmd.Dir = config.WorkingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	cmd.Env = replacementEnvs

	// If specified, run the command as a specific user
	if config.User != "" {
		userAndGroup := strings.Split(config.User, ":")
		userStr := userAndGroup[0]
		var groupStr string
		if len(userAndGroup) > 1 {
			groupStr = userAndGroup[1]
		}

		uidStr, gidStr, err := util.GetUserFromUsername(userStr, groupStr)
		if err != nil {
			return err
		}

		// uid and gid need to be uint32
		uid64, err := strconv.ParseUint(uidStr, 10, 32)
		if err != nil {
			return err
		}
		uid := uint32(uid64)
		var gid uint32
		if gidStr != "" {
			gid64, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return err
			}
			gid = uint32(gid64)
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting command")
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return errors.Wrap(err, "getting group id for process")
	}

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, "waiting for process to exit")
	}

	//it's not an error if there are no grandchildren
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && err.Error() != "no such process" {
		return err
	}

	return nil
}

// FilesToSnapshot returns nil for this command because we don't know which files
// have changed, so we snapshot the entire system.
func (r *RunCommand) FilesToSnapshot() []string {
	return nil
}

// CreatedBy returns some information about the command for the image config
func (r *RunCommand) CreatedBy() string {
	cmdLine := strings.Join(r.cmd.CmdLine, " ")
	if r.cmd.PrependShell {
		// TODO: Support shell command here
		shell := []string{"/bin/sh", "-c"}
		return strings.Join(append(shell, cmdLine), " ")
	}
	return cmdLine
}
