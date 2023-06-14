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
Package cli contains the implementations of all of the subcommands that are
exposed via the command line.
*/
package cli

import (
	"flag"
	"fmt"
)

type cmd struct {
	name, synopsis string
}

// Name returns the name of the command.
func (c *cmd) Name() string { return c.name }

// Synopsis returns the synopsis of the command.
func (c *cmd) Synopsis() string { return c.synopsis }

// Usage returns the name of the command followed by its synopsis and a new line.
func (c *cmd) Usage() string {
	return fmt.Sprintf("%s: %s\n", c.Name(), c.Synopsis())
}

// SetFlags is a no-op in order to implement the Command interface.
func (*cmd) SetFlags(*flag.FlagSet) {}
