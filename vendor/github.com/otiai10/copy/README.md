# copy

[![Go Reference](https://pkg.go.dev/badge/github.com/otiai10/copy.svg)](https://pkg.go.dev/github.com/otiai10/copy)
[![Actions Status](https://github.com/otiai10/copy/workflows/Go/badge.svg)](https://github.com/otiai10/copy/actions)
[![codecov](https://codecov.io/gh/otiai10/copy/branch/main/graph/badge.svg)](https://codecov.io/gh/otiai10/copy)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/otiai10/copy/blob/main/LICENSE)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fotiai10%2Fcopy.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fotiai10%2Fcopy?ref=badge_shield)
[![CodeQL](https://github.com/otiai10/copy/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/otiai10/copy/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/otiai10/copy)](https://goreportcard.com/report/github.com/otiai10/copy)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/otiai10/copy?sort=semver)](https://pkg.go.dev/github.com/otiai10/copy)
[![Docker Test](https://github.com/otiai10/copy/actions/workflows/docker-test.yml/badge.svg)](https://github.com/otiai10/copy/actions/workflows/docker-test.yml)
[![Vagrant Test](https://github.com/otiai10/copy/actions/workflows/vagrant-test.yml/badge.svg)](https://github.com/otiai10/copy/actions/workflows/vagrant-test.yml)

`copy` copies directories recursively.

# Example Usage

```go
err := Copy("your/directory", "your/directory.copy")
```

# Advanced Usage

```go
// Options specifies optional actions on copying.
type Options struct {

	// OnSymlink can specify what to do on symlink
	OnSymlink func(src string) SymlinkAction

	// OnDirExists can specify what to do when there is a directory already existing in destination.
	OnDirExists func(src, dest string) DirExistsAction

	// Skip can specify which files should be skipped
	Skip func(src string) (bool, error)

	// AddPermission to every entry,
	// NO MORE THAN 0777
	AddPermission os.FileMode

	// Sync file after copy.
	// Useful in case when file must be on the disk
	// (in case crash happens, for example),
	// at the expense of some performance penalty
	Sync bool

	// Preserve the atime and the mtime of the entries
	// On linux we can preserve only up to 1 millisecond accuracy
	PreserveTimes bool

	// Preserve the uid and the gid of all entries.
	PreserveOwner bool

	// The byte size of the buffer to use for copying files.
	// If zero, the internal default buffer of 32KB is used.
	// See https://golang.org/pkg/io/#CopyBuffer for more information.
	CopyBufferSize uint
}
```

```go
// For example...
opt := Options{
	Skip: func(src string) (bool, error) {
		return strings.HasSuffix(src, ".git"), nil
	},
}
err := Copy("your/directory", "your/directory.copy", opt)
```

# Issues

- https://github.com/otiai10/copy/issues


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fotiai10%2Fcopy.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fotiai10%2Fcopy?ref=badge_large)