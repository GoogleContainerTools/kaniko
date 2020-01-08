package idxfile_test

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/idxfile"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

type WriterSuite struct {
	fixtures.Suite
}

var _ = Suite(&WriterSuite{})

func (s *WriterSuite) TestWriter(c *C) {
	f := fixtures.Basic().One()
	scanner := packfile.NewScanner(f.Packfile())

	obs := new(idxfile.Writer)
	parser, err := packfile.NewParser(scanner, obs)
	c.Assert(err, IsNil)

	_, err = parser.Parse()
	c.Assert(err, IsNil)

	idx, err := obs.Index()
	c.Assert(err, IsNil)

	idxFile := f.Idx()
	expected, err := ioutil.ReadAll(idxFile)
	c.Assert(err, IsNil)
	idxFile.Close()

	buf := new(bytes.Buffer)
	encoder := idxfile.NewEncoder(buf)
	n, err := encoder.Encode(idx)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(expected))

	c.Assert(buf.Bytes(), DeepEquals, expected)
}

func (s *WriterSuite) TestWriterLarge(c *C) {
	writer := new(idxfile.Writer)
	err := writer.OnHeader(uint32(len(fixture4GbEntries)))
	c.Assert(err, IsNil)

	for _, o := range fixture4GbEntries {
		err = writer.OnInflatedObjectContent(plumbing.NewHash(o.hash), o.offset, o.crc, nil)
		c.Assert(err, IsNil)
	}

	err = writer.OnFooter(fixture4GbChecksum)
	c.Assert(err, IsNil)

	idx, err := writer.Index()
	c.Assert(err, IsNil)

	// load fixture index
	f := bytes.NewBufferString(fixtureLarge4GB)
	expected, err := ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, f))
	c.Assert(err, IsNil)

	buf := new(bytes.Buffer)
	encoder := idxfile.NewEncoder(buf)
	n, err := encoder.Encode(idx)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(expected))

	c.Assert(buf.Bytes(), DeepEquals, expected)
}

var (
	fixture4GbChecksum = plumbing.NewHash("afabc2269205cf85da1bf7e2fdff42f73810f29b")

	fixture4GbEntries = []struct {
		offset int64
		hash   string
		crc    uint32
	}{
		{12, "303953e5aa461c203a324821bc1717f9b4fff895", 0xbc347c4c},
		{142, "5296768e3d9f661387ccbff18c4dea6c997fd78c", 0xcdc22842},
		{1601322837, "03fc8d58d44267274edef4585eaeeb445879d33f", 0x929dfaaa},
		{2646996529, "8f3ceb4ea4cb9e4a0f751795eb41c9a4f07be772", 0xa61def8a},
		{3452385606, "e0d1d625010087f79c9e01ad9d8f95e1628dda02", 0x06bea180},
		{3707047470, "90eba326cdc4d1d61c5ad25224ccbf08731dd041", 0x7193f3ba},
		{5323223332, "bab53055add7bc35882758a922c54a874d6b1272", 0xac269b8e},
		{5894072943, "1b8995f51987d8a449ca5ea4356595102dc2fbd4", 0x2187c056},
		{5924278919, "35858be9c6f5914cbe6768489c41eb6809a2bceb", 0x9c89d9d2},
	}
)
