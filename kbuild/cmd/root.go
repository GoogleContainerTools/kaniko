/*
Copyright 2018 Google, Inc. All rights reserved.

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
	"fmt"
	"github.com/GoogleCloudPlatform/k8s-container-builder/contexts/source"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/util"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	batch "k8s.io/api/batch/v1"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	dockerfile string
	name       string
	srcContext string
	logLevel   string
	kubeconfig string
)

var RootCmd = &cobra.Command{
	Use:   "kbuild",
	Short: "kbuild is a CLI tool for building container images with full Dockerfile support without the need for Docker",
	Long: `kbuild is a CLI tool for building container images with full Dockerfile support. It doesn't require Docker,
			and builds the images in a Kubernetes cluster before pushing the final image to a registry.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return errors.Wrap(err, "parsing log level")
		}
		logrus.SetLevel(lvl)
		if err := checkFlags(); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func checkFlags() error {
	if srcContext == "" {
		return errors.Errorf("Please provide source context with --context or -c flag")
	}
	if name == "" {
		return errors.Errorf("Please provide name of final image with --name or -n flag")
	}
	if _, err := os.Stat(dockerfile); err != nil {
		return errors.Wrap(err, "please provide valid path to Dockerfile with --dockerfile or -d flag")
	}
	return nil
}

// execute uploads the source context and runs the Kubernetes job
func execute() error {
	logrus.SetLevel(logrus.DebugLevel)
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// create source context
	d, err := os.Open(dockerfile)
	if err != nil {
		panic(err)
	}
	dockerfileDeps, err := docker.GetDockerfileDependencies(srcContext, d)
	ctx := source.GetContext(srcContext)
	logrus.Debugf("Copying %s to context", dockerfile)
	source, err := ctx.CopyFilesToContext(dockerfileDeps)
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadFile(dockerfile)
	if err != nil {
		log.Fatal(err)
	}

	cfg := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "build-dockerfile",
		},
		Data: map[string]string{
			"Dockerfile": string(b),
		},
	}

	cfgmap, err := clientset.CoreV1().ConfigMaps("default").Create(cfg)
	if err != nil {
		log.Fatal(err)
	}

	j := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "build-job-",
		},
		Spec: batch.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "init-static",
							Image: "gcr.io/priya-wadhwa/kbuilder:latest",
							Command: []string{
								"/work-dir/main", "--source", source, "--dest", name, "--verbosity", logLevel,
							},
							Args:         []string{},
							VolumeMounts: []v1.VolumeMount{v1.VolumeMount{Name: "dockerfile", MountPath: "/dockerfile"}},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
					Volumes: []v1.Volume{
						{
							Name: "dockerfile",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: cfgmap.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	job, err := clientset.BatchV1().Jobs("default").Create(j)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created build job: ", job.Name)

	stopCh := make(chan bool)

	for {
		j, err = clientset.BatchV1().Jobs("default").Get(job.Name, metav1.GetOptions{})
		if err != nil {
			// wait until the job exists
			continue
		}
		break
	}
	for {
		opts := metav1.ListOptions{LabelSelector: labels.Set(j.Spec.Selector.MatchLabels).AsSelector().String()}
		jobPods, err := clientset.CoreV1().Pods("default").List(opts)
		if err != nil {
			continue
		}
		// Stream logs
		for _, p := range jobPods.Items {
			f := func() {
				streamLogs(clientset, "init-static", p.Name, "default")
			}
			go util.Until(f, stopCh)
		}
		break
	}
	for {
		j, err := clientset.BatchV1().Jobs("default").Get(job.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}

		if j.Status.CompletionTime == nil {
			time.Sleep(2 * time.Second)
		} else {
			fmt.Println("Job finished.")
			stopCh <- true
			break
		}
	}
	return nil
}

func streamLogs(clientset *kubernetes.Clientset, container, pod, namespace string) error {
	r, err := clientset.CoreV1().Pods(namespace).GetLogs(pod, &v1.PodLogOptions{Container: container, Follow: true}).Stream()
	if err != nil {
		return err
	}
	defer r.Close()
	if _, err := io.Copy(os.Stderr, r); err != nil {
		return err
	}
	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&dockerfile, "dockerfile", "d", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&srcContext, "context", "c", "", "Path to the dockerfile context.")
	RootCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "Name of the registry location the final image should be pushed to (ex: gcr.io/test/example:latest)")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", constants.DefaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
	kubeConfigDefaultPath := filepath.Join(homeDir(), ".kube", "config")
	RootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", kubeConfigDefaultPath, "(optional) absolute path to the kubeconfig file")
}
