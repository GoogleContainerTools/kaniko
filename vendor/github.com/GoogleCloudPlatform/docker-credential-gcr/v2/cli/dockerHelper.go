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

	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/config"
	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/credhelper"
	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/store"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/google/subcommands"
)

type helperCmd struct {
	cmd
}

func (*helperCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	store, err := store.DefaultGCRCredStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: %v\n", err)
		return subcommands.ExitFailure
	}
	userCfg, err := config.LoadUserConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failure: %v\n", err)
		return subcommands.ExitFailure
	}

	credentials.Serve(credhelper.NewGCRCredentialHelper(store, userCfg))
	return subcommands.ExitSuccess
}

// NewStoreSubcommand returns a subcommands.Command which implements the Docker
// credential store 'store' API.
func NewStoreSubcommand() subcommands.Command {
	return &helperCmd{
		cmd{
			name:     "store",
			synopsis: "(UNIMPLEMENTED) for the specified server, store the credentials provided via stdin",
		},
	}
}

// NewGetSubcommand returns a subcommands.Command which implements the Docker
// credential store 'get' API.
func NewGetSubcommand() subcommands.Command {
	return &helperCmd{
		cmd{
			name:     "get",
			synopsis: "for the server specified via stdin, return the stored credentials via stdout",
		},
	}
}

// NewEraseSubcommand returns a subcommands.Command which implements the Docker
// credential store 'erase' API.
func NewEraseSubcommand() subcommands.Command {
	return &helperCmd{
		cmd{
			name:     "erase",
			synopsis: "(UNIMPLEMENTED) erase any stored credentials for the server specified via stdin",
		},
	}
}

// NewListSubcommand returns a subcommands.Command which implements the Docker
// credential store 'list' API.
func NewListSubcommand() subcommands.Command {
	return &helperCmd{
		cmd{
			name:     "list",
			synopsis: "(UNIMPLEMENTED) list all stored credentials",
		},
	}
}
