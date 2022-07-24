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
	"net/http"
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

	chrootuser "github.com/GoogleContainerTools/kaniko/pkg/chroot/user"
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
	fp, err := shlex.ProcessWord(value, envs)
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
	resolvedEnvs, err := ResolveEnvironmentReplacementList(sd, envs, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to resolve environment")
	}
	if len(resolvedEnvs) == 0 {
		return nil, "", errors.New("resolved envs is empty")
	}
	dest := resolvedEnvs[len(resolvedEnvs)-1]
	// Resolve wildcards and get a list of resolved sources
	srcs, err := ResolveSources(resolvedEnvs[0:len(resolvedEnvs)-1], fileContext.Root)
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
	resolved, err := matchSources(root, srcs, files)
	if err != nil {
		return nil, errors.Wrap(err, "matching sources")
	}
	logrus.Debugf("Resolved sources to %v", resolved)
	return resolved, nil
}

// matchSources returns a list of sources that match wildcards
func matchSources(rootDir string, srcs, files []string) ([]string, error) {
	var matchedSources []string
	for _, src := range srcs {
		if IsSrcRemoteFileURL(src) {
			matchedSources = append(matchedSources, src)
			continue
		}
		src = filepath.Clean(src)
		for _, file := range files {
			if filepath.IsAbs(src) {
				file = filepath.Join(rootDir, file)
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
//	If dest is a dir, copy it to /dest/relpath
// 	If dest is a file, copy directly to dest
// If source is a dir:
//	Assume dest is also a dir, and copy to dest/
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
	urlBase := filepath.Base(rawurl)
	urlBase, err := ResolveEnvironmentReplacement(urlBase, envs, true)
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
	srcs := srcsAndDest[:len(srcsAndDest)-1]
	dest := srcsAndDest[len(srcsAndDest)-1]

	if !ContainsWildcards(srcs) {
		totalSrcs := 0
		for _, src := range srcs {
			if fileContext.ExcludesFile(src) {
				continue
			}
			totalSrcs++
		}
		if totalSrcs > 1 && !IsDestDir(dest) {
			return errors.New(
				"when specifying multiple sources in a COPY command, destination must be a directory and end in '/'",
			)
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
	if totalFiles == 0 {
		return errors.New("copy failed: no source files specified")
	}
	// If there are wildcards, and the destination is a file, there must be exactly one file to copy over,
	// Otherwise, return an error
	if !IsDestDir(dest) && totalFiles > 1 {
		return errors.New("when specifying multiple sources in a COPY command, destination must be a directory and end in '/'")
	}
	return nil
}

func IsSrcRemoteFileURL(rawurl string) bool {
	_, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return false
	}
	_, err = http.Get(rawurl)
	return err == nil
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

func GetUserGroup(rootDir string, chownStr string, env []string) (int64, int64, error) {
	if chownStr == "" {
		return DoNotChangeUID, DoNotChangeGID, nil
	}

	chown, err := ResolveEnvironmentReplacement(chownStr, env, false)
	if err != nil {
		return -1, -1, err
	}

	uid32, gid32, err := getUIDAndGIDFromString(rootDir, chown, true)
	if err != nil {
		return -1, -1, err
	}

	return int64(uid32), int64(gid32), nil
}

// Extract user and group id from a string formatted 'user:group'.
// If fallbackToUID is set, the gid is equal to uid if the group is not specified
// otherwise gid is set to zero.
// UserID and GroupID don't need to be present on the system.
func getUIDAndGIDFromString(rootDir string, userGroupString string, fallbackToUID bool) (uint32, uint32, error) {
	userAndGroup := strings.Split(userGroupString, ":")
	userStr := userAndGroup[0]
	var groupStr string
	if len(userAndGroup) > 1 {
		groupStr = userAndGroup[1]
	}
	return getUIDAndGIDFunc(rootDir, userStr, groupStr, fallbackToUID)
}

func getUIDAndGID(rootDir string, userStr string, groupStr string, fallbackToUID bool) (uint32, uint32, error) {
	user, err := LookupUser(rootDir, userStr)
	if err != nil {
		return 0, 0, err
	}
	uid32, err := parseUID(user.Uid)
	if err != nil {
		return 0, 0, err
	}

	gid, err := getGIDFromName(rootDir, groupStr, fallbackToUID)
	if err != nil {
		if errors.Is(err, fallbackErr) {
			return uid32, uid32, nil
		}

		return 0, 0, err
	}
	return uid32, gid, nil
}

// parseGID tries to parse the gid or falls back to getGroupFromName if it's not an id
func parseGID(groupStr string, fallbackToUID bool) (uint32, error) {
	gid, err := strconv.ParseUint(groupStr, 10, 32)
	if err != nil {
		return 0, fallbackToUIDOrError(err, fallbackToUID)
	}
	return uint32(gid), nil
}

// getGIDFromName tries to parse the groupStr into an existing group.
// if the group doesn't exist, fallback to getGID to parse non-existing valid GIDs.
func getGIDFromName(rootDir string, groupStr string, fallbackToUID bool) (uint32, error) {
	group, err := lookupGroup(rootDir, groupStr)
	if err != nil {
		// at this point user.LookupGroup and user.LookupGroudId failed, so the group doesnt exist
		// try to raw parse the groupStr as a gid
		return parseGID(groupStr, fallbackToUID)
	}
	return parseGID(group.Gid, fallbackToUID)
}

// lookupGroup will use chrootuser.GetGroup function to get the group
// This is done to get the group struct in environments, where the group file is not at /etc/group
func lookupGroup(rootDir string, group string) (*user.Group, error) {
	if rootDir != "/" {
		logrus.Debugf("detected chroot environment, getting group %v via chrootuser pkg", group)
		group, err := chrootuser.GetGroup(rootDir, group)
		if err != nil {
			return nil, fmt.Errorf("getting group %v: %w", group, err)
		}
		return group, nil
	}
	grp, err := user.LookupGroup(group)
	if err != nil {
		var unknownGroupErr user.UnknownGroupError
		// if error is of type unknownUserError, try to look up the user with Id
		if errors.As(err, &unknownGroupErr) {
			grp, err = user.LookupGroupId(group)
		}
	}
	return grp, err
}

var fallbackErr = fallbackToUIDError{}

type fallbackToUIDError struct{}

func (e fallbackToUIDError) Error() string {
	return "fallback to uid"
}

func fallbackToUIDOrError(err error, fallbackToUID bool) error {
	if fallbackToUID {
		return fallbackErr
	}
	return err
}

// LookupUser will try to lookup the userStr inside the passwd file.
// If the user does not exists, the function will fallback to parsing the userStr as an uid.
func LookupUser(rootDir, userStr string) (*user.User, error) {
	userObj, err := lookupUser(rootDir, userStr)
	if err != nil {
		// at this point, user.LookupUser and user.LookupId failed, so try to parse userStr as a raw uid
		uid, err := parseUID(userStr)
		if err != nil {
			// at this point, the user does not exist and the userStr is not a valid number.
			return nil, fmt.Errorf("user %v is not a uid and does not exist on the system", userStr)
		}
		return &user.User{
			Uid:     fmt.Sprint(uid),
			HomeDir: "/",
		}, nil
	}
	return userObj, nil
}

// lookupUser will lookup userStr as username or as uid if username lookup fails.
// In situations where rootdir != / use chrootuser.GetUser function to get the current user.
// This is done to get the user struct in environments, where the passwd file is not at /etc/passwd.
func lookupUser(rootDir string, userStr string) (*user.User, error) {
	if rootDir != "/" {
		logrus.Debugf("detected chroot environment, getting user %v via chrootuser pkg", userStr)
		user, err := chrootuser.GetUser(rootDir, userStr)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	u, err := user.Lookup(userStr)
	if err != nil {
		var unknownUserErr user.UnknownUserError
		// if error is of type unknownUserError, try to look up the user with Id
		if errors.As(err, &unknownUserErr) {
			logrus.Debugf("user lookup with unknownUserError, try lookup %v as uid", userStr)
			u, err = user.LookupId(userStr)
		}
	}
	return u, err
}

func parseUID(userStr string) (uint32, error) {
	// checkif userStr is a valid id
	uid, err := strconv.ParseUint(userStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(uid), nil
}
