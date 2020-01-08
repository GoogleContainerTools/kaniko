package object

import (
	"bytes"
	"io"
	"io/ioutil"

	"gopkg.in/src-d/go-git.v4/plumbing"

	. "gopkg.in/check.v1"
)

type BlobsSuite struct {
	BaseObjectsSuite
}

var _ = Suite(&BlobsSuite{})

func (s *BlobsSuite) TestBlobHash(c *C) {
	o := &plumbing.MemoryObject{}
	o.SetType(plumbing.BlobObject)
	o.SetSize(3)

	writer, err := o.Writer()
	c.Assert(err, IsNil)
	defer func() { c.Assert(writer.Close(), IsNil) }()

	writer.Write([]byte{'F', 'O', 'O'})

	blob := &Blob{}
	c.Assert(blob.Decode(o), IsNil)

	c.Assert(blob.Size, Equals, int64(3))
	c.Assert(blob.Hash.String(), Equals, "d96c7efbfec2814ae0301ad054dc8d9fc416c9b5")

	reader, err := blob.Reader()
	c.Assert(err, IsNil)
	defer func() { c.Assert(reader.Close(), IsNil) }()

	data, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, "FOO")
}

func (s *BlobsSuite) TestBlobDecodeEncodeIdempotent(c *C) {
	var objects []*plumbing.MemoryObject
	for _, str := range []string{"foo", "foo\n"} {
		obj := &plumbing.MemoryObject{}
		obj.Write([]byte(str))
		obj.SetType(plumbing.BlobObject)
		obj.Hash()
		objects = append(objects, obj)
	}
	for _, object := range objects {
		blob := &Blob{}
		err := blob.Decode(object)
		c.Assert(err, IsNil)
		newObject := &plumbing.MemoryObject{}
		err = blob.Encode(newObject)
		c.Assert(err, IsNil)
		newObject.Hash() // Ensure Hash is pre-computed before deep comparison
		c.Assert(newObject, DeepEquals, object)
	}
}

func (s *BlobsSuite) TestBlobIter(c *C) {
	encIter, err := s.Storer.IterEncodedObjects(plumbing.BlobObject)
	c.Assert(err, IsNil)
	iter := NewBlobIter(s.Storer, encIter)

	blobs := []*Blob{}
	iter.ForEach(func(b *Blob) error {
		blobs = append(blobs, b)
		return nil
	})

	c.Assert(len(blobs) > 0, Equals, true)
	iter.Close()

	encIter, err = s.Storer.IterEncodedObjects(plumbing.BlobObject)
	c.Assert(err, IsNil)
	iter = NewBlobIter(s.Storer, encIter)

	i := 0
	for {
		b, err := iter.Next()
		if err == io.EOF {
			break
		}

		c.Assert(err, IsNil)
		c.Assert(b.ID(), Equals, blobs[i].ID())
		c.Assert(b.Size, Equals, blobs[i].Size)
		c.Assert(b.Type(), Equals, blobs[i].Type())

		r1, err := b.Reader()
		c.Assert(err, IsNil)

		b1, err := ioutil.ReadAll(r1)
		c.Assert(err, IsNil)
		c.Assert(r1.Close(), IsNil)

		r2, err := blobs[i].Reader()
		c.Assert(err, IsNil)

		b2, err := ioutil.ReadAll(r2)
		c.Assert(err, IsNil)
		c.Assert(r2.Close(), IsNil)

		c.Assert(bytes.Compare(b1, b2), Equals, 0)
		i++
	}

	iter.Close()
}
