// Copyright 2016 Google, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/auth"
	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/store"
	"github.com/google/subcommands"
)

type loginCmd struct {
	cmd
}

// NewGCRLoginSubcommand returns a subcommands.Command which implements the GCR
// login operation.
func NewGCRLoginSubcommand() subcommands.Command {
	return &loginCmd{
		cmd{
			name:     "gcr-login",
			synopsis: "log in to GCR",
		},
	}
}

func (c *loginCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	if err := c.GCRLogin(); err != nil {
		fmt.Fprintf(os.Stderr, "Login failure: %v\n", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// GCRLogin performs the actions necessary to generate a GCR access token
// and persist it for later use.
func (c *loginCmd) GCRLogin() error {
	loginAgent := &auth.GCRLoginAgent{}
	s, err := store.DefaultGCRCredStore()
	if err != nil {
		return err
	}

	tok, err := loginAgent.PerformLogin()
	if err != nil {
		return fmt.Errorf("unable to authenticate user: %v", err)
	}

	if err = s.SetGCRAuth(tok); err != nil {
		return fmt.Errorf("unable to persist access token: %v", err)
	}

	return nil
}
