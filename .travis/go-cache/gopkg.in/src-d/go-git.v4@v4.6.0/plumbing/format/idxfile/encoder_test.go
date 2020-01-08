package idxfile_test

import (
	"bytes"
	"io/ioutil"

	. "gopkg.in/src-d/go-git.v4/plumbing/format/idxfile"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

func (s *IdxfileSuite) TestDecodeEncode(c *C) {
	fixtures.ByTag("packfile").Test(c, func(f *fixtures.Fixture) {
		expected, err := ioutil.ReadAll(f.Idx())
		c.Assert(err, IsNil)

		idx := new(MemoryIndex)
		d := NewDecoder(bytes.NewBuffer(expected))
		err = d.Decode(idx)
		c.Assert(err, IsNil)

		result := bytes.NewBuffer(nil)
		e := NewEncoder(result)
		size, err := e.Encode(idx)
		c.Assert(err, IsNil)

		c.Assert(size, Equals, len(expected))
		c.Assert(result.Bytes(), DeepEquals, expected)
	})
}
