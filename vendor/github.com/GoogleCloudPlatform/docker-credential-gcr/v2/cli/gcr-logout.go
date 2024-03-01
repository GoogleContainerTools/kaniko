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

	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/store"
	"github.com/google/subcommands"
)

type logoutCmd struct {
	cmd
}

// NewGCRLogoutSubcommand returns a subcommands.Command which implements the GCR
// logout operation.
func NewGCRLogoutSubcommand() subcommands.Command {
	return &logoutCmd{
		cmd{
			name:     "gcr-logout",
			synopsis: "log out from GCR",
		},
	}
}

func (c *logoutCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	if err := c.GCRLogout(); err != nil {
		fmt.Fprintf(os.Stderr, "Logout failure: %v\n", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// GCRLogout performs the actions necessary to remove any GCR credentials
// from the credential store.
func (*logoutCmd) GCRLogout() error {
	s, err := store.DefaultGCRCredStore()
	if err != nil {
		return err
	}
	return s.DeleteGCRAuth()
}
