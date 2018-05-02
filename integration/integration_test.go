package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

const (
	executorImage           = "executor-image"
	dockerImage             = "gcr.io/cloud-builders/docker"
	ubuntuImage             = "ubuntu"
	testRepo                = "gcr.io/kaniko-test/"
	dockerPrefix            = "docker-"
	kanikoPrefix            = "kaniko-"
	daemonPrefix            = "daemon://"
	containerDiffOutputFile = "container-diff.json"
	kanikoTestBucket        = "kaniko-test-bucket"
	dockerfilesPath         = "dockerfiles"
	onbuildBaseImage        = testRepo + "onbuild-base:latest"
	buildContextPath        = "/workspace"
	emptyContainerDiff      = `[
     {
       "Image1": "%s:latest",
       "Image2": "%s:latest",
       "DiffType": "File",
       "Diff": {
	 "Adds": null,
	 "Dels": null,
	 "Mods": null
       }
     }
   ]`
)

func TestMain(m *testing.M) {
	buildKaniko := exec.Command("docker", "build", "-t", executorImage, "-f", "../deploy/Dockerfile", "..")
	output, err := buildKaniko.CombinedOutput()
	if err != nil {
		fmt.Printf("output=%s\n", output)
		fmt.Printf("err=%s\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	dockerfiles, err := ioutil.ReadDir(dockerfilesPath)
	if err != nil {
		fmt.Printf("err=%s", err)
		t.FailNow()
	}

	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	// Grab the latest container-diff binary
	getContainerDiff := exec.Command("gsutil", "cp", "gs://container-diff/latest/container-diff-linux-amd64", ".")
	err = getContainerDiff.Run()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	containerDiffPerms := exec.Command("chmod", "+x", "container-diff-linux-amd64")
	err = containerDiffPerms.Run()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}


	for _, dockerfile := range dockerfiles {
		if strings.HasSuffix(dockerfile.Name(), ".yaml") {
			continue
		}
		t.Run("test_"+dockerfile.Name(), func(t *testing.T) {
			// Parallelization is broken
			// t.Parallel()

			// build docker image
			dockerImage := strings.ToLower(testRepo + dockerPrefix + dockerfile.Name())
			dockerCmd := exec.Command("docker", "build", "-t", dockerImage, "-f", path.Join(dockerfilesPath, dockerfile.Name()), ".")
			err = dockerCmd.Run()
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			// build kaniko image
			kanikoImage := strings.ToLower(testRepo + kanikoPrefix + dockerfile.Name())
			kanikoCmd := exec.Command("docker", "run",
				"-v", os.Getenv("HOME")+"/.config/gcloud:/root/.config/gcloud",
				"-v", cwd + ":/workspace",
				executorImage,
				"-f", path.Join(buildContextPath, dockerfilesPath, dockerfile.Name()),
				"-d", kanikoImage,
				"-c", buildContextPath,
			)

			err = kanikoCmd.Run()
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			// container-diff
			daemonDockerImage := daemonPrefix + dockerImage
			//daemonKanikoImage := daemonPrefix + kanikoImage

			containerdiffCmd := exec.Command("./container-diff-linux-amd64", "diff", daemonDockerImage, kanikoImage, "-q", "--type=file", "--json")
			diff, err := containerdiffCmd.CombinedOutput()
			if err != nil {
				t.Error(err)
				t.Fail()
			}
			t.Logf("diff = %s", string(diff))

			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage)

			// Let's compare the json objects themselves instead of strings to avoid
			// issues with spaces and indents
			var diffInt interface{}
			var expectedInt interface{}

			err = json.Unmarshal(diff, &diffInt)
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			err = json.Unmarshal([]byte(expected), &expectedInt)
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			testutil.CheckErrorAndDeepEqual(t, false, nil, expectedInt, diffInt)
		})
	}
}
