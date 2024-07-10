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
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/linter"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/pkg/errors"
)

func ParseStages(opts *config.KanikoOptions) ([]instructions.Stage, []instructions.ArgCommand, error) {
	var err error
	var d []uint8
	match, _ := regexp.MatchString("^https?://", opts.DockerfilePath)
	if match {
		response, e := http.Get(opts.DockerfilePath) //nolint:noctx
		if e != nil {
			return nil, nil, e
		}
		d, err = io.ReadAll(response.Body)
	} else {
		d, err = os.ReadFile(opts.DockerfilePath)
	}

	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("reading dockerfile at path %s", opts.DockerfilePath))
	}

	stages, metaArgs, err := Parse(d)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing dockerfile")
	}

	metaArgs, err = expandNestedArgs(metaArgs, opts.BuildArgs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "expanding meta ARGs")
	}

	return stages, metaArgs, nil
}

// baseImageIndex returns the index of the stage the current stage is built off
// returns -1 if the current stage isn't built off a previous stage
func baseImageIndex(currentStage int, stages []instructions.Stage) int {
	currentStageBaseName := strings.ToLower(stages[currentStage].BaseName)

	for i, stage := range stages {
		if i >= currentStage {
			break
		}
		if stage.Name == currentStageBaseName {
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
	stages, metaArgs, err := instructions.Parse(p.AST, &linter.Linter{})
	if err != nil {
		return nil, nil, err
	}

	metaArgs, err = stripEnclosingQuotes(metaArgs)
	if err != nil {
		return nil, nil, err
	}

	return stages, metaArgs, nil
}

// expandNestedArgs tries to resolve nested ARG value against the previously defined ARGs
func expandNestedArgs(metaArgs []instructions.ArgCommand, buildArgs []string) ([]instructions.ArgCommand, error) {
	var prevArgs []string
	for i, marg := range metaArgs {
		for j, arg := range marg.Args {
			v := arg.Value
			if v != nil {
				val, err := util.ResolveEnvironmentReplacement(*v, append(prevArgs, buildArgs...), false)
				if err != nil {
					return nil, err
				}
				prevArgs = append(prevArgs, arg.Key+"="+val)
				arg.Value = &val
				metaArgs[i].Args[j] = arg
			}
		}
	}
	return metaArgs, nil
}

// stripEnclosingQuotes removes quotes enclosing the value of each instructions.ArgCommand in a slice
// if the quotes are escaped it leaves them
func stripEnclosingQuotes(metaArgs []instructions.ArgCommand) ([]instructions.ArgCommand, error) {
	for i, marg := range metaArgs {
		for j, arg := range marg.Args {
			v := arg.Value
			if v != nil {
				val, err := extractValFromQuotes(*v)
				if err != nil {
					return nil, err
				}

				arg.Value = &val
				metaArgs[i].Args[j] = arg
			}
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
		logrus.Infof("Leader %s tail %s", leader, tail)
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
		if strings.EqualFold(stage.Name, target) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("%s is not a valid target build stage", target)
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
	currentStageName := stages[index].Name

	for stageIndex, stage := range stages {
		if stageIndex <= index {
			continue
		}

		if strings.ToLower(stage.BaseName) == currentStageName {
			if stage.BaseName != "" {
				return true
			}
		}
	}

	return false
}

// ResolveCrossStageCommands resolves any calls to previous stages with names to indices
// Ex. --from=secondStage should be --from=1 for easier processing later on
// As third party library lowers stage name in FROM instruction, this function resolves stage case insensitively.
func ResolveCrossStageCommands(cmds []instructions.Command, stageNameToIdx map[string]string) {
	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case *instructions.CopyCommand:
			if c.From != "" {
				if val, ok := stageNameToIdx[strings.ToLower(c.From)]; ok {
					c.From = val
				}
			}
		}
	}
}

// resolveStagesArgs resolves all the args from list of stages
func resolveStagesArgs(stages []instructions.Stage, args []string) error {
	for i, s := range stages {
		resolvedBaseName, err := util.ResolveEnvironmentReplacement(s.BaseName, args, false)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("resolving base name %s", s.BaseName))
		}
		if s.BaseName != resolvedBaseName {
			stages[i].BaseName = resolvedBaseName
		}
	}
	return nil
}

func MakeKanikoStages(opts *config.KanikoOptions, stages []instructions.Stage, metaArgs []instructions.ArgCommand) ([]config.KanikoStage, error) {
	targetStage, err := targetStage(stages, opts.Target)
	if err != nil {
		return nil, errors.Wrap(err, "Error finding target stage")
	}
	args := unifyArgs(metaArgs, opts.BuildArgs)
	if err := resolveStagesArgs(stages, args); err != nil {
		return nil, errors.Wrap(err, "resolving args")
	}
	if opts.SkipUnusedStages {
		stages = skipUnusedStages(stages, &targetStage, opts.Target)
	}
	var kanikoStages []config.KanikoStage
	for index, stage := range stages {
		if len(stage.Name) > 0 {
			logrus.Infof("Resolved base name %s to %s", stage.BaseName, stage.Name)
		}
		baseImageIndex := baseImageIndex(index, stages)
		kanikoStages = append(kanikoStages, config.KanikoStage{
			Stage:                  stage,
			BaseImageIndex:         baseImageIndex,
			BaseImageStoredLocally: (baseImageIndex != -1),
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

func GetOnBuildInstructions(config *v1.Config, stageNameToIdx map[string]string) ([]instructions.Command, error) {
	if config.OnBuild == nil || len(config.OnBuild) == 0 {
		return nil, nil
	}

	cmds, err := ParseCommands(config.OnBuild)
	if err != nil {
		return nil, err
	}

	// Iterate over commands and replace references to other stages with their index
	ResolveCrossStageCommands(cmds, stageNameToIdx)
	return cmds, nil
}

// unifyArgs returns the unified args between metaArgs and --build-arg
// by default --build-arg overrides metaArgs except when --build-arg is empty
func unifyArgs(metaArgs []instructions.ArgCommand, buildArgs []string) []string {
	argsMap := make(map[string]string)
	for _, marg := range metaArgs {
		for _, arg := range marg.Args {
			if arg.Value != nil {
				argsMap[arg.Key] = *arg.Value
			}
		}
	}
	splitter := "="
	for _, a := range buildArgs {
		s := strings.Split(a, splitter)
		if len(s) > 1 && s[1] != "" {
			argsMap[s[0]] = s[1]
		}
	}
	var args []string
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}
	return args
}

// skipUnusedStages returns the list of used stages without the unnecessaries ones
func skipUnusedStages(stages []instructions.Stage, lastStageIndex *int, target string) []instructions.Stage {
	stagesDependencies := make(map[string]bool)
	var onlyUsedStages []instructions.Stage
	idx := *lastStageIndex

	lastStageBaseName := stages[idx].BaseName

	for i := idx; i >= 0; i-- {
		s := stages[i]
		if (s.Name != "" && stagesDependencies[s.Name]) || s.Name == lastStageBaseName || i == idx {
			for _, c := range s.Commands {
				switch cmd := c.(type) {
				case *instructions.CopyCommand:
					stageName := cmd.From
					if copyFromIndex, err := strconv.Atoi(stageName); err == nil {
						stageName = stages[copyFromIndex].Name
					}
					if !stagesDependencies[stageName] {
						stagesDependencies[stageName] = true
					}
				}
			}
			if i != idx {
				stagesDependencies[s.BaseName] = true
			}
		}
	}
	dependenciesLen := len(stagesDependencies)
	if target == "" && dependenciesLen == 0 {
		return stages
	} else if dependenciesLen > 0 {
		for i := 0; i < idx; i++ {
			if stages[i].Name == "" {
				continue
			}
			s := stages[i]
			if stagesDependencies[s.Name] || s.Name == lastStageBaseName {
				onlyUsedStages = append(onlyUsedStages, s)
			}
		}
	}
	onlyUsedStages = append(onlyUsedStages, stages[idx])
	if idx > len(onlyUsedStages)-1 {
		*lastStageIndex = len(onlyUsedStages) - 1
	}

	return onlyUsedStages
}
