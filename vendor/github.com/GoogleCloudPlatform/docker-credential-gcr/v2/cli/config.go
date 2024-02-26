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
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/docker-credential-gcr/v2/config"
	"github.com/google/subcommands"
)

const (
	tokenSourceFlag = "token-source"
	resetAllFlag    = "unset-all"
)

type configCmd struct {
	cmd
	tokenSources string
	resetAll     bool
}

// NewConfigSubcommand returns a subcommands.Command which allows for user
// configuration of cred helper behavior.
func NewConfigSubcommand() subcommands.Command {
	return &configCmd{
		cmd{
			name:     "config",
			synopsis: "configure the credential helper",
		},
		// Because only specified flags are iterated by FlagSet.Visit,
		// these values will always be explicitly set by the user if visited.
		"unused",
		false,
	}
}

func (c *configCmd) SetFlags(fs *flag.FlagSet) {
	srcs := make([]string, 0, len(config.SupportedGCRTokenSources))
	for src := range config.SupportedGCRTokenSources {
		srcs = append(srcs, src)
	}
	supportedSources := strings.Join(srcs, ", ")
	defaultSources := strings.Join(config.DefaultTokenSources[:], ", ")
	fs.StringVar(&c.tokenSources, tokenSourceFlag, defaultSources, "The source(s), in order, to search for credentials. Supported sources are: "+supportedSources)
	fs.BoolVar(&c.resetAll, resetAllFlag, false, "Resets all settings to default")
}

func (c *configCmd) Execute(_ context.Context, flags *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.resetAll {
		if err := resetAll(); err != nil {
			printError(resetAllFlag, err)
			return subcommands.ExitFailure
		}
		printSuccess("Config reset.")
		return subcommands.ExitSuccess
	}

	result := subcommands.ExitSuccess
	flags.Visit(func(f *flag.Flag) {
		if f.Name == tokenSourceFlag {
			if err := setTokenSources(c.tokenSources); err != nil {
				printError(tokenSourceFlag, err)
				result = subcommands.ExitFailure
				return
			}
			printSuccess("Token source(s) set.")
			result = subcommands.ExitSuccess
		}
	})

	return result
}

func resetAll() error {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return err
	}
	return cfg.ResetAll()
}

func setTokenSources(rawSource string) error {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return err
	}
	strReader := strings.NewReader(rawSource)
	sources, err := csv.NewReader(strReader).Read()
	if err != nil {
		return err
	}
	for i, src := range sources {
		sources[i] = strings.TrimSpace(src)
	}
	return cfg.SetTokenSources(sources)
}

func printSuccess(msg string) {
	fmt.Fprintf(os.Stdout, "Success: %s\n", msg)
}

func printError(flag string, err error) {
	fmt.Fprintf(os.Stderr, "Failure: %s: %v\n", flag, err)
}
