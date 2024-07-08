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

package util

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
)

// for testing
var (
	getUIDAndGIDFunc = getUIDAndGID
)

const (
	pathSeparator = "/"
)

// ResolveEnvironmentReplacementList resolves a list of values by calling resolveEnvironmentReplacement
func ResolveEnvironmentReplacementList(values, envs []string, isFilepath bool) ([]string, error) {
	var resolvedValues []string
	for _, value := range values {
		resolved, err := ResolveEnvironmentReplacement(value, envs, isFilepath)
		logrus.Debugf("Resolved %s to %s", value, resolved)
		if err != nil {
			return nil, err
		}
		resolvedValues = append(resolvedValues, resolved)
	}
	return resolvedValues, nil
}

// ResolveEnvironmentReplacement resolves replacing env variables in some text from envs
// It takes in a string representation of the command, the value to be resolved, and a list of envs (config.Env)
// Ex: value = $foo/newdir, envs = [foo=/foodir], then this should return /foodir/newdir
// The dockerfile/shell package handles processing env values
// It handles escape characters and supports expansion from the config.Env array
// Shlex handles some of the following use cases (these and more are tested in integration tests)
// ""a'b'c"" -> "a'b'c"
// "Rex\ The\ Dog \" -> "Rex The Dog"
// "a\"b" -> "a"b"
func ResolveEnvironmentReplacement(value string, envs []string, isFilepath bool) (string, error) {
	shlex := shell.NewLex(parser.DefaultEscapeToken)
	fp, _, err := shlex.ProcessWord(value, envs)
	// Check after replacement if value is a remote URL
	if !isFilepath || IsSrcRemoteFileURL(fp) {
		return fp, err
	}
	if err != nil {
		return "", err
	}
	isDir := strings.HasSuffix(fp, pathSeparator)
	fp = filepath.Clean(fp)
	if isDir && !strings.HasSuffix(fp, pathSeparator) {
		fp = fp + pathSeparator
	}
	return fp, nil
}

func ResolveEnvAndWildcards(sd instructions.SourcesAndDest, fileContext FileContext, envs []string) ([]string, string, error) {
	// First, resolve any environment replacement
	resolvedEnvs, err := ResolveEnvironmentReplacementList(sd.SourcePaths, envs, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to resolve environment")
	}
	if len(resolvedEnvs) == 0 {
		return nil, "", errors.New("resolved envs is empty")
	}
	dests, err := ResolveEnvironmentReplacementList([]string{sd.DestPath}, envs, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to resolve environment for dest path")
	}
	dest := dests[0]
	sd.DestPath = dest
	// Resolve wildcards and get a list of resolved sources
	srcs, err := ResolveSources(resolvedEnvs, fileContext.Root)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to resolve sources")
	}
	err = IsSrcsValid(sd, srcs, fileContext)
	return srcs, dest, err
}

// ContainsWildcards returns true if any entry in paths contains wildcards
func ContainsWildcards(paths []string) bool {
	for _, path := range paths {
		if strings.ContainsAny(path, "*?[") {
			return true
		}
	}
	return false
}

// ResolveSources resolves the given sources if the sources contains wildcards
// It returns a list of resolved sources
func ResolveSources(srcs []string, root string) ([]string, error) {
	// If sources contain wildcards, we first need to resolve them to actual paths
	if !ContainsWildcards(srcs) {
		return srcs, nil
	}
	logrus.Infof("Resolving srcs %v...", srcs)
	files, err := RelativeFiles("", root)
	if err != nil {
		return nil, errors.Wrap(err, "resolving sources")
	}
	resolved, err := matchSources(srcs, files)
	if err != nil {
		return nil, errors.Wrap(err, "matching sources")
	}
	logrus.Debugf("Resolved sources to %v", resolved)
	return resolved, nil
}

// matchSources returns a list of sources that match wildcards
func matchSources(srcs, files []string) ([]string, error) {
	var matchedSources []string
	for _, src := range srcs {
		if IsSrcRemoteFileURL(src) {
			matchedSources = append(matchedSources, src)
			continue
		}
		src = filepath.Clean(src)
		for _, file := range files {
			if filepath.IsAbs(src) {
				file = filepath.Join(config.RootDir, file)
			}
			matched, err := filepath.Match(src, file)
			if err != nil {
				return nil, err
			}
			if matched || src == file {
				matchedSources = append(matchedSources, file)
			}
		}
	}
	return matchedSources, nil
}

func IsDestDir(path string) bool {
	// try to stat the path
	fileInfo, err := os.Stat(path)
	if err != nil {
		// fall back to string-based determination
		return strings.HasSuffix(path, pathSeparator) || path == "."
	}
	// if it's a real path, check the fs response
	return fileInfo.IsDir()
}

// DestinationFilepath returns the destination filepath from the build context to the image filesystem
// If source is a file:
//
//	If dest is a dir, copy it to /dest/relpath
//	If dest is a file, copy directly to dest
//
// If source is a dir:
//
//	Assume dest is also a dir, and copy to dest/
//
// If dest is not an absolute filepath, add /cwd to the beginning
func DestinationFilepath(src, dest, cwd string) (string, error) {
	_, srcFileName := filepath.Split(src)
	newDest := dest

	if !filepath.IsAbs(newDest) {
		newDest = filepath.Join(cwd, newDest)
		// join call clean on all results.
		if strings.HasSuffix(dest, pathSeparator) || strings.HasSuffix(dest, ".") {
			newDest += pathSeparator
		}
	}
	if IsDestDir(newDest) {
		newDest = filepath.Join(newDest, srcFileName)
	}

	if len(srcFileName) <= 0 && !strings.HasSuffix(newDest, pathSeparator) {
		newDest += pathSeparator
	}

	return newDest, nil
}

// URLDestinationFilepath gives the destination a file from a remote URL should be saved to
func URLDestinationFilepath(rawurl, dest, cwd string, envs []string) (string, error) {
	if !IsDestDir(dest) {
		if !filepath.IsAbs(dest) {
			return filepath.Join(cwd, dest), nil
		}
		return dest, nil
	}

	urlBase, err := ResolveEnvironmentReplacement(rawurl, envs, true)
	if err != nil {
		return "", err
	}

	urlBase, err = extractFilename(urlBase)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(dest, urlBase)

	if !filepath.IsAbs(dest) {
		destPath = filepath.Join(cwd, destPath)
	}
	return destPath, nil
}

func IsSrcsValid(srcsAndDest instructions.SourcesAndDest, resolvedSources []string, fileContext FileContext) error {
	srcs := srcsAndDest.SourcePaths
	dest := srcsAndDest.DestPath

	if !ContainsWildcards(srcs) {
		totalSrcs := 0
		for _, src := range srcs {
			if fileContext.ExcludesFile(src) {
				continue
			}
			totalSrcs++
		}
		if totalSrcs > 1 && !IsDestDir(dest) {
			return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
		}
	}

	// If there is only one source and it's a directory, docker assumes the dest is a directory
	if len(resolvedSources) == 1 {
		if IsSrcRemoteFileURL(resolvedSources[0]) {
			return nil
		}
		path := filepath.Join(fileContext.Root, resolvedSources[0])
		fi, err := os.Lstat(path)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get fileinfo for %v", path))
		}
		if fi.IsDir() {
			return nil
		}
	}

	totalFiles := 0
	for _, src := range resolvedSources {
		if IsSrcRemoteFileURL(src) {
			totalFiles++
			continue
		}
		src = filepath.Clean(src)
		files, err := RelativeFiles(src, fileContext.Root)
		if err != nil {
			return errors.Wrap(err, "failed to get relative files")
		}
		for _, file := range files {
			if fileContext.ExcludesFile(file) {
				continue
			}
			totalFiles++
		}
	}
	// ignore the case where whildcards and there are no files to copy
	if totalFiles == 0 {
		// using log warning instead of return errors.New("copy failed: no source files specified")
		logrus.Warn("No files to copy")
	}
	// If there are wildcards, and the destination is a file, there must be exactly one file to copy over,
	// Otherwise, return an error
	if !IsDestDir(dest) && totalFiles > 1 {
		return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
	}
	return nil
}

func IsSrcRemoteFileURL(rawurl string) bool {
	u, err := url.ParseRequestURI(rawurl)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func UpdateConfigEnv(envVars []instructions.KeyValuePair, config *v1.Config, replacementEnvs []string) error {
	newEnvs := make([]instructions.KeyValuePair, len(envVars))
	for index, pair := range envVars {
		expandedKey, err := ResolveEnvironmentReplacement(pair.Key, replacementEnvs, false)
		if err != nil {
			return err
		}
		expandedValue, err := ResolveEnvironmentReplacement(pair.Value, replacementEnvs, false)
		if err != nil {
			return err
		}
		newEnvs[index] = instructions.KeyValuePair{
			Key:   expandedKey,
			Value: expandedValue,
		}
	}

	// First, convert config.Env array to []instruction.KeyValuePair
	var kvps []instructions.KeyValuePair
	for _, env := range config.Env {
		entry := strings.SplitN(env, "=", 2)
		kvps = append(kvps, instructions.KeyValuePair{
			Key:   entry[0],
			Value: entry[1],
		})
	}
	// Iterate through new environment variables, and replace existing keys
	// We can't use a map because we need to preserve the order of the environment variables
Loop:
	for _, newEnv := range newEnvs {
		for index, kvp := range kvps {
			// If key exists, replace the KeyValuePair...
			if kvp.Key == newEnv.Key {
				logrus.Debugf("Replacing environment variable %v with %v in config", kvp, newEnv)
				kvps[index] = newEnv
				continue Loop
			}
		}
		// ... Else, append it as a new env variable
		kvps = append(kvps, newEnv)
	}
	// Convert back to array and set in config
	envArray := []string{}
	for _, kvp := range kvps {
		entry := kvp.Key + "=" + kvp.Value
		envArray = append(envArray, entry)
	}
	config.Env = envArray
	return nil
}

func GetUserGroup(chownStr string, env []string) (int64, int64, error) {
	if chownStr == "" {
		return DoNotChangeUID, DoNotChangeGID, nil
	}

	chown, err := ResolveEnvironmentReplacement(chownStr, env, false)
	if err != nil {
		return -1, -1, err
	}

	uid32, gid32, err := getUIDAndGIDFromString(chown)
	if err != nil {
		return -1, -1, err
	}

	return int64(uid32), int64(gid32), nil
}

func GetChmod(chmodStr string, env []string) (chmod fs.FileMode, useDefault bool, err error) {
	if chmodStr == "" {
		return fs.FileMode(0o600), true, nil
	}

	chmodStr, err = ResolveEnvironmentReplacement(chmodStr, env, false)
	if err != nil {
		return 0, false, err
	}

	mode, err := strconv.ParseUint(chmodStr, 8, 32)
	if err != nil {
		return 0, false, errors.Wrap(err, "parsing value from chmod")
	}
	chmod = fs.FileMode(mode)
	return
}

// Extract user and group id from a string formatted 'user:group'.
// UserID and GroupID don't need to be present on the system.
func getUIDAndGIDFromString(userGroupString string) (uint32, uint32, error) {
	userAndGroup := strings.Split(userGroupString, ":")
	userStr := userAndGroup[0]
	var groupStr string
	if len(userAndGroup) > 1 {
		groupStr = userAndGroup[1]
	}
	return getUIDAndGIDFunc(userStr, groupStr)
}

func getUIDAndGID(userStr string, groupStr string) (uint32, uint32, error) {
	user, err := LookupUser(userStr)
	if err != nil {
		return 0, 0, err
	}
	uid32, err := getUID(user.Uid)
	if err != nil {
		return 0, 0, err
	}

	if groupStr != "" {
		gid32, err := getGIDFromName(groupStr)
		if err != nil {
			if errors.Is(err, fallbackToUIDError) {
				return uid32, uid32, nil
			}
			return 0, 0, err
		}
		return uid32, gid32, nil
	}

	return uid32, uid32, nil
}

// getGID tries to parse the gid
func getGID(groupStr string) (uint32, error) {
	gid, err := strconv.ParseUint(groupStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(gid), nil
}

// getGIDFromName tries to parse the groupStr into an existing group.
func getGIDFromName(groupStr string) (uint32, error) {
	group, err := user.LookupGroup(groupStr)
	if err != nil {
		// unknown group error could relate to a non existing group
		var groupErr user.UnknownGroupError
		if errors.As(err, &groupErr) {
			return getGID(groupStr)
		}
		group, err = user.LookupGroupId(groupStr)
		if err != nil {
			return getGID(groupStr)
		}
	}
	return getGID(group.Gid)
}

var fallbackToUIDError = new(fallbackToUIDErrorType)

type fallbackToUIDErrorType struct{}

func (e fallbackToUIDErrorType) Error() string {
	return "fallback to uid"
}

// LookupUser will try to lookup the userStr inside the passwd file.
// If the user does not exists, the function will fallback to parsing the userStr as an uid.
func LookupUser(userStr string) (*user.User, error) {
	userObj, err := user.Lookup(userStr)
	if err != nil {
		unknownUserErr := new(user.UnknownUserError)
		// only return if it's not an unknown user error or the passwd file does not exist
		if !errors.As(err, unknownUserErr) && !os.IsNotExist(err) {
			return nil, err
		}

		// Lookup by id
		userObj, err = user.LookupId(userStr)
		if err != nil {
			uid, err := getUID(userStr)
			if err != nil {
				// at this point, the user does not exist and the userStr is not a valid number.
				return nil, fmt.Errorf("user %v is not a uid and does not exist on the system", userStr)
			}
			userObj = &user.User{
				Uid:     fmt.Sprint(uid),
				HomeDir: "/",
			}
		}
	}
	return userObj, nil
}

func getUID(userStr string) (uint32, error) {
	// checkif userStr is a valid id
	uid, err := strconv.ParseUint(userStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(uid), nil
}

// ExtractFilename extracts the filename from a URL without its query url
func extractFilename(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	filename := filepath.Base(parsedURL.Path)
	return filename, nil
}
