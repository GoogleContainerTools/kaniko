package integration

import (
	"encoding/json"
	"fmt"
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
	executorImage      = "executor-image"
	dockerImage        = "gcr.io/cloud-builders/docker"
	ubuntuImage        = "ubuntu"
	testRepo           = "gcr.io/kaniko-test/"
	dockerPrefix       = "docker-"
	kanikoPrefix       = "kaniko-"
	daemonPrefix       = "daemon://"
	kanikoTestBucket   = "kaniko-test-bucket"
	dockerfilesPath    = "dockerfiles"
	onbuildBaseImage   = testRepo + "onbuild-base:latest"
	buildContextPath   = "/workspace"
	emptyContainerDiff = `[
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
	out, err := buildKaniko.Run()
	if err != nil {
		fmt.Print(out)
		fmt.Print(err)
		fmt.Print("Building kaniko failed.")
		os.Exit(1)
	}

	// Make sure container-diff is on user's PATH
	err = exec.Command("container-diff").Run()
	if err != nil {
		fmt.Print("Make sure you have container-diff installed and on your PATH")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	dockerfiles, err := filepath.Glob(path.Join(dockerfilesPath, "Dockerfile*"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	_, ex, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(ex)

	for _, dockerfile := range dockerfiles {
		t.Run("test_"+dockerfile, func(t *testing.T) {
			fmt.Printf("%s\n", dockerfile)
			// Parallelization is broken
			// t.Parallel()

			// build docker image
			dockerImage := strings.ToLower(testRepo + dockerPrefix + dockerfile)
			dockerCmd := exec.Command("docker", "build",
				"-t", dockerImage,
				"-f", dockerfile,
				".")
			RunCommand(dockerCmd, t)

			// build kaniko image
			kanikoImage := strings.ToLower(testRepo + kanikoPrefix + dockerfile)
			kanikoCmd := exec.Command("docker", "run",
				"-v", os.Getenv("HOME")+"/.config/gcloud:/root/.config/gcloud",
				"-v", cwd+":/workspace",
				executorImage,
				"-f", path.Join(buildContextPath, dockerfile),
				"-d", kanikoImage,
				"-c", buildContextPath,
			)

			RunCommand(kanikoCmd, t)

			// container-diff
			daemonDockerImage := daemonPrefix + dockerImage
			//daemonKanikoImage := daemonPrefix + kanikoImage

			containerdiffCmd := exec.Command("container-diff", "diff",
				daemonDockerImage, kanikoImage,
				"-q", "--type=file", "--json")
			diff := RunCommand(containerdiffCmd, t)
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

func RunCommand(cmd *exec.Cmd, t *testing.T) []byte {
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(cmd.Args)
		t.Log(output)
		t.Error(err)
		t.Fail()
	}

	return output
}
