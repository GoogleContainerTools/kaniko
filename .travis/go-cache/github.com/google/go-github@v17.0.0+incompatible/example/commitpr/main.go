// Copyright 2018 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The commitpr command utilizes go-github as a CLI tool for
// pushing files to a branch and creating a pull request from it.
// It takes an auth token as an environment variable and creates
// the commit and the PR under the account affiliated with that token.
//
// The purpose of this example is to show how to use refs, trees and commits to
// create commits and pull requests.
//
// Note, if you want to push a single file, you probably prefer to use the
// content API. An example is available here:
// https://godoc.org/github.com/google/go-github/github#example-RepositoriesService-CreateFile
//
// Note, for this to work at least 1 commit is needed, so you if you use this
// after creating a repository you might want to make sure you set `AutoInit` to
// `true`.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	sourceOwner   = flag.String("source-owner", "", "Name of the owner (user or org) of the repo to create the commit in.")
	sourceRepo    = flag.String("source-repo", "", "Name of repo to create the commit in.")
	commitMessage = flag.String("commit-message", "", "Content of the commit message.")
	commitBranch  = flag.String("commit-branch", "", "Name of branch to create the commit in. If it does not already exists, it will be created using the `base-branch` parameter")
	baseBranch    = flag.String("base-branch", "master", "Name of branch to create the `commit-branch` from.")
	prRepoOwner   = flag.String("merge-repo-owner", "", "Name of the owner (user or org) of the repo to create the PR against. If not specified, the value of the `-source-owner` flag will be used.")
	prRepo        = flag.String("merge-repo", "", "Name of repo to create the PR against. If not specified, the value of the `-source-repo` flag will be used.")
	prBranch      = flag.String("merge-branch", "master", "Name of branch to create the PR against (the one you want to merge your branch in via the PR).")
	prSubject     = flag.String("pr-title", "", "Title of the pull request. If not specified, no pull request will be created.")
	prDescription = flag.String("pr-text", "", "Text to put in the description of the pull request.")
	sourceFiles   = flag.String("files", "", `Comma-separated list of files to commit and their location.
The local file is separated by its target location by a semi-colon.
If the file should be in the same location with the same name, you can just put the file name and omit the repetition.
Example: README.md,main.go:github/examples/commitpr/main.go`)
	authorName  = flag.String("author-name", "", "Name of the author of the commit.")
	authorEmail = flag.String("author-email", "", "Email of the author of the commit.")
)

var client *github.Client
var ctx = context.Background()

// getRef returns the commit branch reference object if it exists or creates it
// from the base branch before returning it.
func getRef() (ref *github.Reference, err error) {
	if ref, _, err = client.Git.GetRef(ctx, *sourceOwner, *sourceRepo, "refs/heads/"+*commitBranch); err == nil {
		return ref, nil
	}

	// We consider that an error means the branch has not been found and needs to
	// be created.
	if *commitBranch == *baseBranch {
		return nil, errors.New("The commit branch does not exist but `-base-branch` is the same as `-commit-branch`")
	}

	if *baseBranch == "" {
		return nil, errors.New("The `-base-branch` should not be set to an empty string when the branch specified by `-commit-branch` does not exists")
	}

	var baseRef *github.Reference
	if baseRef, _, err = client.Git.GetRef(ctx, *sourceOwner, *sourceRepo, "refs/heads/"+*baseBranch); err != nil {
		return nil, err
	}
	newRef := &github.Reference{Ref: github.String("refs/heads/" + *commitBranch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = client.Git.CreateRef(ctx, *sourceOwner, *sourceRepo, newRef)
	return ref, err
}

// getTree generates the tree to commit based on the given files and the commit
// of the ref you got in getRef.
func getTree(ref *github.Reference) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []github.TreeEntry{}

	// Load each file into the tree.
	for _, fileArg := range strings.Split(*sourceFiles, ",") {
		file, content, err := getFileContent(fileArg)
		if err != nil {
			return nil, err
		}
		entries = append(entries, github.TreeEntry{Path: github.String(file), Type: github.String("blob"), Content: github.String(string(content)), Mode: github.String("100644")})
	}

	tree, _, err = client.Git.CreateTree(ctx, *sourceOwner, *sourceRepo, *ref.Object.SHA, entries)
	return tree, err
}

// getFileContent loads the local content of a file and return the target name
// of the file in the target repository and its contents.
func getFileContent(fileArg string) (targetName string, b []byte, err error) {
	var localFile string
	files := strings.Split(fileArg, ":")
	switch {
	case len(files) < 1:
		return "", nil, errors.New("empty `-files` parameter")
	case len(files) == 1:
		localFile = files[0]
		targetName = files[0]
	default:
		localFile = files[0]
		targetName = files[1]
	}

	b, err = ioutil.ReadFile(localFile)
	return targetName, b, err
}

// createCommit creates the commit in the given reference using the given tree.
func pushCommit(ref *github.Reference, tree *github.Tree) (err error) {
	// Get the parent commit to attach the commit to.
	parent, _, err := client.Repositories.GetCommit(ctx, *sourceOwner, *sourceRepo, *ref.Object.SHA)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	author := &github.CommitAuthor{Date: &date, Name: authorName, Email: authorEmail}
	commit := &github.Commit{Author: author, Message: commitMessage, Tree: tree, Parents: []github.Commit{*parent.Commit}}
	newCommit, _, err := client.Git.CreateCommit(ctx, *sourceOwner, *sourceRepo, commit)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = client.Git.UpdateRef(ctx, *sourceOwner, *sourceRepo, ref, false)
	return err
}

// createPR creates a pull requesit. Based on: https://godoc.org/github.com/google/go-github/github#example-PullRequestsService-Create
func createPR() (err error) {
	if *prSubject == "" {
		return errors.New("missing `-pr-title` flag; skipping PR creation")
	}

	if *prRepoOwner != "" && *prRepoOwner != *sourceOwner {
		*commitBranch = fmt.Sprintf("%s:%s", *sourceOwner, *commitBranch)
	} else {
		prRepoOwner = sourceOwner
	}

	if *prRepo == "" {
		prRepo = sourceRepo
	}

	newPR := &github.NewPullRequest{
		Title:               prSubject,
		Head:                commitBranch,
		Base:                prBranch,
		Body:                prDescription,
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := client.PullRequests.Create(ctx, *prRepoOwner, *prRepo, newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}

func main() {
	flag.Parse()
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	if *sourceOwner == "" || *sourceRepo == "" || *commitBranch == "" || *sourceFiles == "" || *authorName == "" || *authorEmail == "" {
		log.Fatal("You need to specify a non-empty value for the flags `-source-owner`, `-source-repo`, `-commit-branch`, `-files`, `-author-name` and `-author-email`")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	ref, err := getRef()
	if err != nil {
		log.Fatalf("Unable to get/create the commit reference: %s\n", err)
	}
	if ref == nil {
		log.Fatalf("No error where returned but the reference is nil")
	}

	tree, err := getTree(ref)
	if err != nil {
		log.Fatalf("Unable to create the tree based on the provided files: %s\n", err)
	}

	if err := pushCommit(ref, tree); err != nil {
		log.Fatalf("Unable to create the commit: %s\n", err)
	}

	if err := createPR(); err != nil {
		log.Fatalf("Error while creating the pull request: %s", err)
	}
}
