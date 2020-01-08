package packfile_test

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

type ParserSuite struct {
	fixtures.Suite
}

var _ = Suite(&ParserSuite{})

func (s *ParserSuite) TestParserHashes(c *C) {
	f := fixtures.Basic().One()
	scanner := packfile.NewScanner(f.Packfile())

	obs := new(testObserver)
	parser, err := packfile.NewParser(scanner, obs)
	c.Assert(err, IsNil)

	ch, err := parser.Parse()
	c.Assert(err, IsNil)

	checksum := "a3fed42da1e8189a077c0e6846c040dcf73fc9dd"
	c.Assert(ch.String(), Equals, checksum)

	c.Assert(obs.checksum, Equals, checksum)
	c.Assert(int(obs.count), Equals, int(31))

	commit := plumbing.CommitObject
	blob := plumbing.BlobObject
	tree := plumbing.TreeObject

	objs := []observerObject{
		{"e8d3ffab552895c19b9fcf7aa264d277cde33881", commit, 254, 12, 0xaa07ba4b},
		{"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", commit, 245, 186, 0xf706df58},
		{"918c48b83bd081e863dbe1b80f8998f058cd8294", commit, 242, 286, 0x12438846},
		{"af2d6a6954d532f8ffb47615169c8fdf9d383a1a", commit, 242, 449, 0x2905a38c},
		{"1669dce138d9b841a518c64b10914d88f5e488ea", commit, 333, 615, 0xd9429436},
		{"a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69", commit, 332, 838, 0xbecfde4e},
		{"35e85108805c84807bc66a02d91535e1e24b38b9", commit, 244, 1063, 0x780e4b3e},
		{"b8e471f58bcbca63b07bda20e428190409c2db47", commit, 243, 1230, 0xdc18344f},
		{"b029517f6300c2da0f4b651b8642506cd6aaf45d", commit, 187, 1392, 0xcf4e4280},
		{"32858aad3c383ed1ff0a0f9bdf231d54a00c9e88", blob, 189, 1524, 0x1f08118a},
		{"d3ff53e0564a9f87d8e84b6e28e5060e517008aa", blob, 18, 1685, 0xafded7b8},
		{"c192bd6a24ea1ab01d78686e417c8bdc7c3d197f", blob, 1072, 1713, 0xcc1428ed},
		{"d5c0f4ab811897cadf03aec358ae60d21f91c50d", blob, 76110, 2351, 0x1631d22f},
		{"880cd14280f4b9b6ed3986d6671f907d7cc2a198", blob, 2780, 78050, 0xbfff5850},
		{"49c6bb89b17060d7b4deacb7b338fcc6ea2352a9", blob, 217848, 78882, 0xd108e1d8},
		{"c8f1d8c61f9da76f4cb49fd86322b6e685dba956", blob, 706, 80725, 0x8e97ba25},
		{"9a48f23120e880dfbe41f7c9b7b708e9ee62a492", blob, 11488, 80998, 0x7316ff70},
		{"9dea2395f5403188298c1dabe8bdafe562c491e3", blob, 78, 84032, 0xdb4fce56},
		{"dbd3641b371024f44d0e469a9c8f5457b0660de1", tree, 272, 84115, 0x901cce2c},
		{"a8d315b2b1c615d43042c3a62402b8a54288cf5c", tree, 271, 84375, 0xec4552b0},
		{"a39771a7651f97faf5c72e08224d857fc35133db", tree, 38, 84430, 0x847905bf},
		{"5a877e6a906a2743ad6e45d99c1793642aaf8eda", tree, 75, 84479, 0x3689459a},
		{"586af567d0bb5e771e49bdd9434f5e0fb76d25fa", tree, 38, 84559, 0xe67af94a},
		{"cf4aa3b38974fb7d81f367c0830f7d78d65ab86b", tree, 34, 84608, 0xc2314a2e},
		{"7e59600739c96546163833214c36459e324bad0a", blob, 9, 84653, 0xcd987848},
		{"fb72698cab7617ac416264415f13224dfd7a165e", tree, 238, 84671, 0x8a853a6d},
		{"4d081c50e250fa32ea8b1313cf8bb7c2ad7627fd", tree, 179, 84688, 0x70c6518},
		{"eba74343e2f15d62adedfd8c883ee0262b5c8021", tree, 148, 84708, 0x4f4108e2},
		{"c2d30fa8ef288618f65f6eed6e168e0d514886f4", tree, 110, 84725, 0xd6fe09e9},
		{"8dcef98b1d52143e1e2dbc458ffe38f925786bf2", tree, 111, 84741, 0xf07a2804},
		{"aa9b383c260e1d05fbbf6b30a02914555e20c725", tree, 73, 84760, 0x1d75d6be},
	}

	c.Assert(obs.objects, DeepEquals, objs)
}

type observerObject struct {
	hash   string
	otype  plumbing.ObjectType
	size   int64
	offset int64
	crc    uint32
}

type testObserver struct {
	count    uint32
	checksum string
	objects  []observerObject
	pos      map[int64]int
}

func (t *testObserver) OnHeader(count uint32) error {
	t.count = count
	t.pos = make(map[int64]int, count)
	return nil
}

func (t *testObserver) OnInflatedObjectHeader(otype plumbing.ObjectType, objSize int64, pos int64) error {
	o := t.get(pos)
	o.otype = otype
	o.size = objSize
	o.offset = pos

	t.put(pos, o)

	return nil
}

func (t *testObserver) OnInflatedObjectContent(h plumbing.Hash, pos int64, crc uint32, _ []byte) error {
	o := t.get(pos)
	o.hash = h.String()
	o.crc = crc

	t.put(pos, o)

	return nil
}

func (t *testObserver) OnFooter(h plumbing.Hash) error {
	t.checksum = h.String()
	return nil
}

func (t *testObserver) get(pos int64) observerObject {
	i, ok := t.pos[pos]
	if ok {
		return t.objects[i]
	}

	return observerObject{}
}

func (t *testObserver) put(pos int64, o observerObject) {
	i, ok := t.pos[pos]
	if ok {
		t.objects[i] = o
		return
	}

	t.pos[pos] = len(t.objects)
	t.objects = append(t.objects, o)
}

func BenchmarkParse(b *testing.B) {
	if err := fixtures.Init(); err != nil {
		b.Fatal(err)
	}

	defer func() {
		if err := fixtures.Clean(); err != nil {
			b.Fatal(err)
		}
	}()

	for _, f := range fixtures.ByTag("packfile") {
		b.Run(f.URL, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				parser, err := packfile.NewParser(packfile.NewScanner(f.Packfile()))
				if err != nil {
					b.Fatal(err)
				}

				_, err = parser.Parse()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseBasic(b *testing.B) {
	if err := fixtures.Init(); err != nil {
		b.Fatal(err)
	}

	defer func() {
		if err := fixtures.Clean(); err != nil {
			b.Fatal(err)
		}
	}()

	f := fixtures.Basic().One()
	for i := 0; i < b.N; i++ {
		parser, err := packfile.NewParser(packfile.NewScanner(f.Packfile()))
		if err != nil {
			b.Fatal(err)
		}

		_, err = parser.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}
