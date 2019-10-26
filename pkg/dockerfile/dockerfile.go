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

package dockerfile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/pkg/errors"
)

// Stages parses a Dockerfile and returns an array of KanikoStage
func Stages(opts *config.KanikoOptions) ([]config.KanikoStage, error) {
	var err error
	var d []uint8
	match, _ := regexp.MatchString("^https?://", opts.DockerfilePath)
	if match {
		response, e := http.Get(opts.DockerfilePath)
		if e != nil {
			return nil, e
		}
		d, err = ioutil.ReadAll(response.Body)
	} else {
		d, err = ioutil.ReadFile(opts.DockerfilePath)
	}

	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("reading dockerfile at path %s", opts.DockerfilePath))
	}

	stages, metaArgs, err := Parse(d)
	if err != nil {
		return nil, errors.Wrap(err, "parsing dockerfile")
	}
	targetStage, err := targetStage(stages, opts.Target)
	if err != nil {
		return nil, err
	}
	resolveStages(stages)
	var kanikoStages []config.KanikoStage
	for index, stage := range stages {
		resolvedBaseName, err := util.ResolveEnvironmentReplacement(stage.BaseName, opts.BuildArgs, false)
		if err != nil {
			return nil, errors.Wrap(err, "resolving base name")
		}
		stage.Name = resolvedBaseName
		logrus.Infof("Resolved base name %s to %s", stage.BaseName, stage.Name)
		kanikoStages = append(kanikoStages, config.KanikoStage{
			Stage:                  stage,
			BaseImageIndex:         baseImageIndex(index, stages),
			BaseImageStoredLocally: (baseImageIndex(index, stages) != -1),
			SaveStage:              saveStage(index, stages),
			Final:                  index == targetStage,
			MetaArgs:               metaArgs,
			Index:                  index,
		})
		if index == targetStage {
			break
		}
	}

	return kanikoStages, nil
}

// baseImageIndex returns the index of the stage the current stage is built off
// returns -1 if the current stage isn't built off a previous stage
func baseImageIndex(currentStage int, stages []instructions.Stage) int {
	for i, stage := range stages {
		if i > currentStage {
			break
		}
		if stage.Name == stages[currentStage].BaseName {
			return i
		}
	}
	return -1
}

// Parse parses the contents of a Dockerfile and returns a list of commands
func Parse(b []byte) ([]instructions.Stage, []instructions.ArgCommand, error) {
	p, err := parser.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	stages, metaArgs, err := instructions.Parse(p.AST)
	if err != nil {
		return nil, nil, err
	}

	metaArgs, err = stripEnclosingQuotes(metaArgs)
	if err != nil {
		return nil, nil, err
	}

	return stages, metaArgs, nil
}

// stripEnclosingQuotes removes quotes enclosing the value of each instructions.ArgCommand in a slice
// if the quotes are escaped it leaves them
func stripEnclosingQuotes(metaArgs []instructions.ArgCommand) ([]instructions.ArgCommand, error) {
	for i := range metaArgs {
		arg := metaArgs[i]
		v := arg.Value
		if v != nil {
			val, err := extractValFromQuotes(*v)
			if err != nil {
				return nil, err
			}

			arg.Value = &val
			metaArgs[i] = arg
		}
	}
	return metaArgs, nil
}

func extractValFromQuotes(val string) (string, error) {
	backSlash := byte('\\')
	if len(val) < 2 {
		return val, nil
	}

	var leader string
	var tail string

	switch char := val[0]; char {
	case '\'', '"':
		leader = string([]byte{char})
	case backSlash:
		switch char := val[1]; char {
		case '\'', '"':
			leader = string([]byte{backSlash, char})
		}
	}

	// If the length of leader is greater than one then it must be an escaped
	// character.
	if len(leader) < 2 {
		switch char := val[len(val)-1]; char {
		case '\'', '"':
			tail = string([]byte{char})
		}
	} else {
		switch char := val[len(val)-2:]; char {
		case `\'`, `\"`:
			tail = char
		}
	}

	if leader != tail {
		logrus.Infof("leader %s tail %s", leader, tail)
		return "", errors.New("quotes wrapping arg values must be matched")
	}

	if leader == "" {
		return val, nil
	}

	if len(leader) == 2 {
		return val, nil
	}

	return val[1 : len(val)-1], nil
}

// targetStage returns the index of the target stage kaniko is trying to build
func targetStage(stages []instructions.Stage, target string) (int, error) {
	if target == "" {
		return len(stages) - 1, nil
	}
	for i, stage := range stages {
		if stage.Name == target {
			return i, nil
		}
	}
	return -1, fmt.Errorf("%s is not a valid target build stage", target)
}

// resolveStages resolves any calls to previous stages with names to indices
// Ex. --from=second_stage should be --from=1 for easier processing later on
// As third party library lowers stage name in FROM instruction, this function resolves stage case insensitively.
func resolveStages(stages []instructions.Stage) {
	nameToIndex := make(map[string]string)
	for i, stage := range stages {
		index := strconv.Itoa(i)
		if stage.Name != index {
			nameToIndex[stage.Name] = index
		}
		for _, cmd := range stage.Commands {
			switch c := cmd.(type) {
			case *instructions.CopyCommand:
				if c.From != "" {
					if val, ok := nameToIndex[strings.ToLower(c.From)]; ok {
						c.From = val
					}

				}
			}
		}
	}
}

// ParseCommands parses an array of commands into an array of instructions.Command; used for onbuild
func ParseCommands(cmdArray []string) ([]instructions.Command, error) {
	var cmds []instructions.Command
	cmdString := strings.Join(cmdArray, "\n")
	ast, err := parser.Parse(strings.NewReader(cmdString))
	if err != nil {
		return nil, err
	}
	for _, child := range ast.AST.Children {
		cmd, err := instructions.ParseCommand(child)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

// SaveStage returns true if the current stage will be needed later in the Dockerfile
func saveStage(index int, stages []instructions.Stage) bool {
	for stageIndex, stage := range stages {
		if stageIndex <= index {
			continue
		}
		if stage.BaseName == stages[index].Name {
			if stage.BaseName != "" {
				return true
			}
		}
	}
	return false
}
