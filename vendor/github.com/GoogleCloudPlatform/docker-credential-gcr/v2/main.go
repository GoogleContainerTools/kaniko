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

/*
Program docker-credential-gcr implements the Docker credential helper API
and allows for more advanced login/authentication schemes for GCR customers.

See README.md
*/
package main

import (
	"context"
	"flag"
	"os"

	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/cli"
	"github.com/google/subcommands"
)

const (
	gcrGroup             = "GCR authentication"
	dockerCredStoreGroup = "Docker credential store API"
	configGroup          = "Config"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(cli.NewStoreSubcommand(), dockerCredStoreGroup)
	subcommands.Register(cli.NewGetSubcommand(), dockerCredStoreGroup)
	subcommands.Register(cli.NewEraseSubcommand(), dockerCredStoreGroup)
	subcommands.Register(cli.NewListSubcommand(), dockerCredStoreGroup)
	subcommands.Register(cli.NewGCRLoginSubcommand(), gcrGroup)
	subcommands.Register(cli.NewGCRLogoutSubcommand(), gcrGroup)
	subcommands.Register(cli.NewDockerConfigSubcommand(), configGroup)
	subcommands.Register(cli.NewConfigSubcommand(), configGroup)
	subcommands.Register(cli.NewVersionSubcommand(), "")
	subcommands.Register(cli.NewClearSubcommand(), "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
