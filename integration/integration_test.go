package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
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
	emptyContainerDiff      = `[
	{
	  "Image1": %s,
	  "Image2": %s,
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

	ex, err := os.Executable()
	if err != nil {
		fmt.Printf("err=%s", err)
		t.FailNow()
	}

	buildcontextPath := filepath.Dir(ex)

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
			output, err := dockerCmd.CombinedOutput()
			if err != nil {
				t.Logf("output=%s", output)
				t.Error(err)
				t.Fail()
			}

			// build kaniko image
			kanikoImage := strings.ToLower(testRepo + kanikoPrefix + dockerfile.Name())
			// kanikoCmd := exec.Command("./run_in_docker.sh", path.Join(dockerfilesPath, dockerfile.Name()), buildcontextPath, kanikoImage)
			kanikoCmd := exec.Command("docker", "run",
				"-v", os.Getenv("HOME")+"/.config/gcloud:/root/.config/gcloud",
				"-v", buildcontextPath+":/workspace",
				executorImage,
				"-f", path.Join(buildcontextPath, "integration", dockerfilesPath, dockerfile.Name()),
				"-d", kanikoImage,
				"-c", "/workspace",
			)

			t.Logf("args=%s", kanikoCmd.Args)
			output, err = kanikoCmd.CombinedOutput()
			if err != nil {
				t.Logf(string(output))
				t.Error(err)
				t.Fail()
			}

			// container-diff
			daemonDockerImage := daemonPrefix + dockerImage
			daemonKanikoImage := daemonPrefix + kanikoImage
			containerdiffCmd := exec.Command("container-diff", "diff", daemonDockerImage, daemonKanikoImage)
			diff, err := containerdiffCmd.CombinedOutput()
			if err != nil {
				t.Error(err)
				t.Fail()
			}

			t.Logf("diff = %s", string(diff))

			// make sure the json is empty
			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage)
			if expected != string(diff) {
				t.Errorf("container-diff produced unexpected output: %s", string(diff))
				t.Fail()
			}
		})
	}
}
