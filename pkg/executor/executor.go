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

package executor

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/kaniko/pkg/version"
)

// KanikoBuildArgs contains all the args required to build the image
type KanikoBuildArgs struct {
	DockerfilePath string
	SrcContext     string
	SnapshotMode   string
	Args           []string
	SingleSnapshot bool
	Reproducible   bool
	Target         string
}

func DoBuild(k KanikoBuildArgs) (v1.Image, error) {
	// Parse dockerfile and unpack base image to root
	stages, err := dockerfile.Stages(k.DockerfilePath, k.Target)
	if err != nil {
		return nil, err
	}

	hasher, err := getHasher(k.SnapshotMode)
	if err != nil {
		return nil, err
	}
	for index, stage := range stages {
		finalStage := finalStage(index, k.Target, stages)
		// Unpack file system to root
		sourceImage, err := util.RetrieveSourceImage(index, k.Args, stages)
		if err != nil {
			return nil, err
		}
		if err := util.GetFSFromImage(constants.RootDir, sourceImage); err != nil {
			return nil, err
		}
		l := snapshot.NewLayeredMap(hasher)
		snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)
		// Take initial snapshot
		if err := snapshotter.Init(); err != nil {
			return nil, err
		}
		imageConfig, err := util.RetrieveConfigFile(sourceImage)
		if err != nil {
			return nil, err
		}
		if err := resolveOnBuild(&stage, &imageConfig.Config); err != nil {
			return nil, err
		}
		buildArgs := dockerfile.NewBuildArgs(k.Args)
		for index, cmd := range stage.Commands {
			finalCmd := index == len(stage.Commands)-1
			dockerCommand, err := commands.GetCommand(cmd, k.SrcContext)
			if err != nil {
				return nil, err
			}
			if dockerCommand == nil {
				continue
			}
			if err := dockerCommand.ExecuteCommand(&imageConfig.Config, buildArgs); err != nil {
				return nil, err
			}
			// Don't snapshot if it's not the final stage and not the final command
			// Also don't snapshot if it's the final stage, not the final command, and single snapshot is set
			if (!finalStage && !finalCmd) || (finalStage && !finalCmd && k.SingleSnapshot) {
				continue
			}
			// Now, we get the files to snapshot from this command and take the snapshot
			snapshotFiles := dockerCommand.FilesToSnapshot()
			if finalCmd {
				snapshotFiles = nil
			}
			contents, err := snapshotter.TakeSnapshot(snapshotFiles)
			if err != nil {
				return nil, err
			}
			util.MoveVolumeWhitelistToWhitelist()
			if contents == nil {
				logrus.Info("No files were changed, appending empty layer to config.")
				continue
			}
			// Append the layer to the image
			opener := func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bytes.NewReader(contents)), nil
			}
			layer, err := tarball.LayerFromOpener(opener)
			if err != nil {
				return nil, err
			}
			sourceImage, err = mutate.Append(sourceImage,
				mutate.Addendum{
					Layer: layer,
					History: v1.History{
						Author:    constants.Author,
						CreatedBy: dockerCommand.CreatedBy(),
					},
				},
			)
			if err != nil {
				return nil, err
			}
		}
		sourceImage, err = mutate.Config(sourceImage, imageConfig.Config)
		if err != nil {
			return nil, err
		}
		if finalStage {
			if k.Reproducible {
				sourceImage, err = mutate.Canonical(sourceImage)
				if err != nil {
					return nil, err
				}
			}
			return sourceImage, nil
		}
		if dockerfile.SaveStage(index, stages) {
			if err := saveStageAsTarball(index, sourceImage); err != nil {
				return nil, err
			}
			if err := extractImageToDependecyDir(index, sourceImage); err != nil {
				return nil, err
			}
		}
		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, err
		}
	}
	return nil, err
}

type withUserAgent struct {
	t http.RoundTripper
}

func (w *withUserAgent) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", fmt.Sprintf("kaniko/%s", version.Version()))
	return w.t.RoundTrip(r)
}

func DoPush(image v1.Image, destinations []string, tarPath string) error {

	// continue pushing unless an error occurs
	for _, destination := range destinations {
		// Push the image
		destRef, err := name.NewTag(destination, name.WeakValidation)
		if err != nil {
			return err
		}

		if tarPath != "" {
			return tarball.WriteToFile(tarPath, destRef, image, nil)
		}

		k8sc, err := k8schain.NewNoClient()
		if err != nil {
			return err
		}
		kc := authn.NewMultiKeychain(authn.DefaultKeychain, k8sc)
		pushAuth, err := kc.Resolve(destRef.Context().Registry)
		if err != nil {
			return err
		}

		// Create a transport to set our user-agent.
		rt := &withUserAgent{t: http.DefaultTransport}

		if err := remote.Write(destRef, image, pushAuth, rt, remote.WriteOptions{}); err != nil {
			logrus.Error(fmt.Errorf("Failed to push to destination %s", destination))
			return err
		}
	}
	return nil
}

func finalStage(index int, target string, stages []instructions.Stage) bool {
	if index == len(stages)-1 {
		return true
	}
	if target == "" {
		return false
	}
	return target == stages[index].Name
}

func extractImageToDependecyDir(index int, image v1.Image) error {
	dependencyDir := filepath.Join(constants.KanikoDir, strconv.Itoa(index))
	if err := os.MkdirAll(dependencyDir, 0755); err != nil {
		return err
	}
	logrus.Infof("trying to extract to %s", dependencyDir)
	return util.GetFSFromImage(dependencyDir, image)
}

func saveStageAsTarball(stageIndex int, image v1.Image) error {
	destRef, err := name.NewTag("temp/tag", name.WeakValidation)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(constants.KanikoIntermediateStagesDir, 0750); err != nil {
		return err
	}
	tarPath := filepath.Join(constants.KanikoIntermediateStagesDir, strconv.Itoa(stageIndex))
	logrus.Infof("Storing source image from stage %d at path %s", stageIndex, tarPath)
	return tarball.WriteToFile(tarPath, destRef, image, nil)
}

func getHasher(snapshotMode string) (func(string) (string, error), error) {
	if snapshotMode == constants.SnapshotModeTime {
		logrus.Info("Only file modification time will be considered when snapshotting")
		return util.MtimeHasher(), nil
	}
	if snapshotMode == constants.SnapshotModeFull {
		return util.Hasher(), nil
	}
	return nil, fmt.Errorf("%s is not a valid snapshot mode", snapshotMode)
}

func resolveOnBuild(stage *instructions.Stage, config *v1.Config) error {
	if config.OnBuild == nil {
		return nil
	}
	// Otherwise, parse into commands
	cmds, err := dockerfile.ParseCommands(config.OnBuild)
	if err != nil {
		return err
	}
	// Append to the beginning of the commands in the stage
	stage.Commands = append(cmds, stage.Commands...)
	logrus.Infof("Executing %v build triggers", len(cmds))

	// Blank out the Onbuild command list for this image
	config.OnBuild = nil
	return nil
}
