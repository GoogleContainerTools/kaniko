package idxfile_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/idxfile"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

func BenchmarkFindOffset(b *testing.B) {
	idx, err := fixtureIndex()
	if err != nil {
		b.Fatalf(err.Error())
	}

	for i := 0; i < b.N; i++ {
		for _, h := range fixtureHashes {
			_, err := idx.FindOffset(h)
			if err != nil {
				b.Fatalf("error getting offset: %s", err)
			}
		}
	}
}

func BenchmarkFindCRC32(b *testing.B) {
	idx, err := fixtureIndex()
	if err != nil {
		b.Fatalf(err.Error())
	}

	for i := 0; i < b.N; i++ {
		for _, h := range fixtureHashes {
			_, err := idx.FindCRC32(h)
			if err != nil {
				b.Fatalf("error getting crc32: %s", err)
			}
		}
	}
}

func BenchmarkContains(b *testing.B) {
	idx, err := fixtureIndex()
	if err != nil {
		b.Fatalf(err.Error())
	}

	for i := 0; i < b.N; i++ {
		for _, h := range fixtureHashes {
			ok, err := idx.Contains(h)
			if err != nil {
				b.Fatalf("error checking if hash is in index: %s", err)
			}

			if !ok {
				b.Error("expected hash to be in index")
			}
		}
	}
}

func BenchmarkEntries(b *testing.B) {
	idx, err := fixtureIndex()
	if err != nil {
		b.Fatalf(err.Error())
	}

	for i := 0; i < b.N; i++ {
		iter, err := idx.Entries()
		if err != nil {
			b.Fatalf("unexpected error getting entries: %s", err)
		}

		var entries int
		for {
			_, err := iter.Next()
			if err != nil {
				if err == io.EOF {
					break
				}

				b.Errorf("unexpected error getting entry: %s", err)
			}

			entries++
		}

		if entries != len(fixtureHashes) {
			b.Errorf("expecting entries to be %d, got %d", len(fixtureHashes), entries)
		}
	}
}

type IndexSuite struct {
	fixtures.Suite
}

var _ = Suite(&IndexSuite{})

func (s *IndexSuite) TestFindHash(c *C) {
	idx, err := fixtureIndex()
	c.Assert(err, IsNil)

	for i, pos := range fixtureOffsets {
		hash, err := idx.FindHash(pos)
		c.Assert(err, IsNil)
		c.Assert(hash, Equals, fixtureHashes[i])
	}
}

var fixtureHashes = []plumbing.Hash{
	plumbing.NewHash("303953e5aa461c203a324821bc1717f9b4fff895"),
	plumbing.NewHash("5296768e3d9f661387ccbff18c4dea6c997fd78c"),
	plumbing.NewHash("03fc8d58d44267274edef4585eaeeb445879d33f"),
	plumbing.NewHash("8f3ceb4ea4cb9e4a0f751795eb41c9a4f07be772"),
	plumbing.NewHash("e0d1d625010087f79c9e01ad9d8f95e1628dda02"),
	plumbing.NewHash("90eba326cdc4d1d61c5ad25224ccbf08731dd041"),
	plumbing.NewHash("bab53055add7bc35882758a922c54a874d6b1272"),
	plumbing.NewHash("1b8995f51987d8a449ca5ea4356595102dc2fbd4"),
	plumbing.NewHash("35858be9c6f5914cbe6768489c41eb6809a2bceb"),
}

var fixtureOffsets = []int64{
	12,
	142,
	1601322837,
	2646996529,
	3452385606,
	3707047470,
	5323223332,
	5894072943,
	5924278919,
}

func fixtureIndex() (*idxfile.MemoryIndex, error) {
	f := bytes.NewBufferString(fixtureLarge4GB)

	idx := new(idxfile.MemoryIndex)

	d := idxfile.NewDecoder(base64.NewDecoder(base64.StdEncoding, f))
	err := d.Decode(idx)
	if err != nil {
		return nil, fmt.Errorf("unexpected error decoding index: %s", err)
	}

	return idx, nil
}
