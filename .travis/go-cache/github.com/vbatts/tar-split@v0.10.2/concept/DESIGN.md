# Flow of TAR stream

## `./archive/tar`

The import path `github.com/vbatts/tar-split/archive/tar` is fork of upstream golang stdlib [`archive/tar`](http://golang.org/pkg/archive/tar/).
It adds plumbing to access raw bytes of the tar stream as the headers and payload are read.

## Packer interface

For ease of storage and usage of the raw bytes, there will be a storage
interface, that accepts an io.Writer (This way you could pass it an in memory
buffer or a file handle).

Having a Packer interface can allow configuration of hash.Hash for file payloads
and providing your own io.Writer.

Instead of having a state directory to store all the header information for all
Readers, we will leave that up to user of Reader. Because we can not assume an
ID for each Reader, and keeping that information differentiated.

## State Directory

Perhaps we could deduplicate the header info, by hashing the rawbytes and
storing them in a directory tree like:

	./ac/dc/beef

Then reference the hash of the header info, in the positional records for the
tar stream. Though this could be a future feature, and not required for an
initial implementation. Also, this would imply an owned state directory, rather
than just writing storage info to an io.Writer.

## Concept Example

First we'll get an archive to work with. For repeatability, we'll make an
archive from what you've just cloned:

```
git archive --format=tar -o tar-split.tar HEAD .
```

Then build the example main.go:

```
go build ./main.go
```

Now run the example over the archive:

```
$ ./main tar-split.tar
2015/02/20 15:00:58 writing "tar-split.tar" to "tar-split.tar.out"
pax_global_header pre: 512 read: 52
.travis.yml pre: 972 read: 374
DESIGN.md pre: 650 read: 1131
LICENSE pre: 917 read: 1075
README.md pre: 973 read: 4289
archive/ pre: 831 read: 0
archive/tar/ pre: 512 read: 0
archive/tar/common.go pre: 512 read: 7790
[...]
tar/storage/entry_test.go pre: 667 read: 1137
tar/storage/getter.go pre: 911 read: 2741
tar/storage/getter_test.go pre: 843 read: 1491
tar/storage/packer.go pre: 557 read: 3141
tar/storage/packer_test.go pre: 955 read: 3096
EOF padding: 1512
Remainder: 512
Size: 215040; Sum: 215040
```

*What are we seeing here?* 

* `pre` is the header of a file entry, and potentially the padding from the
  end of the prior file's payload. Also with particular tar extensions and pax
  attributes, the header can exceed 512 bytes.
* `read` is the size of the file payload from the entry
* `EOF padding` is the expected 1024 null bytes on the end of a tar archive,
  plus potential padding from the end of the prior file entry's payload
* `Remainder` is the remaining bytes of an archive. This is typically deadspace
  as most tar implmentations will return after having reached the end of the
  1024 null bytes. Though various implementations will include some amount of
  bytes here, which will affect the checksum of the resulting tar archive,
  therefore this must be accounted for as well.

Ideally the input tar and output `*.out`, will match:

```
$ sha1sum tar-split.tar*
ca9e19966b892d9ad5960414abac01ef585a1e22  tar-split.tar
ca9e19966b892d9ad5960414abac01ef585a1e22  tar-split.tar.out
```


