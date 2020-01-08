package packfile_test

import (
	"io"
	"math"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-billy.v4/osfs"
	fixtures "gopkg.in/src-d/go-git-fixtures.v3"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/idxfile"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

type PackfileSuite struct {
	fixtures.Suite
	p   *packfile.Packfile
	idx *idxfile.MemoryIndex
	f   *fixtures.Fixture
}

var _ = Suite(&PackfileSuite{})

func (s *PackfileSuite) TestGet(c *C) {
	for h := range expectedEntries {
		obj, err := s.p.Get(h)
		c.Assert(err, IsNil)
		c.Assert(obj, Not(IsNil))
		c.Assert(obj.Hash(), Equals, h)
	}

	_, err := s.p.Get(plumbing.ZeroHash)
	c.Assert(err, Equals, plumbing.ErrObjectNotFound)
}

func (s *PackfileSuite) TestGetByOffset(c *C) {
	for h, o := range expectedEntries {
		obj, err := s.p.GetByOffset(o)
		c.Assert(err, IsNil)
		c.Assert(obj, Not(IsNil))
		c.Assert(obj.Hash(), Equals, h)
	}

	_, err := s.p.GetByOffset(math.MaxInt64)
	c.Assert(err, Equals, plumbing.ErrObjectNotFound)
}

func (s *PackfileSuite) TestID(c *C) {
	id, err := s.p.ID()
	c.Assert(err, IsNil)
	c.Assert(id, Equals, s.f.PackfileHash)
}

func (s *PackfileSuite) TestGetAll(c *C) {
	iter, err := s.p.GetAll()
	c.Assert(err, IsNil)

	var objects int
	for {
		o, err := iter.Next()
		if err == io.EOF {
			break
		}
		c.Assert(err, IsNil)

		objects++
		_, ok := expectedEntries[o.Hash()]
		c.Assert(ok, Equals, true)
	}

	c.Assert(objects, Equals, len(expectedEntries))
}

var expectedEntries = map[plumbing.Hash]int64{
	plumbing.NewHash("1669dce138d9b841a518c64b10914d88f5e488ea"): 615,
	plumbing.NewHash("32858aad3c383ed1ff0a0f9bdf231d54a00c9e88"): 1524,
	plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9"): 1063,
	plumbing.NewHash("49c6bb89b17060d7b4deacb7b338fcc6ea2352a9"): 78882,
	plumbing.NewHash("4d081c50e250fa32ea8b1313cf8bb7c2ad7627fd"): 84688,
	plumbing.NewHash("586af567d0bb5e771e49bdd9434f5e0fb76d25fa"): 84559,
	plumbing.NewHash("5a877e6a906a2743ad6e45d99c1793642aaf8eda"): 84479,
	plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"): 186,
	plumbing.NewHash("7e59600739c96546163833214c36459e324bad0a"): 84653,
	plumbing.NewHash("880cd14280f4b9b6ed3986d6671f907d7cc2a198"): 78050,
	plumbing.NewHash("8dcef98b1d52143e1e2dbc458ffe38f925786bf2"): 84741,
	plumbing.NewHash("918c48b83bd081e863dbe1b80f8998f058cd8294"): 286,
	plumbing.NewHash("9a48f23120e880dfbe41f7c9b7b708e9ee62a492"): 80998,
	plumbing.NewHash("9dea2395f5403188298c1dabe8bdafe562c491e3"): 84032,
	plumbing.NewHash("a39771a7651f97faf5c72e08224d857fc35133db"): 84430,
	plumbing.NewHash("a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69"): 838,
	plumbing.NewHash("a8d315b2b1c615d43042c3a62402b8a54288cf5c"): 84375,
	plumbing.NewHash("aa9b383c260e1d05fbbf6b30a02914555e20c725"): 84760,
	plumbing.NewHash("af2d6a6954d532f8ffb47615169c8fdf9d383a1a"): 449,
	plumbing.NewHash("b029517f6300c2da0f4b651b8642506cd6aaf45d"): 1392,
	plumbing.NewHash("b8e471f58bcbca63b07bda20e428190409c2db47"): 1230,
	plumbing.NewHash("c192bd6a24ea1ab01d78686e417c8bdc7c3d197f"): 1713,
	plumbing.NewHash("c2d30fa8ef288618f65f6eed6e168e0d514886f4"): 84725,
	plumbing.NewHash("c8f1d8c61f9da76f4cb49fd86322b6e685dba956"): 80725,
	plumbing.NewHash("cf4aa3b38974fb7d81f367c0830f7d78d65ab86b"): 84608,
	plumbing.NewHash("d3ff53e0564a9f87d8e84b6e28e5060e517008aa"): 1685,
	plumbing.NewHash("d5c0f4ab811897cadf03aec358ae60d21f91c50d"): 2351,
	plumbing.NewHash("dbd3641b371024f44d0e469a9c8f5457b0660de1"): 84115,
	plumbing.NewHash("e8d3ffab552895c19b9fcf7aa264d277cde33881"): 12,
	plumbing.NewHash("eba74343e2f15d62adedfd8c883ee0262b5c8021"): 84708,
	plumbing.NewHash("fb72698cab7617ac416264415f13224dfd7a165e"): 84671,
}

func (s *PackfileSuite) SetUpTest(c *C) {
	s.f = fixtures.Basic().One()

	fs := osfs.New("")
	f, err := fs.Open(s.f.Packfile().Name())
	c.Assert(err, IsNil)

	s.idx = idxfile.NewMemoryIndex()
	c.Assert(idxfile.NewDecoder(s.f.Idx()).Decode(s.idx), IsNil)

	s.p = packfile.NewPackfile(s.idx, fs, f)
}

func (s *PackfileSuite) TearDownTest(c *C) {
	c.Assert(s.p.Close(), IsNil)
}

func (s *PackfileSuite) TestDecode(c *C) {
	fixtures.Basic().ByTag("packfile").Test(c, func(f *fixtures.Fixture) {
		index := getIndexFromIdxFile(f.Idx())
		fs := osfs.New("")
		pf, err := fs.Open(f.Packfile().Name())
		c.Assert(err, IsNil)

		p := packfile.NewPackfile(index, fs, pf)
		defer p.Close()

		for _, h := range expectedHashes {
			obj, err := p.Get(plumbing.NewHash(h))
			c.Assert(err, IsNil)
			c.Assert(obj.Hash().String(), Equals, h)
		}
	})
}

func (s *PackfileSuite) TestDecodeByTypeRefDelta(c *C) {
	f := fixtures.Basic().ByTag("ref-delta").One()

	index := getIndexFromIdxFile(f.Idx())
	fs := osfs.New("")
	pf, err := fs.Open(f.Packfile().Name())
	c.Assert(err, IsNil)

	packfile := packfile.NewPackfile(index, fs, pf)
	defer packfile.Close()

	iter, err := packfile.GetByType(plumbing.CommitObject)
	c.Assert(err, IsNil)

	var count int
	for {
		obj, err := iter.Next()
		if err == io.EOF {
			break
		}
		count++
		c.Assert(err, IsNil)
		c.Assert(obj.Type(), Equals, plumbing.CommitObject)
	}

	c.Assert(count > 0, Equals, true)
}

func (s *PackfileSuite) TestDecodeByType(c *C) {
	ts := []plumbing.ObjectType{
		plumbing.CommitObject,
		plumbing.TagObject,
		plumbing.TreeObject,
		plumbing.BlobObject,
	}

	fixtures.Basic().ByTag("packfile").Test(c, func(f *fixtures.Fixture) {
		for _, t := range ts {
			index := getIndexFromIdxFile(f.Idx())
			fs := osfs.New("")
			pf, err := fs.Open(f.Packfile().Name())
			c.Assert(err, IsNil)

			packfile := packfile.NewPackfile(index, fs, pf)
			defer packfile.Close()

			iter, err := packfile.GetByType(t)
			c.Assert(err, IsNil)

			c.Assert(iter.ForEach(func(obj plumbing.EncodedObject) error {
				c.Assert(obj.Type(), Equals, t)
				return nil
			}), IsNil)
		}
	})
}

func (s *PackfileSuite) TestDecodeByTypeConstructor(c *C) {
	f := fixtures.Basic().ByTag("packfile").One()
	index := getIndexFromIdxFile(f.Idx())
	fs := osfs.New("")
	pf, err := fs.Open(f.Packfile().Name())
	c.Assert(err, IsNil)

	packfile := packfile.NewPackfile(index, fs, pf)
	defer packfile.Close()

	_, err = packfile.GetByType(plumbing.OFSDeltaObject)
	c.Assert(err, Equals, plumbing.ErrInvalidType)

	_, err = packfile.GetByType(plumbing.REFDeltaObject)
	c.Assert(err, Equals, plumbing.ErrInvalidType)

	_, err = packfile.GetByType(plumbing.InvalidObject)
	c.Assert(err, Equals, plumbing.ErrInvalidType)
}

var expectedHashes = []string{
	"918c48b83bd081e863dbe1b80f8998f058cd8294",
	"af2d6a6954d532f8ffb47615169c8fdf9d383a1a",
	"1669dce138d9b841a518c64b10914d88f5e488ea",
	"a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
	"b8e471f58bcbca63b07bda20e428190409c2db47",
	"35e85108805c84807bc66a02d91535e1e24b38b9",
	"b029517f6300c2da0f4b651b8642506cd6aaf45d",
	"32858aad3c383ed1ff0a0f9bdf231d54a00c9e88",
	"d3ff53e0564a9f87d8e84b6e28e5060e517008aa",
	"c192bd6a24ea1ab01d78686e417c8bdc7c3d197f",
	"d5c0f4ab811897cadf03aec358ae60d21f91c50d",
	"49c6bb89b17060d7b4deacb7b338fcc6ea2352a9",
	"cf4aa3b38974fb7d81f367c0830f7d78d65ab86b",
	"9dea2395f5403188298c1dabe8bdafe562c491e3",
	"586af567d0bb5e771e49bdd9434f5e0fb76d25fa",
	"9a48f23120e880dfbe41f7c9b7b708e9ee62a492",
	"5a877e6a906a2743ad6e45d99c1793642aaf8eda",
	"c8f1d8c61f9da76f4cb49fd86322b6e685dba956",
	"a8d315b2b1c615d43042c3a62402b8a54288cf5c",
	"a39771a7651f97faf5c72e08224d857fc35133db",
	"880cd14280f4b9b6ed3986d6671f907d7cc2a198",
	"fb72698cab7617ac416264415f13224dfd7a165e",
	"4d081c50e250fa32ea8b1313cf8bb7c2ad7627fd",
	"eba74343e2f15d62adedfd8c883ee0262b5c8021",
	"c2d30fa8ef288618f65f6eed6e168e0d514886f4",
	"8dcef98b1d52143e1e2dbc458ffe38f925786bf2",
	"aa9b383c260e1d05fbbf6b30a02914555e20c725",
	"6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
	"dbd3641b371024f44d0e469a9c8f5457b0660de1",
	"e8d3ffab552895c19b9fcf7aa264d277cde33881",
	"7e59600739c96546163833214c36459e324bad0a",
}

func assertObjects(c *C, s storer.EncodedObjectStorer, expects []string) {
	i, err := s.IterEncodedObjects(plumbing.AnyObject)
	c.Assert(err, IsNil)

	var count int
	err = i.ForEach(func(plumbing.EncodedObject) error { count++; return nil })
	c.Assert(err, IsNil)
	c.Assert(count, Equals, len(expects))

	for _, exp := range expects {
		obt, err := s.EncodedObject(plumbing.AnyObject, plumbing.NewHash(exp))
		c.Assert(err, IsNil)
		c.Assert(obt.Hash().String(), Equals, exp)
	}
}

func getIndexFromIdxFile(r io.Reader) idxfile.Index {
	idxf := idxfile.NewMemoryIndex()
	d := idxfile.NewDecoder(r)
	if err := d.Decode(idxf); err != nil {
		panic(err)
	}

	return idxf
}
