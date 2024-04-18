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
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/timing"
	"github.com/docker/docker/pkg/archive"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/karrick/godirwalk"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/moby/patternmatcher"
	otiai10Cpy "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DoNotChangeUID = -1
	DoNotChangeGID = -1
)

const (
	snapshotTimeout = "SNAPSHOT_TIMEOUT_DURATION"
	defaultTimeout  = "90m"
)

type IgnoreListEntry struct {
	Path            string
	PrefixMatchOnly bool
}

var defaultIgnoreList = []IgnoreListEntry{
	{
		Path:            filepath.Clean(config.KanikoDir),
		PrefixMatchOnly: false,
	},
	{
		// similarly, we ignore /etc/mtab, since there is no way to know if the file was mounted or came
		// from the base image
		Path:            "/etc/mtab",
		PrefixMatchOnly: false,
	},
	{
		// we ingore /tmp/apt-key-gpghome, since the apt keys are added temporarily in this directory.
		// from the base image
		Path:            "/tmp/apt-key-gpghome",
		PrefixMatchOnly: true,
	},
}

var ignorelist = append([]IgnoreListEntry{}, defaultIgnoreList...)

var volumes = []string{}

// skipKanikoDir opts to skip the '/kaniko' dir for otiai10.copy which should be ignored in root
var skipKanikoDir = otiai10Cpy.Options{
	Skip: func(info os.FileInfo, src, dest string) (bool, error) {
		return strings.HasSuffix(src, "/kaniko"), nil
	},
}

type FileContext struct {
	Root          string
	ExcludedFiles []string
}

type ExtractFunction func(string, *tar.Header, string, io.Reader) error

type FSConfig struct {
	includeWhiteout bool
	extractFunc     ExtractFunction
}

type FSOpt func(*FSConfig)

func IgnoreList() []IgnoreListEntry {
	return ignorelist
}

func AddToIgnoreList(entry IgnoreListEntry) {
	ignorelist = append(ignorelist, IgnoreListEntry{
		Path:            filepath.Clean(entry.Path),
		PrefixMatchOnly: entry.PrefixMatchOnly,
	})
}

func AddToDefaultIgnoreList(entry IgnoreListEntry) {
	defaultIgnoreList = append(defaultIgnoreList, IgnoreListEntry{
		Path:            filepath.Clean(entry.Path),
		PrefixMatchOnly: entry.PrefixMatchOnly,
	})
}

func IncludeWhiteout() FSOpt {
	return func(opts *FSConfig) {
		opts.includeWhiteout = true
	}
}

func ExtractFunc(extractFunc ExtractFunction) FSOpt {
	return func(opts *FSConfig) {
		opts.extractFunc = extractFunc
	}
}

// GetFSFromImage extracts the layers of img to root
// It returns a list of all files extracted
func GetFSFromImage(root string, img v1.Image, extract ExtractFunction) ([]string, error) {
	if img == nil {
		return nil, errors.New("image cannot be nil")
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	return GetFSFromLayers(root, layers, ExtractFunc(extract))
}

func GetFSFromLayers(root string, layers []v1.Layer, opts ...FSOpt) ([]string, error) {
	volumes = []string{}
	cfg := new(FSConfig)
	if err := InitIgnoreList(); err != nil {
		return nil, errors.Wrap(err, "initializing filesystem ignore list")
	}
	logrus.Debugf("Ignore list: %v", ignorelist)

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.extractFunc == nil {
		return nil, errors.New("must supply an extract function")
	}

	extractedFiles := []string{}
	for i, l := range layers {
		if mediaType, err := l.MediaType(); err == nil {
			logrus.Tracef("Extracting layer %d of media type %s", i, mediaType)
		} else {
			logrus.Tracef("Extracting layer %d", i)
		}

		r, err := l.Uncompressed()
		if err != nil {
			return nil, err
		}
		defer r.Close()

		tr := tar.NewReader(r)
		for {
			hdr, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("error reading tar %d", i))
			}

			cleanedName := filepath.Clean(hdr.Name)
			path := filepath.Join(root, cleanedName)
			base := filepath.Base(path)
			dir := filepath.Dir(path)

			if strings.HasPrefix(base, archive.WhiteoutPrefix) {
				logrus.Tracef("Whiting out %s", path)

				name := strings.TrimPrefix(base, archive.WhiteoutPrefix)
				path := filepath.Join(dir, name)

				if CheckCleanedPathAgainstIgnoreList(path) {
					logrus.Tracef("Not deleting %s, as it's ignored", path)
					continue
				}
				if childDirInIgnoreList(path) {
					logrus.Tracef("Not deleting %s, as it contains a ignored path", path)
					continue
				}

				if err := os.RemoveAll(path); err != nil {
					return nil, errors.Wrapf(err, "removing whiteout %s", hdr.Name)
				}

				if !cfg.includeWhiteout {
					logrus.Trace("Not including whiteout files")
					continue
				}

			}

			if err := cfg.extractFunc(root, hdr, cleanedName, tr); err != nil {
				return nil, err
			}

			extractedFiles = append(extractedFiles, filepath.Join(root, cleanedName))
		}
	}
	return extractedFiles, nil
}

// DeleteFilesystem deletes the extracted image file system
func DeleteFilesystem() error {
	logrus.Info("Deleting filesystem...")
	return filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// ignore errors when deleting.
			return nil //nolint:nilerr
		}

		if CheckCleanedPathAgainstIgnoreList(path) {
			if !isExist(path) {
				logrus.Debugf("Path %s ignored, but not exists", path)
				return nil
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			logrus.Debugf("Not deleting %s, as it's ignored", path)
			return nil
		}
		if childDirInIgnoreList(path) {
			logrus.Debugf("Not deleting %s, as it contains a ignored path", path)
			return nil
		}
		if path == config.RootDir {
			return nil
		}
		return os.RemoveAll(path)
	})
}

// isExists returns true if path exists
func isExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// childDirInIgnoreList returns true if there is a child file or directory of the path in the ignorelist
func childDirInIgnoreList(path string) bool {
	for _, d := range ignorelist {
		if HasFilepathPrefix(d.Path, path, d.PrefixMatchOnly) {
			return true
		}
	}
	return false
}

// UnTar returns a list of files that have been extracted from the tar archive at r to the path at dest
func UnTar(r io.Reader, dest string) ([]string, error) {
	var extractedFiles []string
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		cleanedName := filepath.Clean(hdr.Name)
		if err := ExtractFile(dest, hdr, cleanedName, tr); err != nil {
			return nil, err
		}
		extractedFiles = append(extractedFiles, filepath.Join(dest, cleanedName))
	}
	return extractedFiles, nil
}

func ExtractFile(dest string, hdr *tar.Header, cleanedName string, tr io.Reader) error {
	path := filepath.Join(dest, cleanedName)
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	mode := hdr.FileInfo().Mode()
	uid := hdr.Uid
	gid := hdr.Gid

	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if CheckCleanedPathAgainstIgnoreList(abs) && !checkIgnoreListRoot(dest) {
		logrus.Debugf("Not adding %s because it is ignored", path)
		return nil
	}
	switch hdr.Typeflag {
	case tar.TypeReg:
		logrus.Tracef("Creating file %s", path)

		// It's possible a file is in the tar before its directory,
		// or a file was copied over a directory prior to now
		fi, err := os.Stat(dir)
		if os.IsNotExist(err) || !fi.IsDir() {
			logrus.Debugf("Base %s for file %s does not exist. Creating.", base, path)

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}

		// Check if something already exists at path (symlinks etc.)
		// If so, delete it
		if FilepathExists(path) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new file.", path)
			}
		}

		currFile, err := os.Create(path)
		if err != nil {
			return err
		}

		if _, err = io.Copy(currFile, tr); err != nil {
			return err
		}

		if err = setFilePermissions(path, mode, uid, gid); err != nil {
			return err
		}

		if err = writeSecurityXattrToTarFile(path, hdr); err != nil {
			return err
		}

		if err = setFileTimes(path, hdr.AccessTime, hdr.ModTime); err != nil {
			return err
		}

		currFile.Close()
	case tar.TypeDir:
		logrus.Tracef("Creating dir %s", path)
		if err := MkdirAllWithPermissions(path, mode, int64(uid), int64(gid)); err != nil {
			return err
		}

	case tar.TypeLink:
		logrus.Tracef("Link from %s to %s", hdr.Linkname, path)
		abs, err := filepath.Abs(hdr.Linkname)
		if err != nil {
			return err
		}
		if CheckCleanedPathAgainstIgnoreList(abs) {
			logrus.Tracef("Skipping link from %s to %s because %s is ignored", hdr.Linkname, path, hdr.Linkname)
			return nil
		}
		// The base directory for a link may not exist before it is created.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if FilepathExists(path) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new link", hdr.Name)
			}
		}
		link := filepath.Clean(filepath.Join(dest, hdr.Linkname))
		if err := os.Link(link, path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		logrus.Tracef("Symlink from %s to %s", hdr.Linkname, path)
		// The base directory for a symlink may not exist before it is created.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		// Check if something already exists at path
		// If so, delete it
		if FilepathExists(path) {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrapf(err, "error removing %s to make way for new symlink", hdr.Name)
			}
		}
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}
	}
	return nil
}

func IsInProvidedIgnoreList(path string, wl []IgnoreListEntry) bool {
	path = filepath.Clean(path)
	for _, entry := range wl {
		if !entry.PrefixMatchOnly && path == entry.Path {
			return true
		}
	}

	return false
}

func IsInIgnoreList(path string) bool {
	return IsInProvidedIgnoreList(path, ignorelist)
}

func CheckCleanedPathAgainstProvidedIgnoreList(path string, wl []IgnoreListEntry) bool {
	for _, wl := range ignorelist {
		if hasCleanedFilepathPrefix(path, wl.Path, wl.PrefixMatchOnly) {
			return true
		}
	}

	return false
}

func CheckIgnoreList(path string) bool {
	return CheckCleanedPathAgainstIgnoreList(filepath.Clean(path))
}

func CheckCleanedPathAgainstIgnoreList(path string) bool {
	return CheckCleanedPathAgainstProvidedIgnoreList(path, ignorelist)
}

func checkIgnoreListRoot(root string) bool {
	if root == config.RootDir {
		return false
	}
	return CheckIgnoreList(root)
}

// Get ignorelist from roots of mounted files
// Each line of /proc/self/mountinfo is in the form:
// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)
// Where (5) is the mount point relative to the process's root
// From: https://www.kernel.org/doc/Documentation/filesystems/proc.txt
func DetectFilesystemIgnoreList(path string) error {
	logrus.Trace("Detecting filesystem ignore list")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		logrus.Tracef("Read the following line from %s: %s", path, line)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		lineArr := strings.Split(line, " ")
		if len(lineArr) < 5 {
			if err == io.EOF {
				logrus.Tracef("Reached end of file %s", path)
				break
			}
			continue
		}
		if lineArr[4] != config.RootDir {
			logrus.Tracef("Adding ignore list entry %s from line: %s", lineArr[4], line)
			AddToIgnoreList(IgnoreListEntry{
				Path:            lineArr[4],
				PrefixMatchOnly: false,
			})
		}
		if err == io.EOF {
			logrus.Tracef("Reached end of file %s", path)
			break
		}
	}
	return nil
}

// RelativeFiles returns a list of all files at the filepath relative to root
func RelativeFiles(fp string, root string) ([]string, error) {
	var files []string
	fullPath := filepath.Join(root, fp)
	cleanedRoot := filepath.Clean(root)
	logrus.Debugf("Getting files and contents at root %s for %s", root, fullPath)
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if CheckCleanedPathAgainstIgnoreList(path) && !hasCleanedFilepathPrefix(filepath.Clean(path), cleanedRoot, false) {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	return files, err
}

// ParentDirectories returns a list of paths to all parent directories
// Ex. /some/temp/dir -> [/, /some, /some/temp, /some/temp/dir]
func ParentDirectories(path string) []string {
	dir := filepath.Clean(path)
	var paths []string
	for {
		if dir == filepath.Clean(config.RootDir) || dir == "" || dir == "." {
			break
		}
		dir, _ = filepath.Split(dir)
		dir = filepath.Clean(dir)
		paths = append([]string{dir}, paths...)
	}
	if len(paths) == 0 {
		paths = []string{config.RootDir}
	}
	return paths
}

// ParentDirectoriesWithoutLeadingSlash returns a list of paths to all parent directories
// all subdirectories do not contain a leading /
// Ex. /some/temp/dir -> [/, some, some/temp, some/temp/dir]
func ParentDirectoriesWithoutLeadingSlash(path string) []string {
	path = filepath.Clean(path)
	dirs := strings.Split(path, "/")
	dirPath := ""
	paths := []string{config.RootDir}
	for index, dir := range dirs {
		if dir == "" || index == (len(dirs)-1) {
			continue
		}
		dirPath = filepath.Join(dirPath, dir)
		paths = append(paths, dirPath)
	}
	return paths
}

// FilepathExists returns true if the path exists
func FilepathExists(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

// resetFileOwnershipIfNotMatching function changes ownership of the file at path to newUID and newGID.
// If the ownership already matches, chown is not executed.
func resetFileOwnershipIfNotMatching(path string, newUID, newGID uint32) error {
	fsInfo, err := os.Lstat(path)
	if err != nil {
		return errors.Wrap(err, "getting stat of present file")
	}
	stat, ok := fsInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("can't convert fs.FileInfo of %v to linux syscall.Stat_t", path)
	}
	if stat.Uid != newUID && stat.Gid != newGID {
		err = os.Chown(path, int(newUID), int(newGID))
		if err != nil {
			return errors.Wrap(err, "reseting file ownership to root")
		}
	}
	return nil
}

// CreateFile creates a file at path and copies over contents from the reader
func CreateFile(path string, reader io.Reader, perm os.FileMode, uid uint32, gid uint32) error {
	// Create directory path if it doesn't exist
	if err := createParentDirectory(path, int(uid), int(gid)); err != nil {
		return errors.Wrap(err, "creating parent dir")
	}

	// if the file is already created with ownership other than root, reset the ownership
	if FilepathExists(path) {
		logrus.Debugf("file at %v already exists, resetting file ownership to root", path)
		err := resetFileOwnershipIfNotMatching(path, 0, 0)
		if err != nil {
			return errors.Wrap(err, "reseting file ownership")
		}
	}

	dest, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "creating file")
	}
	defer dest.Close()
	if _, err := io.Copy(dest, reader); err != nil {
		return errors.Wrap(err, "copying file")
	}
	return setFilePermissions(path, perm, int(uid), int(gid))
}

// AddVolumePath adds the given path to the volume ignorelist.
func AddVolumePathToIgnoreList(path string) {
	logrus.Infof("Adding volume %s to ignorelist", path)
	AddToIgnoreList(IgnoreListEntry{
		Path:            path,
		PrefixMatchOnly: true,
	})
	volumes = append(volumes, path)
}

// DownloadFileToDest downloads the file at rawurl to the given dest for the ADD command
// From add command docs:
//  1. If <src> is a remote file URL:
//     - destination will have permissions of 0600 by default if not specified with chmod
//     - If remote file has HTTP Last-Modified header, we set the mtime of the file to that timestamp
func DownloadFileToDest(rawurl, dest string, uid, gid int64, chmod fs.FileMode) error {
	resp, err := http.Get(rawurl) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("invalid response status %d", resp.StatusCode)
	}

	if err := CreateFile(dest, resp.Body, chmod, uint32(uid), uint32(gid)); err != nil {
		return err
	}
	mTime := time.Time{}
	lastMod := resp.Header.Get("Last-Modified")
	if lastMod != "" {
		if parsedMTime, err := http.ParseTime(lastMod); err == nil {
			mTime = parsedMTime
		}
	}
	return os.Chtimes(dest, mTime, mTime)
}

// DetermineTargetFileOwnership returns the user provided uid/gid combination.
// If they are set to -1, the uid/gid from the original file is used.
func DetermineTargetFileOwnership(fi os.FileInfo, uid, gid int64) (int64, int64) {
	if uid <= DoNotChangeUID {
		uid = int64(fi.Sys().(*syscall.Stat_t).Uid)
	}
	if gid <= DoNotChangeGID {
		gid = int64(fi.Sys().(*syscall.Stat_t).Gid)
	}
	return uid, gid
}

// CopyDir copies the file or directory at src to dest
// It returns a list of files it copied over
func CopyDir(src, dest string, context FileContext, uid, gid int64, chmod fs.FileMode, useDefaultChmod bool) ([]string, error) {
	files, err := RelativeFiles("", src)
	if err != nil {
		return nil, errors.Wrap(err, "copying dir")
	}
	var copiedFiles []string
	for _, file := range files {
		fullPath := filepath.Join(src, file)
		if context.ExcludesFile(fullPath) {
			logrus.Debugf("%s found in .dockerignore, ignoring", src)
			continue
		}
		fi, err := os.Lstat(fullPath)
		if err != nil {
			return nil, errors.Wrap(err, "copying dir")
		}
		destPath := filepath.Join(dest, file)
		if fi.IsDir() {
			logrus.Tracef("Creating directory %s", destPath)

			mode := chmod
			if useDefaultChmod {
				mode = fi.Mode()
			}
			uid, gid := DetermineTargetFileOwnership(fi, uid, gid)
			if err := MkdirAllWithPermissions(destPath, mode, uid, gid); err != nil {
				return nil, err
			}
		} else if IsSymlink(fi) {
			// If file is a symlink, we want to create the same relative symlink
			if _, err := CopySymlink(fullPath, destPath, context); err != nil {
				return nil, err
			}
		} else {
			// ... Else, we want to copy over a file
			mode := chmod
			if useDefaultChmod {
				mode = fs.FileMode(0o600)
			}

			if _, err := CopyFile(fullPath, destPath, context, uid, gid, mode, useDefaultChmod); err != nil {
				return nil, err
			}
		}
		copiedFiles = append(copiedFiles, destPath)
	}
	return copiedFiles, nil
}

// CopySymlink copies the symlink at src to dest.
func CopySymlink(src, dest string, context FileContext) (bool, error) {
	if context.ExcludesFile(src) {
		logrus.Debugf("%s found in .dockerignore, ignoring", src)
		return true, nil
	}
	if FilepathExists(dest) {
		if err := os.RemoveAll(dest); err != nil {
			return false, err
		}
	}
	if err := createParentDirectory(dest, DoNotChangeUID, DoNotChangeGID); err != nil {
		return false, err
	}
	link, err := os.Readlink(src)
	if err != nil {
		logrus.Debugf("Could not read link for %s", src)
	}
	return false, os.Symlink(link, dest)
}

// CopyFile copies the file at src to dest
func CopyFile(src, dest string, context FileContext, uid, gid int64, chmod fs.FileMode, useDefaultChmod bool) (bool, error) {
	if context.ExcludesFile(src) {
		logrus.Debugf("%s found in .dockerignore, ignoring", src)
		return true, nil
	}
	if src == dest {
		// This is a no-op. Move on, but don't list it as ignored.
		// We have to make sure we do this so we don't overwrite our own file.
		// See iusse #904 for an example.
		return false, nil
	}
	fi, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	logrus.Debugf("Copying file %s to %s", src, dest)
	srcFile, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()
	uid, gid = DetermineTargetFileOwnership(fi, uid, gid)

	mode := chmod
	if useDefaultChmod {
		mode = fi.Mode()
	}
	return false, CreateFile(dest, srcFile, mode, uint32(uid), uint32(gid))
}

func NewFileContextFromDockerfile(dockerfilePath, buildcontext string) (FileContext, error) {
	fileContext := FileContext{Root: buildcontext}
	excludedFiles, err := getExcludedFiles(dockerfilePath, buildcontext)
	if err != nil {
		return fileContext, err
	}
	fileContext.ExcludedFiles = excludedFiles
	return fileContext, nil
}

// getExcludedFiles returns a list of files to exclude from the .dockerignore
func getExcludedFiles(dockerfilePath, buildcontext string) ([]string, error) {
	path := dockerfilePath + ".dockerignore"
	if !FilepathExists(path) {
		path = filepath.Join(buildcontext, ".dockerignore")
	}
	if !FilepathExists(path) {
		return nil, nil
	}
	logrus.Infof("Using dockerignore file: %v", path)
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "parsing .dockerignore")
	}
	reader := bytes.NewBuffer(contents)
	return dockerignore.ReadAll(reader)
}

// ExcludesFile returns true if the file context specified this file should be ignored.
// Usually this is specified via .dockerignore
func (c FileContext) ExcludesFile(path string) bool {
	if HasFilepathPrefix(path, c.Root, false) {
		var err error
		path, err = filepath.Rel(c.Root, path)
		if err != nil {
			logrus.Errorf("Unable to get relative path, including %s in build: %v", path, err)
			return false
		}
	}
	match, err := patternmatcher.Matches(path, c.ExcludedFiles)
	if err != nil {
		logrus.Errorf("Error matching, including %s in build: %v", path, err)
		return false
	}
	return match
}

// HasFilepathPrefix checks if the given file path begins with prefix
func HasFilepathPrefix(path, prefix string, prefixMatchOnly bool) bool {
	return hasCleanedFilepathPrefix(filepath.Clean(path), filepath.Clean(prefix), prefixMatchOnly)
}

func hasCleanedFilepathPrefix(path, prefix string, prefixMatchOnly bool) bool {
	prefixArray := strings.Split(prefix, "/")
	pathArray := strings.SplitN(path, "/", len(prefixArray)+1)
	if len(pathArray) < len(prefixArray) {
		return false
	}
	if prefixMatchOnly && len(pathArray) == len(prefixArray) {
		return false
	}

	for index := range prefixArray {
		m, err := filepath.Match(prefixArray[index], pathArray[index])
		if err != nil {
			return false
		}
		if !m {
			return false
		}
	}
	return true
}

func Volumes() []string {
	return volumes
}

func MkdirAllWithPermissions(path string, mode os.FileMode, uid, gid int64) error {
	// Check if a file already exists on the path, if yes then delete it
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		logrus.Tracef("Removing file because it needs to be a directory %s", path)
		if err := os.Remove(path); err != nil {
			return errors.Wrapf(err, "error removing %s to make way for new directory.", path)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "error calling stat on %s.", path)
	}

	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	if uid > math.MaxUint32 || gid > math.MaxUint32 {
		// due to https://github.com/golang/go/issues/8537
		return errors.New(
			fmt.Sprintf(
				"Numeric User-ID or Group-ID greater than %v are not properly supported.",
				uint64(math.MaxUint32),
			),
		)
	}
	if err := os.Chown(path, int(uid), int(gid)); err != nil {
		return err
	}
	// In some cases, MkdirAll doesn't change the permissions, so run Chmod
	// Must chmod after chown because chown resets the file mode.
	return os.Chmod(path, mode)
}

func setFilePermissions(path string, mode os.FileMode, uid, gid int) error {
	if err := os.Chown(path, uid, gid); err != nil {
		return err
	}
	// manually set permissions on file, since the default umask (022) will interfere
	// Must chmod after chown because chown resets the file mode.
	return os.Chmod(path, mode)
}

func setFileTimes(path string, aTime, mTime time.Time) error {
	// The zero value of time.Time is not a valid argument to os.Chtimes as it cannot be
	// converted into a valid argument to the syscall that os.Chtimes uses. If mTime or
	// aTime are zero we convert them to the zero value for Unix Epoch.
	if mTime.IsZero() {
		logrus.Tracef("Mod time for %s is zero, converting to zero for epoch", path)
		mTime = time.Unix(0, 0)
	}

	if aTime.IsZero() {
		logrus.Tracef("Access time for %s is zero, converting to zero for epoch", path)
		aTime = time.Unix(0, 0)
	}

	// We set AccessTime because its a required arg but we only care about
	// ModTime. The file will get accessed again so AccessTime will change.
	if err := os.Chtimes(path, aTime, mTime); err != nil {
		return errors.Wrapf(
			err,
			"couldn't modify times: atime %v mtime %v",
			aTime,
			mTime,
		)
	}

	return nil
}

// CreateTargetTarfile creates target tar file for downloading the context file.
// Make directory if directory does not exist
func CreateTargetTarfile(tarpath string) (*os.File, error) {
	baseDir := filepath.Dir(tarpath)
	if _, err := os.Lstat(baseDir); os.IsNotExist(err) {
		logrus.Debugf("BaseDir %s for file %s does not exist. Creating.", baseDir, tarpath)
		if err := os.MkdirAll(baseDir, 0o755); err != nil {
			return nil, err
		}
	}
	return os.Create(tarpath)
}

// Returns true if a file is a symlink
func IsSymlink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}

var ErrNotSymLink = fmt.Errorf("not a symlink")

func GetSymLink(path string) (string, error) {
	if err := getSymlink(path); err != nil {
		return "", err
	}
	return os.Readlink(path)
}

func EvalSymLink(path string) (string, error) {
	if err := getSymlink(path); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}

func getSymlink(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !IsSymlink(fi) {
		return ErrNotSymLink
	}
	return nil
}

// For cross stage dependencies kaniko must persist the referenced path so that it can be used in
// the dependent stage. For symlinks we copy the target path because copying the symlink would
// result in a dead link
func CopyFileOrSymlink(src string, destDir string, root string) error {
	destFile := filepath.Join(destDir, src)
	src = filepath.Join(root, src)
	fi, err := os.Lstat(src)
	if err != nil {
		return errors.Wrap(err, "getting file info")
	}
	if IsSymlink(fi) {
		link, err := os.Readlink(src)
		if err != nil {
			return errors.Wrap(err, "copying file or symlink")
		}
		if err := createParentDirectory(destFile, DoNotChangeUID, DoNotChangeGID); err != nil {
			return err
		}
		return os.Symlink(link, destFile)
	}
	if err := otiai10Cpy.Copy(src, destFile, skipKanikoDir); err != nil {
		return errors.Wrap(err, "copying file")
	}
	if err := CopyOwnership(src, destDir, root); err != nil {
		return errors.Wrap(err, "copying ownership")
	}
	if err := os.Chmod(destFile, fi.Mode()); err != nil {
		return errors.Wrap(err, "copying file mode")
	}
	return nil
}

// CopyOwnership copies the file or directory ownership recursively at src to dest
func CopyOwnership(src string, destDir string, root string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if IsSymlink(info) {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, relPath)

		if CheckCleanedPathAgainstIgnoreList(src) && CheckCleanedPathAgainstIgnoreList(destPath) {
			if !isExist(destPath) {
				logrus.Debugf("Path %s ignored, but not exists", destPath)
				return nil
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			logrus.Debugf("Not copying ownership for %s, as it's ignored", destPath)
			return nil
		}
		if CheckIgnoreList(destDir) && CheckCleanedPathAgainstIgnoreList(path) {
			if !isExist(path) {
				logrus.Debugf("Path %s ignored, but not exists", path)
				return nil
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			logrus.Debugf("Not copying ownership for %s, as it's ignored", path)
			return nil
		}

		info, err = os.Stat(path)
		if err != nil {
			return errors.Wrap(err, "reading ownership")
		}
		stat := info.Sys().(*syscall.Stat_t)
		return os.Chown(destPath, int(stat.Uid), int(stat.Gid))
	})
}

func createParentDirectory(path string, uid int, gid int) error {
	baseDir := filepath.Dir(path)
	if info, err := os.Lstat(baseDir); os.IsNotExist(err) {
		logrus.Tracef("BaseDir %s for file %s does not exist. Creating.", baseDir, path)

		dir := baseDir
		dirs := []string{baseDir}
		for {
			if dir == "/" || dir == "." || dir == "" {
				break
			}
			dir = filepath.Dir(dir)
			dirs = append(dirs, dir)
		}

		for i := len(dirs) - 1; i >= 0; i-- {
			dir := dirs[i]

			if _, err := os.Lstat(dir); os.IsNotExist(err) {
				os.Mkdir(dir, 0o755)
				if uid != DoNotChangeUID {
					if gid != DoNotChangeGID {
						os.Chown(dir, uid, gid)
					} else {
						return errors.New(fmt.Sprintf("UID=%d but GID=-1, i.e. it is not set for %s", uid, dir))
					}
				} else {
					if gid != DoNotChangeGID {
						return errors.New(fmt.Sprintf("GID=%d but UID=-1, i.e. it is not set for %s", gid, dir))
					}
				}
			} else if err != nil {
				return err
			}
		}
	} else if IsSymlink(info) {
		logrus.Infof("Destination cannot be a symlink %v", baseDir)
		return errors.New("destination cannot be a symlink")
	}
	return nil
}

// InitIgnoreList will initialize the ignore list using:
// - defaultIgnoreList
// - mounted paths via DetectFilesystemIgnoreList()
func InitIgnoreList() error {
	logrus.Trace("Initializing ignore list")
	ignorelist = append([]IgnoreListEntry{}, defaultIgnoreList...)

	if err := DetectFilesystemIgnoreList(config.MountInfoPath); err != nil {
		return errors.Wrap(err, "checking filesystem mount paths for ignore list")
	}

	return nil
}

type walkFSResult struct {
	filesAdded    []string
	existingPaths map[string]struct{}
}

// WalkFS given a directory dir and list of existing files existingPaths,
// returns a list of changed files determined by `changeFunc` and a list
// of deleted files. Input existingPaths is changed inside this function and
// returned as deleted files map.
// It timesout after 90 mins which can be configured via setting an environment variable
// SNAPSHOT_TIMEOUT in the kaniko pod definition.
func WalkFS(
	dir string,
	existingPaths map[string]struct{},
	changeFunc func(string) (bool, error),
) ([]string, map[string]struct{}) {
	timeOutStr := os.Getenv(snapshotTimeout)
	if timeOutStr == "" {
		logrus.Tracef("Environment '%s' not set. Using default snapshot timeout '%s'", snapshotTimeout, defaultTimeout)
		timeOutStr = defaultTimeout
	}
	timeOut, err := time.ParseDuration(timeOutStr)
	if err != nil {
		logrus.Fatalf("Could not parse duration '%s'", timeOutStr)
	}
	timer := timing.Start("Walking filesystem with timeout")

	ch := make(chan walkFSResult, 1)

	go func() {
		ch <- gowalkDir(dir, existingPaths, changeFunc)
	}()

	// Listen on our channel AND a timeout channel - which ever happens first.
	select {
	case res := <-ch:
		timing.DefaultRun.Stop(timer)
		return res.filesAdded, res.existingPaths
	case <-time.After(timeOut):
		timing.DefaultRun.Stop(timer)
		logrus.Fatalf("Timed out snapshotting FS in %s", timeOutStr)
		return nil, nil
	}
}

func gowalkDir(dir string, existingPaths map[string]struct{}, changeFunc func(string) (bool, error)) walkFSResult {
	foundPaths := make([]string, 0)
	deletedFiles := existingPaths // Make a reference.

	callback := func(path string, ent *godirwalk.Dirent) error {
		logrus.Tracef("Analyzing path '%s'", path)

		if IsInIgnoreList(path) {
			if IsDestDir(path) {
				logrus.Tracef("Skipping paths under '%s', as it is an ignored directory", path)
				return filepath.SkipDir
			}
			return nil
		}

		// File is existing on disk, remove it from deleted files.
		delete(deletedFiles, path)

		if isChanged, err := changeFunc(path); err != nil {
			return err
		} else if isChanged {
			foundPaths = append(foundPaths, path)
		}

		return nil
	}

	godirwalk.Walk(dir,
		&godirwalk.Options{
			Callback: callback,
			Unsorted: true,
		})

	return walkFSResult{foundPaths, deletedFiles}
}

// GetFSInfoMap given a directory gets a map of FileInfo for all files
func GetFSInfoMap(dir string, existing map[string]os.FileInfo) (map[string]os.FileInfo, []string) {
	fileMap := map[string]os.FileInfo{}
	foundPaths := []string{}
	timer := timing.Start("Walking filesystem with Stat")
	godirwalk.Walk(dir, &godirwalk.Options{
		Callback: func(path string, ent *godirwalk.Dirent) error {
			if CheckCleanedPathAgainstIgnoreList(path) {
				if IsDestDir(path) {
					logrus.Tracef("Skipping paths under %s, as it is a ignored directory", path)
					return filepath.SkipDir
				}
				return nil
			}
			if fi, err := os.Lstat(path); err == nil {
				if fiPrevious, ok := existing[path]; ok {
					// check if file changed
					if !isSame(fiPrevious, fi) {
						fileMap[path] = fi
						foundPaths = append(foundPaths, path)
					}
				} else {
					// new path
					fileMap[path] = fi
					foundPaths = append(foundPaths, path)
				}
			}
			return nil
		},
		Unsorted: true,
	},
	)
	timing.DefaultRun.Stop(timer)
	return fileMap, foundPaths
}

func isSame(fi1, fi2 os.FileInfo) bool {
	return fi1.Mode() == fi2.Mode() &&
		// file modification time
		fi1.ModTime() == fi2.ModTime() &&
		// file size
		fi1.Size() == fi2.Size() &&
		// file user id
		uint64(fi1.Sys().(*syscall.Stat_t).Uid) == uint64(fi2.Sys().(*syscall.Stat_t).Uid) &&
		// file group id is
		uint64(fi1.Sys().(*syscall.Stat_t).Gid) == uint64(fi2.Sys().(*syscall.Stat_t).Gid)
}
