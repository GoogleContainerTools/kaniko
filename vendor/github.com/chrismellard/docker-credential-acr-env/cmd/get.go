/*
Copyright © 2020 Chris Mellard chris.mellard@icloud.com

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
package cmd

import (
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "for the server specified via stdin, return the stored credentials via stdout",
	Run: func(cmd *cobra.Command, args []string) {
		credentials.Serve(credhelper.NewACRCredentialsHelper())
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
