package godirwalk

import (
	"os"
	"path/filepath"
	"testing"
)

const testScratchBufferSize = 16 * 1024

func helperFilepathWalk(tb testing.TB, osDirname string) []string {
	var entries []string
	err := filepath.Walk(osDirname, func(osPathname string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "skip" {
			return filepath.SkipDir
		}
		// filepath.Walk invokes callback function with a slashed version of the
		// pathname, while godirwalk invokes callback function with the
		// os-specific pathname separator.
		entries = append(entries, filepath.ToSlash(osPathname))
		return nil
	})
	if err != nil {
		tb.Fatal(err)
	}
	return entries
}

func helperGodirwalkWalk(tb testing.TB, osDirname string) []string {
	var entries []string
	err := Walk(osDirname, &Options{
		Callback: func(osPathname string, dirent *Dirent) error {
			if dirent.Name() == "skip" {
				return filepath.SkipDir
			}
			// filepath.Walk invokes callback function with a slashed version of
			// the pathname, while godirwalk invokes callback function with the
			// os-specific pathname separator.
			entries = append(entries, filepath.ToSlash(osPathname))
			return nil
		},
		ScratchBuffer: make([]byte, testScratchBufferSize),
	})
	if err != nil {
		tb.Fatal(err)
	}
	return entries
}

func symlinkAbs(oldname, newname string) error {
	absDir, err := filepath.Abs(oldname)
	if err != nil {
		return err
	}
	return os.Symlink(absDir, newname)
}

func TestWalkSkipDir(t *testing.T) {
	testDataRoot := setup(t)
	defer teardown(t, testDataRoot)

	// Ensure the results from calling filepath.Walk exactly match the results
	// for calling this library's walk function.

	test := func(t *testing.T, osDirname string) {
		osDirname = filepath.Join(testDataRoot, osDirname)
		expected := helperFilepathWalk(t, osDirname)
		actual := helperGodirwalkWalk(t, osDirname)

		if got, want := len(actual), len(expected); got != want {
			t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
		}

		for i := 0; i < len(actual); i++ {
			if got, want := actual[i], expected[i]; got != want {
				t.Errorf("(GOT) %v; (WNT) %v", got, want)
			}
		}
	}

	// Test cases for encountering the filepath.SkipDir error at different times
	// from the call.

	t.Run("SkipFileAtRoot", func(t *testing.T) {
		test(t, "dir1/dir1a")
	})

	t.Run("SkipFileUnderRoot", func(t *testing.T) {
		test(t, "dir1")
	})

	t.Run("SkipDirAtRoot", func(t *testing.T) {
		test(t, "dir2/skip")
	})

	t.Run("SkipDirUnderRoot", func(t *testing.T) {
		test(t, "dir2")
	})

	t.Run("SkipDirOnSymlink", func(t *testing.T) {
		osDirname := filepath.Join(testDataRoot, "dir3")
		actual := helperGodirwalkWalk(t, osDirname)

		expected := []string{
			filepath.Join(testDataRoot, "dir3"),
			filepath.Join(testDataRoot, "dir3/aaa.txt"),
			filepath.Join(testDataRoot, "dir3/zzz"),
			filepath.Join(testDataRoot, "dir3/zzz/aaa.txt"),
		}

		if got, want := len(actual), len(expected); got != want {
			t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
		}

		for i := 0; i < len(actual); i++ {
			if got, want := actual[i], expected[i]; got != want {
				t.Errorf("(GOT) %v; (WNT) %v", got, want)
			}
		}
	})
}

func TestWalkNoAccess(t *testing.T) {
	testDataRoot := setup(t)
	defer teardown(t, testDataRoot)

	var actual []string

	err := Walk(testDataRoot, &Options{
		ScratchBuffer: make([]byte, testScratchBufferSize),
		Callback: func(osPathname string, _ *Dirent) error {
			t.Logf("walk in: %s", osPathname)
			return nil
		},
		ErrorCallback: func(osChildname string, err error) ErrorAction {
			actual = append(actual, osChildname)
			return SkipNode
		},
	})
	if err != nil {
		t.Errorf("(GOT): %v; (WNT): %v", err, nil)
	}

	expected := []string{
		filepath.Join(testDataRoot, "dir6/noaccess"),
	}

	if got, want := len(actual), len(expected); got != want {
		t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
	}

	for i := 0; i < len(actual); i++ {
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

func TestWalkFollowSymbolicLinksFalse(t *testing.T) {
	testDataRoot := setup(t)
	defer teardown(t, testDataRoot)

	osDirname := filepath.Join(testDataRoot, "dir4")
	symlink := filepath.Join(testDataRoot, "dir4/symlinkToAbsDirectory")

	if err := symlinkAbs(filepath.Join(testDataRoot, "dir4/zzz"), symlink); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Remove(symlink); err != nil {
			t.Error(err)
		}
	}()

	var actual []string
	err := Walk(osDirname, &Options{
		Callback: func(osPathname string, dirent *Dirent) error {
			if dirent.Name() == "skip" {
				return filepath.SkipDir
			}
			// filepath.Walk invokes callback function with a slashed version of
			// the pathname, while godirwalk invokes callback function with the
			// os-specific pathname separator.
			actual = append(actual, filepath.ToSlash(osPathname))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		filepath.Join(testDataRoot, "dir4"),
		filepath.Join(testDataRoot, "dir4/aaa.txt"),
		filepath.Join(testDataRoot, "dir4/symlinkToAbsDirectory"),
		filepath.Join(testDataRoot, "dir4/symlinkToDirectory"),
		filepath.Join(testDataRoot, "dir4/symlinkToFile"),
		filepath.Join(testDataRoot, "dir4/zzz"),
		filepath.Join(testDataRoot, "dir4/zzz/aaa.txt"),
	}

	if got, want := len(actual), len(expected); got != want {
		t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
	}

	for i := 0; i < len(actual); i++ {
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

func TestWalkFollowSymbolicLinksTrue(t *testing.T) {
	testDataRoot := setup(t)
	defer teardown(t, testDataRoot)

	osDirname := filepath.Join(testDataRoot, "dir4")
	symlink := filepath.Join(testDataRoot, "dir4/symlinkToAbsDirectory")

	if err := symlinkAbs(filepath.Join(testDataRoot, "dir4/zzz"), symlink); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Remove(symlink); err != nil {
			t.Error(err)
		}
	}()

	var actual []string
	err := Walk(osDirname, &Options{
		FollowSymbolicLinks: true,
		Callback: func(osPathname string, dirent *Dirent) error {
			if dirent.Name() == "skip" {
				return filepath.SkipDir
			}
			// filepath.Walk invokes callback function with a slashed version of
			// the pathname, while godirwalk invokes callback function with the
			// os-specific pathname separator.
			actual = append(actual, filepath.ToSlash(osPathname))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		filepath.Join(testDataRoot, "dir4"),
		filepath.Join(testDataRoot, "dir4/aaa.txt"),
		filepath.Join(testDataRoot, "dir4/symlinkToAbsDirectory"),
		filepath.Join(testDataRoot, "dir4/symlinkToAbsDirectory/aaa.txt"),
		filepath.Join(testDataRoot, "dir4/symlinkToDirectory"),
		filepath.Join(testDataRoot, "dir4/symlinkToDirectory/aaa.txt"),
		filepath.Join(testDataRoot, "dir4/symlinkToFile"),
		filepath.Join(testDataRoot, "dir4/zzz"),
		filepath.Join(testDataRoot, "dir4/zzz/aaa.txt"),
	}

	if got, want := len(actual), len(expected); got != want {
		t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
	}

	for i := 0; i < len(actual); i++ {
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

func TestPostChildrenCallback(t *testing.T) {
	testDataRoot := setup(t)
	defer teardown(t, testDataRoot)

	osDirname := filepath.Join(testDataRoot, "dir5")

	var actual []string

	err := Walk(osDirname, &Options{
		ScratchBuffer: make([]byte, testScratchBufferSize),
		Callback: func(osPathname string, _ *Dirent) error {
			t.Logf("walk in: %s", osPathname)
			return nil
		},
		PostChildrenCallback: func(osPathname string, de *Dirent) error {
			t.Logf("walk out: %s", osPathname)
			actual = append(actual, osPathname)
			return nil
		},
	})
	if err != nil {
		t.Errorf("(GOT): %v; (WNT): %v", err, nil)
	}

	expected := []string{
		filepath.Join(testDataRoot, "dir5/a2/a2a"),
		filepath.Join(testDataRoot, "dir5/a2"),
		filepath.Join(testDataRoot, "dir5"),
	}

	if got, want := len(actual), len(expected); got != want {
		t.Errorf("(GOT) %v; (WNT) %v", got, want)
	}

	for i := 0; i < len(actual); i++ {
		if i >= len(expected) {
			t.Fatalf("(GOT) %v; (WNT): %v", actual[i], nil)
		}
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

var goPrefix = filepath.Join(os.Getenv("GOPATH"), "src")

func BenchmarkFilepathWalk(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark using user's Go source directory")
	}

	for i := 0; i < b.N; i++ {
		_ = helperFilepathWalk(b, goPrefix)
	}
}

func BenchmarkGoDirWalk(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark using user's Go source directory")
	}

	for i := 0; i < b.N; i++ {
		_ = helperGodirwalkWalk(b, goPrefix)
	}
}

const flameIterations = 10

func BenchmarkFlameGraphFilepathWalk(b *testing.B) {
	for i := 0; i < flameIterations; i++ {
		_ = helperFilepathWalk(b, goPrefix)
	}
}

func BenchmarkFlameGraphGoDirWalk(b *testing.B) {
	for i := 0; i < flameIterations; i++ {
		_ = helperGodirwalkWalk(b, goPrefix)
	}
}
