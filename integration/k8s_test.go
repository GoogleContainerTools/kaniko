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
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"text/template"
)

type K8sConfig struct {
	KanikoImage string
	Context     string
	Name        string
}

func TestK8s(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(cwd, "dockerfiles-with-context")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	testDirs := make([]fs.FileInfo, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		testDirs = append(testDirs, info)
	}

	builder := NewDockerFileBuilder()

	for _, tdInfo := range testDirs {
		name := tdInfo.Name()
		testDir := filepath.Join(dir, name)

		t.Run("test_k8s_with_context_"+name, func(t *testing.T) {
			t.Parallel()

			if err := builder.BuildDockerImage(
				t, config.imageRepo, "", name, testDir,
			); err != nil {
				t.Fatal(err)
			}

			dockerImage := GetDockerImage(config.imageRepo, name)
			kanikoImage := GetKanikoImage(config.imageRepo, name)

			tmpfile, err := os.CreateTemp("", "k8s-job-*.yaml")
			if err != nil {
				log.Fatal(err)
			}
			defer os.Remove(tmpfile.Name()) // clean up
			tmpl := template.Must(template.ParseFiles("k8s-job.yaml"))
			job := K8sConfig{KanikoImage: kanikoImage, Context: testDir, Name: name}
			if err := tmpl.Execute(tmpfile, job); err != nil {
				t.Fatal(err)
			}

			t.Logf("Testing K8s based Kaniko building of dockerfile %s and push to %s \n",
				testDir, kanikoImage)
			content, err := os.ReadFile(tmpfile.Name())
			if err != nil {
				log.Fatal(err)
			}
			t.Logf("K8s template %s:\n%s\n", tmpfile.Name(), content)

			kubeCmd := exec.Command("kubectl", "apply", "-f", tmpfile.Name())
			RunCommand(kubeCmd, t)

			t.Logf("Waiting for K8s kaniko build job to finish: %s\n",
				"job/kaniko-test-"+job.Name)

			kubeWaitCmd := exec.Command("kubectl", "wait", "--for=condition=complete", "--timeout=2m",
				"job/kaniko-test-"+job.Name)
			if out, errR := RunCommandWithoutTest(kubeWaitCmd); errR != nil {
				t.Log(kubeWaitCmd.Args)
				t.Log(string(out))
				descCmd := exec.Command("kubectl", "describe", "job/kaniko-test-"+job.Name)
				outD, errD := RunCommandWithoutTest(descCmd)
				if errD != nil {
					t.Error(errD)
				} else {
					t.Log(string(outD))
				}

				descCmd = exec.Command("kubectl", "describe", "pods", "--selector", "job-name=kaniko-test-"+job.Name)
				outD, errD = RunCommandWithoutTest(descCmd)
				if errD != nil {
					t.Error(errD)
				} else {
					t.Log(string(outD))
				}

				logsCmd := exec.Command("kubectl", "logs", "--all-containers", "job/kaniko-test-"+job.Name)
				outL, errL := RunCommandWithoutTest(logsCmd)
				if errL != nil {
					t.Error(errL)
				} else {
					t.Log(string(outL))
				}

				t.Fatal(errR)
			}

			diff := containerDiff(t, daemonPrefix+dockerImage, kanikoImage, "--no-cache")

			expected := fmt.Sprintf(emptyContainerDiff, dockerImage, kanikoImage, dockerImage, kanikoImage)
			checkContainerDiffOutput(t, diff, expected)
		})
	}

	if err := logBenchmarks("benchmark"); err != nil {
		t.Logf("Failed to create benchmark file: %v", err)
	}
}
