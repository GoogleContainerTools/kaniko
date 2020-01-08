package idxfile_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing"
	. "gopkg.in/src-d/go-git.v4/plumbing/format/idxfile"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

func Test(t *testing.T) { TestingT(t) }

type IdxfileSuite struct {
	fixtures.Suite
}

var _ = Suite(&IdxfileSuite{})

func (s *IdxfileSuite) TestDecode(c *C) {
	f := fixtures.Basic().One()

	d := NewDecoder(f.Idx())
	idx := new(MemoryIndex)
	err := d.Decode(idx)
	c.Assert(err, IsNil)

	count, _ := idx.Count()
	c.Assert(count, Equals, int64(31))

	hash := plumbing.NewHash("1669dce138d9b841a518c64b10914d88f5e488ea")
	ok, err := idx.Contains(hash)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	offset, err := idx.FindOffset(hash)
	c.Assert(err, IsNil)
	c.Assert(offset, Equals, int64(615))

	crc32, err := idx.FindCRC32(hash)
	c.Assert(err, IsNil)
	c.Assert(crc32, Equals, uint32(3645019190))

	c.Assert(fmt.Sprintf("%x", idx.IdxChecksum), Equals, "fb794f1ec720b9bc8e43257451bd99c4be6fa1c9")
	c.Assert(fmt.Sprintf("%x", idx.PackfileChecksum), Equals, f.PackfileHash.String())
}

func (s *IdxfileSuite) TestDecode64bitsOffsets(c *C) {
	f := bytes.NewBufferString(fixtureLarge4GB)

	idx := new(MemoryIndex)

	d := NewDecoder(base64.NewDecoder(base64.StdEncoding, f))
	err := d.Decode(idx)
	c.Assert(err, IsNil)

	expected := map[string]uint64{
		"303953e5aa461c203a324821bc1717f9b4fff895": 12,
		"5296768e3d9f661387ccbff18c4dea6c997fd78c": 142,
		"03fc8d58d44267274edef4585eaeeb445879d33f": 1601322837,
		"8f3ceb4ea4cb9e4a0f751795eb41c9a4f07be772": 2646996529,
		"e0d1d625010087f79c9e01ad9d8f95e1628dda02": 3452385606,
		"90eba326cdc4d1d61c5ad25224ccbf08731dd041": 3707047470,
		"bab53055add7bc35882758a922c54a874d6b1272": 5323223332,
		"1b8995f51987d8a449ca5ea4356595102dc2fbd4": 5894072943,
		"35858be9c6f5914cbe6768489c41eb6809a2bceb": 5924278919,
	}

	iter, err := idx.Entries()
	c.Assert(err, IsNil)

	var entries int
	for {
		e, err := iter.Next()
		if err == io.EOF {
			break
		}
		c.Assert(err, IsNil)
		entries++

		c.Assert(expected[e.Hash.String()], Equals, e.Offset)
	}

	c.Assert(entries, Equals, len(expected))
}

const fixtureLarge4GB = `/3RPYwAAAAIAAAAAAAAAAAAAAAAAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEA
AAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAA
AAEAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAA
AgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAADAAAAAwAAAAMAAAADAAAAAwAAAAQAAAAE
AAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQA
AAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABAAAAAQAAAAEAAAABQAA
AAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAA
BQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAF
AAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUA
AAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAAAAUAAAAFAAAABQAA
AAUAAAAFAAAABQAAAAYAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAA
BwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAH
AAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcAAAAHAAAABwAAAAcA
AAAHAAAABwAAAAcAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAA
AAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAA
CAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAgAAAAIAAAACAAAAAkAAAAJ
AAAACQAAAAkAAAAJAAAACQAAAAkAAAAJAAAACQAAAAkAAAAJAAAACQAAAAkAAAAJAAAACQAAAAkA
AAAJAAAACQAAAAkAAAAJAAAACQAAAAkAAAAJAAAACQAAAAkAAAAJAAAACQAAAAkAAAAJAAAACQAA
AAkAAAAJA/yNWNRCZydO3vRYXq7rRFh50z8biZX1GYfYpEnKXqQ1ZZUQLcL71DA5U+WqRhwgOjJI
IbwXF/m0//iVNYWL6cb1kUy+Z2hInEHraAmivOtSlnaOPZ9mE4fMv/GMTepsmX/XjI88606ky55K
D3UXletByaTwe+dykOujJs3E0dYcWtJSJMy/CHMd0EG6tTBVrde8NYgnWKkixUqHTWsScuDR1iUB
AIf3nJ4BrZ2PleFijdoCkp36qiGHwFa8NHxMnInZ0s3CKEKmHe+KcZPzuqwmm44GvqGAX3I/VYAA
AAAAAAAMgAAAAQAAAI6AAAACgAAAA4AAAASAAAAFAAAAAV9Qam8AAAABYR1ShwAAAACdxfYxAAAA
ANz1Di4AAAABPUnxJAAAAADNxzlGr6vCJpIFz4XaG/fi/f9C9zgQ8ptKSQpfQ1NMJBGTDTxxYGGp
ch2xUA==
`

func BenchmarkDecode(b *testing.B) {
	if err := fixtures.Init(); err != nil {
		b.Errorf("unexpected error initializing fixtures: %s", err)
	}

	f := fixtures.Basic().One()
	fixture, err := ioutil.ReadAll(f.Idx())
	if err != nil {
		b.Errorf("unexpected error reading idx file: %s", err)
	}

	defer func() {
		if err := fixtures.Clean(); err != nil {
			b.Errorf("unexpected error cleaning fixtures: %s", err)
		}
	}()

	for i := 0; i < b.N; i++ {
		f := bytes.NewBuffer(fixture)
		idx := new(MemoryIndex)
		d := NewDecoder(f)
		if err := d.Decode(idx); err != nil {
			b.Errorf("unexpected error decoding: %s", err)
		}
	}
}
