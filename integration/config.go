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

package integration

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type buildConfig interface {
	IsTravis() bool
	Identifier() string
	GitRepo() string
}

type integrationTestConfig struct {
	buildInfo          buildConfig
	gcsBucket          string
	imageRepo          string
	onbuildBaseImage   string
	hardlinkBaseImage  string
	serviceAccount     string
	dockerMajorVersion int
}

type TravisBuild struct {
	pullRequest string
	branch      string
	repoSlug    string
}

func (t *TravisBuild) IsTravis() bool {
	return true
}

func (t *TravisBuild) Identifier() string {
	if t.pullRequest != "" {
		return t.pullRequest
	}
	return t.branch
}

func (t *TravisBuild) GitRepo() string {
	return "github.com/" + t.repoSlug + "#refs/heads/" + t.branch
}

type Build struct {
}

func (b *Build) IsTravis() bool {
	return false
}

func (b *Build) Identifier() string {
	return fmt.Sprintf("run_%s", time.Now().Format("2006-01-02-15:04"))
}

func (b *Build) GitRepo() string {
	return "github.com/GoogleContainerTools/kaniko"
}

const gcrRepoPrefix string = "gcr.io/"

func (config *integrationTestConfig) isGcrRepository() bool {
	return strings.HasPrefix(config.imageRepo, gcrRepoPrefix)
}

func initIntegrationTestConfig() *integrationTestConfig {
	var c integrationTestConfig
	flag.StringVar(&c.gcsBucket, "bucket", "gs://kaniko-test-bucket", "The gcs bucket argument to uploaded the tar-ed contents of the `integration` dir to.")
	flag.StringVar(&c.imageRepo, "repo", "gcr.io/kaniko-test", "The (docker) image repo to build and push images to during the test. `gcloud` must be authenticated with this repo or serviceAccount must be set.")
	flag.StringVar(&c.serviceAccount, "serviceAccount", "", "The path to the service account push images to GCR and upload/download files to GCS.")
	flag.Parse()

	if len(c.serviceAccount) > 0 {
		absPath, err := filepath.Abs("../" + c.serviceAccount)
		if err != nil {
			log.Fatalf("Error getting absolute path for service account: %s\n", c.serviceAccount)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Fatalf("Service account does not exist: %s\n", absPath)
		}
		c.serviceAccount = absPath
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", absPath)
	}

	if c.imageRepo == "" {
		log.Fatal("You must provide a image repository")
	}

	if c.isGcrRepository() && c.gcsBucket == "" {
		log.Fatalf("You must provide a gcs bucket when using a Google Container Registry (\"%s\" was provided)", c.imageRepo)
	}
	if !strings.HasSuffix(c.imageRepo, "/") {
		c.imageRepo = c.imageRepo + "/"
	}
	c.dockerMajorVersion = getDockerMajorVersion()
	c.onbuildBaseImage = c.imageRepo + "onbuild-base:latest"
	c.hardlinkBaseImage = c.imageRepo + "hardlink-base:latest"
	c.buildInfo = getBuildConfig()
	return &c
}

func getDockerMajorVersion() int {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err != nil {
		log.Fatal("Error getting docker version of server:", err)
	}
	versionArr := strings.Split(string(out), ".")

	ver, err := strconv.Atoi(versionArr[0])
	if err != nil {
		log.Fatal("Error getting docker version of server during parsing version string:", err)
	}
	return ver
}

func getBuildConfig() buildConfig {
	if _, ok := os.LookupEnv("TRAVIS"); ok {
		t := &TravisBuild{}
		if n := os.Getenv("TRAVIS_PULL_REQUEST"); n != "false" {
			t.pullRequest = n
			t.branch = os.Getenv("TRAVIS_PULL_REQUEST_BRANCH")
			t.repoSlug = os.Getenv("TRAVIS_PULL_REQUEST_SLUG")
			log.Printf("Travis CI Pull request source repo: %s branch: %s\n", t.repoSlug, t.branch)
		} else {
			t.branch = os.Getenv("TRAVIS_BRANCH")
			t.repoSlug = os.Getenv("TRAVIS_REPO_SLUG")
			log.Printf("Travis CI repo: %s branch: %s\n", t.repoSlug, t.branch)
		}
		return t
	}
	return &Build{}
}
