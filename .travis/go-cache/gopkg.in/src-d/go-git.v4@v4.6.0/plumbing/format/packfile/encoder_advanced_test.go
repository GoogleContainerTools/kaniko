package packfile_test

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/idxfile"
	. "gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

type EncoderAdvancedSuite struct {
	fixtures.Suite
}

var _ = Suite(&EncoderAdvancedSuite{})

func (s *EncoderAdvancedSuite) TestEncodeDecode(c *C) {
	if testing.Short() {
		c.Skip("skipping test in short mode.")
	}

	fixs := fixtures.Basic().ByTag("packfile").ByTag(".git")
	fixs = append(fixs, fixtures.ByURL("https://github.com/src-d/go-git.git").
		ByTag("packfile").ByTag(".git").One())
	fixs.Test(c, func(f *fixtures.Fixture) {
		storage, err := filesystem.NewStorage(f.DotGit())
		c.Assert(err, IsNil)
		s.testEncodeDecode(c, storage, 10)
	})
}

func (s *EncoderAdvancedSuite) TestEncodeDecodeNoDeltaCompression(c *C) {
	if testing.Short() {
		c.Skip("skipping test in short mode.")
	}

	fixs := fixtures.Basic().ByTag("packfile").ByTag(".git")
	fixs = append(fixs, fixtures.ByURL("https://github.com/src-d/go-git.git").
		ByTag("packfile").ByTag(".git").One())
	fixs.Test(c, func(f *fixtures.Fixture) {
		storage, err := filesystem.NewStorage(f.DotGit())
		c.Assert(err, IsNil)
		s.testEncodeDecode(c, storage, 0)
	})
}

func (s *EncoderAdvancedSuite) testEncodeDecode(
	c *C,
	storage storer.Storer,
	packWindow uint,
) {
	objIter, err := storage.IterEncodedObjects(plumbing.AnyObject)
	c.Assert(err, IsNil)

	expectedObjects := map[plumbing.Hash]bool{}
	var hashes []plumbing.Hash
	err = objIter.ForEach(func(o plumbing.EncodedObject) error {
		expectedObjects[o.Hash()] = true
		hashes = append(hashes, o.Hash())
		return err

	})
	c.Assert(err, IsNil)

	// Shuffle hashes to avoid delta selector getting order right just because
	// the initial order is correct.
	auxHashes := make([]plumbing.Hash, len(hashes))
	for i, j := range rand.Perm(len(hashes)) {
		auxHashes[j] = hashes[i]
	}
	hashes = auxHashes

	buf := bytes.NewBuffer(nil)
	enc := NewEncoder(buf, storage, false)
	encodeHash, err := enc.Encode(hashes, packWindow)
	c.Assert(err, IsNil)

	fs := memfs.New()
	f, err := fs.Create("packfile")
	c.Assert(err, IsNil)

	_, err = f.Write(buf.Bytes())
	c.Assert(err, IsNil)

	_, err = f.Seek(0, io.SeekStart)
	c.Assert(err, IsNil)

	w := new(idxfile.Writer)
	parser, err := NewParser(NewScanner(f), w)
	c.Assert(err, IsNil)

	_, err = parser.Parse()
	c.Assert(err, IsNil)
	index, err := w.Index()
	c.Assert(err, IsNil)

	_, err = f.Seek(0, io.SeekStart)
	c.Assert(err, IsNil)

	p := NewPackfile(index, fs, f)

	decodeHash, err := p.ID()
	c.Assert(err, IsNil)
	c.Assert(encodeHash, Equals, decodeHash)

	objIter, err = p.GetAll()
	c.Assert(err, IsNil)
	obtainedObjects := map[plumbing.Hash]bool{}
	err = objIter.ForEach(func(o plumbing.EncodedObject) error {
		obtainedObjects[o.Hash()] = true
		return nil
	})
	c.Assert(err, IsNil)
	c.Assert(obtainedObjects, DeepEquals, expectedObjects)

	for h := range obtainedObjects {
		if !expectedObjects[h] {
			c.Errorf("obtained unexpected object: %s", h)
		}
	}

	for h := range expectedObjects {
		if !obtainedObjects[h] {
			c.Errorf("missing object: %s", h)
		}
	}
}
