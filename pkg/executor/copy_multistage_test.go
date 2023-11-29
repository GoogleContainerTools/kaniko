/*
Copyright 2020 Google LLC

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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/testutil"
)

func readDirectory(dirName string) ([]fs.FileInfo, error) {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return nil, err
	}

	testDir := make([]fs.FileInfo, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		testDir = append(testDir, info)
	}
	return testDir, err
}

func TestCopyCommand_Multistage(t *testing.T) {
	t.Run("copy a file across multistage", func(t *testing.T) {
		testDir, fn := setupMultistageTests(t)
		defer fn()
		dockerFile := `
FROM scratch as first
COPY foo/bam.txt copied/
ENV test test

From scratch as second
COPY --from=first copied/bam.txt output/bam.txt`
		os.WriteFile(filepath.Join(testDir, "workspace", "Dockerfile"), []byte(dockerFile), 0755)
		opts := &config.KanikoOptions{
			DockerfilePath: filepath.Join(testDir, "workspace", "Dockerfile"),
			SrcContext:     filepath.Join(testDir, "workspace"),
			SnapshotMode:   constants.SnapshotModeFull,
		}
		_, err := DoBuild(opts)
		testutil.CheckNoError(t, err)
		// Check Image has one layer bam.txt
		files, err := readDirectory(filepath.Join(testDir, "output"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "bam.txt")

	})

	t.Run("copy a file across multistage into a directory", func(t *testing.T) {
		testDir, fn := setupMultistageTests(t)
		defer fn()
		dockerFile := `
FROM scratch as first
COPY foo/bam.txt copied/
ENV test test

From scratch as second
COPY --from=first copied/bam.txt output/`
		os.WriteFile(filepath.Join(testDir, "workspace", "Dockerfile"), []byte(dockerFile), 0755)
		opts := &config.KanikoOptions{
			DockerfilePath: filepath.Join(testDir, "workspace", "Dockerfile"),
			SrcContext:     filepath.Join(testDir, "workspace"),
			SnapshotMode:   constants.SnapshotModeFull,
		}
		_, err := DoBuild(opts)
		testutil.CheckNoError(t, err)
		files, err := readDirectory(filepath.Join(testDir, "output"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 1, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "bam.txt")
	})

	t.Run("copy directory across multistage into a directory", func(t *testing.T) {
		testDir, fn := setupMultistageTests(t)
		defer fn()
		dockerFile := `
FROM scratch as first
COPY foo copied
ENV test test

From scratch as second
COPY --from=first copied another`
		os.WriteFile(filepath.Join(testDir, "workspace", "Dockerfile"), []byte(dockerFile), 0755)
		opts := &config.KanikoOptions{
			DockerfilePath: filepath.Join(testDir, "workspace", "Dockerfile"),
			SrcContext:     filepath.Join(testDir, "workspace"),
			SnapshotMode:   constants.SnapshotModeFull,
		}
		_, err := DoBuild(opts)
		testutil.CheckNoError(t, err)
		// Check Image has one layer bam.txt
		files, err := readDirectory(filepath.Join(testDir, "another"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 2, len(files))
		testutil.CheckDeepEqual(t, files[0].Name(), "bam.link")
		testutil.CheckDeepEqual(t, files[1].Name(), "bam.txt")
		// TODO fix this
		// path := filepath.Join(testDir, "output/another", "bam.link")
		//linkName, err := os.Readlink(path)
		//if err != nil {
		//	t.Fatal(err)
		//}
		//testutil.CheckDeepEqual(t, linkName, "bam.txt")
	})

	t.Run("copy root across multistage", func(t *testing.T) {
		testDir, fn := setupMultistageTests(t)
		defer fn()
		dockerFile := `
FROM scratch as first
COPY foo copied
ENV test test

From scratch as second
COPY --from=first / output/`
		os.WriteFile(filepath.Join(testDir, "workspace", "Dockerfile"), []byte(dockerFile), 0755)
		opts := &config.KanikoOptions{
			DockerfilePath: filepath.Join(testDir, "workspace", "Dockerfile"),
			SrcContext:     filepath.Join(testDir, "workspace"),
			SnapshotMode:   constants.SnapshotModeFull,
		}
		_, err := DoBuild(opts)
		testutil.CheckNoError(t, err)

		filesUnderRoot, err := os.ReadDir(filepath.Join(testDir, "output/"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 3, len(filesUnderRoot))

		files, err := os.ReadDir(filepath.Join(testDir, "output/workspace/foo"))
		if err != nil {
			t.Fatal(err)
		}
		testutil.CheckDeepEqual(t, 2, len(files))
		testutil.CheckDeepEqual(t, "bam.link", files[0].Name())
		testutil.CheckDeepEqual(t, "bam.txt", files[1].Name())
	})

}

func setupMultistageTests(t *testing.T) (string, func()) {
	testDir := t.TempDir()

	// Create workspace with files, dirs, and symlinks
	// workspace tree:
	// /root
	//    /kaniko
	//    /workspace
	//     - /foo
	//          - bam.txt
	//          - bam.link -> bam.txt
	//     - /bin
	//          - exec.link -> ../exec
	//      exec

	// Make directory for stage or else the executor will create with permissions 0664
	// and we will run into issue https://github.com/golang/go/issues/22323
	if err := os.MkdirAll(filepath.Join(testDir, "kaniko/0"), 0755); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(testDir, "workspace")
	// Make foo
	if err := os.MkdirAll(filepath.Join(workspace, "foo"), 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(workspace, "foo", "bam.txt")
	if err := os.WriteFile(file, []byte("meow"), 0755); err != nil {
		t.Fatal(err)
	}
	os.Symlink("bam.txt", filepath.Join(workspace, "foo", "bam.link"))

	// Make a file with contents link
	file = filepath.Join(workspace, "exec")
	if err := os.WriteFile(file, []byte("woof"), 0755); err != nil {
		t.Fatal(err)
	}
	// Make bin
	if err := os.MkdirAll(filepath.Join(workspace, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	os.Symlink("../exec", filepath.Join(workspace, "bin", "exec.link"))

	// set up config
	config.RootDir = testDir
	config.KanikoDir = fmt.Sprintf("%s/%s", testDir, "kaniko")
	// Write path to ignore list
	if err := os.MkdirAll(filepath.Join(testDir, "proc"), 0755); err != nil {
		t.Fatal(err)
	}
	mFile := filepath.Join(testDir, "proc/mountinfo")
	mountInfo := fmt.Sprintf(
		`36 35 98:0 /kaniko %s/kaniko rw,noatime master:1 - ext3 /dev/root rw,errors=continue
36 35 98:0 /proc %s/proc rw,noatime master:1 - ext3 /dev/root rw,errors=continue
`, testDir, testDir)
	if err := os.WriteFile(mFile, []byte(mountInfo), 0644); err != nil {
		t.Fatal(err)
	}
	config.MountInfoPath = mFile
	return testDir, func() {
		config.RootDir = constants.RootDir
		config.MountInfoPath = constants.MountInfoPath
	}
}
