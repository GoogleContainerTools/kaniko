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

package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	token   string
	fromTag string
	toTag   string
)

var rootCmd = &cobra.Command{
	Use:        "listpullreqs fromTag toTag",
	Short:      "Lists pull requests between two versions in our changelog markdown format",
	ArgAliases: []string{"fromTag", "toTag"},
	Run: func(cmd *cobra.Command, args []string) {
		printPullRequests()
	},
}

const org = "GoogleContainerTools"
const repo = "kaniko"

func main() {
	rootCmd.Flags().StringVar(&token, "token", "", "Specify personal Github Token if you are hitting a rate limit anonymously. https://github.com/settings/tokens")
	rootCmd.Flags().StringVar(&fromTag, "fromTag", "", "comparison of commits is based on this tag (defaults to the latest tag in the repo)")
	rootCmd.Flags().StringVar(&toTag, "toTag", "master", "this is the commit that is compared with fromTag")
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func printPullRequests() {
	client := getClient()

	releases, _, _ := client.Repositories.ListReleases(context.Background(), org, repo, &github.ListOptions{})
	lastReleaseTime := *releases[0].PublishedAt

	listSize := 1
	seen := map[int]bool{}

	for page := 0; listSize > 0; page++ {
		pullRequests, _, _ := client.PullRequests.List(context.Background(), org, repo, &github.PullRequestListOptions{
			State:     "closed",
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		})

		for idx := range pullRequests {
			pr := pullRequests[idx]
			if pr.MergedAt != nil {
				if _, ok := seen[*pr.Number]; !ok && pr.GetMergedAt().After(lastReleaseTime.Time) {
					fmt.Printf("* %s [#%d](https://github.com/%s/%s/pull/%d)\n", pr.GetTitle(), *pr.Number, org, repo, *pr.Number)
					seen[*pr.Number] = true
				}
			}
		}

		listSize = len(pullRequests)
	}
}

func getClient() *github.Client {
	if len(token) <= 0 {
		return github.NewClient(nil)
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}
