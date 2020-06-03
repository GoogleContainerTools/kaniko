# Filesystem Resolution 01

* Author(s): cgwippern@google.com
* Reviewers: Tejal Desai
* Date: 2020-02-12
* Status: Reviewed

## Background

Kaniko builds Docker image layers as overlay filesystem layers; specifically it
creates a tar file which contains the entire content of a given layer in the
overlay filesystem. Each overlay layer corresponds to one image layer.

Overlay filesystems should only contain the objects changed in each layer;
meaning that if only one file changes between some layer A and some B, layer B
would only contain a single file (the one that changed).

To accomplish this, Kaniko walks the entire filesystem to discover every object.
Some of these objects may actually be a symlink to another object in the
filesystem; in these cases we must consider both the link and the target object.

Kaniko also maintains a set of ignored (aka ignored) filepaths. Any object
which matches one of these filepaths should be ignored by kaniko.

This results in a 3 dimensional search space

* changed relative to previous layer
* symlink
* whitelisted

Kaniko must also track which objects are referred to by multiple stages; this
functionality is out of scope for this proposal.

This search space is currently managed in an inconsistent and somewhat ad-hoc
way; code that manages the various search dimensions is spread out and
duplicated. There are also a number of edge cases which continue
to cause bugs.

The search space dimensions cannot be reduced or substituted.

Currently there are a number of bugs around symlinks incorrectly resolved,
whitelists not respected, and unchanged files added to layers.

## Design

During snapshotting, filepaths should be resolved using a consitent API which
takes into account both symlinks and whitelist.

* Callers of this API should not be concerned with the type of object at a given filepath (e.g. symlink or not).
* Callers of this API should not be concerned with whether a given path is whitelisted.
* This API should return a set of filepaths which can be checked for changes
  without further link resolution or whitelist checking.

The API should take a limited set of arguments
* A list of absolute filepaths to scan
* The whitelist

The API should return only two arguments
* A set of filepaths
* error or nil

The signature of the API should look similar to
```
  ResolveFilePaths(inputPaths []string, whitelist []WhitelistEntry) (resolvedPaths []string, err error)
```

The API will iterate over the set of filepaths and for each item
* check whether it is whitelisted; if it is, skip it
* check whether it is a symlink
  * if it is a symlink
    * resolve the link ancestor (nearest ancestor which is a symlink) and the
      target
    * add the link ancestor to the output
    * check whether the target is whitelisted and if
      not add the target to the output

All ancestors of each filepath will also be added to the list, but the previous
checks will not be applied to the ancestors. This maintains the current behavior
which we believe is needed to maintain correct permissions on the ancestor
directories.

### Open Issues/Questions

\<Ignore symlinks targeting whitelisted paths?\>

Given some link `/foo/link/bar` whose target is a whitelisted path such as
`/var/run`, should `/foo/link/bar` be added to the layer?

Resolution: Resolved

Yes, it should be added.

\<Adding ancestor directories\>

According to [this comment](https://github.com/GoogleContainerTools/kaniko/blob/1e9f525509d4e6a066a6e07ab9afbef69b3a3b2c/pkg/snapshot/snapshot.go#L193)
the ancestor directories (parent, grandparent, etc) must also be added to the
layer to preserve the permissions on those directories. This brings into
question whether any filtering needs to happen on these ancestors. IIUC the
current whitelist logic it is possible for `/some/dir` to be whitelisted but not
`/some/dir/containing-a-file.txt`. If filtering needs to be applied to these
ancestors does it make most sense to handle this within the proposed filtering
API?

Resolution: Resolved

Yes, this should be handled in the API

\<Should the API handle diff'ing files?\>

The proposal currently states that the list of files returned from the API
should be immediately added to the layer, but this would imply that diff'ing
existing files, finding newly created files, and handling deleted files would
have already been done. It may be advantageous to handle these outside of the
API in order to reduce scope and complexity. If these are handled outside of the
API how can we decouple and encapsulate these two functions?

Resolution: Resolved

The API will not handle file diffing or whiteouts.

## Implementation plan

* Write the new API
* Write tests for the new API
* Integrate the new API into existing code

## Integration test plan

Add integration tests to the existing suite which cover the known bugs

## Notes

Given some path `/usr/lib/foo` which is a link to `/etc/foo/`

And `/etc/foo` contains `/etc/foo/bar.txt`

Adding a link `/usr/lib/foo/bar.txt` => `/etc/foo/bar.txt` will break the image

In a linux shell this raises an error
```
$ ls /usr/lib/bar
=> /usr/lib/bar/foo.txt
$ ln -s /usr/lib/bar barlink
$ ln -s /usr/lib/bar/foo.txt barlink/foo.txt
=> ERROR
```

Given some path `/usr/foo/bar` which is a link to `/dev/null`, and `/dev` is
whitelisted `/dev/null` should not be added to the image.
