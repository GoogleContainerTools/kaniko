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

type clearCmd struct {
	cmd
}

// NewClearSubcommand returns a subcommands.Command which removes all stored
// credentials.
func NewClearSubcommand() subcommands.Command {
	return &clearCmd{
		cmd{
			name:     "clear",
			synopsis: "remove all stored credentials",
		},
	}
}

func (c *clearCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	if err := c.ClearAll(); err != nil {
		fmt.Fprintf(os.Stderr, "failure: %v\n", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// ClearAll removes all credentials from the store (GCR or otherwise).
func (c *clearCmd) ClearAll() error {
	s, err := store.DefaultGCRCredStore()
	if err != nil {
		return err
	}

	return s.DeleteGCRAuth()
}
