package git

import (
	"bytes"
	"strings"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/errors"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/util"
)

func (s *WorktreeSuite) TestCommitInvalidOptions(c *C) {
	r, err := Init(memory.NewStorage(), memfs.New())
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	hash, err := w.Commit("", &CommitOptions{})
	c.Assert(err, Equals, ErrMissingAuthor)
	c.Assert(hash.IsZero(), Equals, true)
}

func (s *WorktreeSuite) TestCommitInitial(c *C) {
	expected := plumbing.NewHash("98c4ac7c29c913f7461eae06e024dc18e80d23a4")

	fs := memfs.New()
	storage := memory.NewStorage()

	r, err := Init(storage, fs)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	util.WriteFile(fs, "foo", []byte("foo"), 0644)

	_, err = w.Add("foo")
	c.Assert(err, IsNil)

	hash, err := w.Commit("foo\n", &CommitOptions{Author: defaultSignature()})
	c.Assert(hash, Equals, expected)
	c.Assert(err, IsNil)

	assertStorageStatus(c, r, 1, 1, 1, expected)
}

func (s *WorktreeSuite) TestCommitParent(c *C) {
	expected := plumbing.NewHash("ef3ca05477530b37f48564be33ddd48063fc7a22")

	fs := memfs.New()
	w := &Worktree{
		r:          s.Repository,
		Filesystem: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	util.WriteFile(fs, "foo", []byte("foo"), 0644)

	_, err = w.Add("foo")
	c.Assert(err, IsNil)

	hash, err := w.Commit("foo\n", &CommitOptions{Author: defaultSignature()})
	c.Assert(hash, Equals, expected)
	c.Assert(err, IsNil)

	assertStorageStatus(c, s.Repository, 13, 11, 10, expected)
}

func (s *WorktreeSuite) TestCommitAll(c *C) {
	expected := plumbing.NewHash("aede6f8c9c1c7ec9ca8d287c64b8ed151276fa28")

	fs := memfs.New()
	w := &Worktree{
		r:          s.Repository,
		Filesystem: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	util.WriteFile(fs, "LICENSE", []byte("foo"), 0644)
	util.WriteFile(fs, "foo", []byte("foo"), 0644)

	hash, err := w.Commit("foo\n", &CommitOptions{
		All:    true,
		Author: defaultSignature(),
	})

	c.Assert(hash, Equals, expected)
	c.Assert(err, IsNil)

	assertStorageStatus(c, s.Repository, 13, 11, 10, expected)
}

func (s *WorktreeSuite) TestRemoveAndCommitAll(c *C) {
	expected := plumbing.NewHash("907cd576c6ced2ecd3dab34a72bf9cf65944b9a9")

	fs := memfs.New()
	w := &Worktree{
		r:          s.Repository,
		Filesystem: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	util.WriteFile(fs, "foo", []byte("foo"), 0644)
	_, err = w.Add("foo")
	c.Assert(err, IsNil)

	_, errFirst := w.Commit("Add in Repo\n", &CommitOptions{
		Author: defaultSignature(),
	})
	c.Assert(errFirst, IsNil)

	errRemove := fs.Remove("foo")
	c.Assert(errRemove, IsNil)

	hash, errSecond := w.Commit("Remove foo\n", &CommitOptions{
		All:    true,
		Author: defaultSignature(),
	})
	c.Assert(errSecond, IsNil)

	c.Assert(hash, Equals, expected)
	c.Assert(err, IsNil)

	assertStorageStatus(c, s.Repository, 13, 11, 11, expected)
}

func (s *WorktreeSuite) TestCommitSign(c *C) {
	fs := memfs.New()
	storage := memory.NewStorage()

	r, err := Init(storage, fs)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	util.WriteFile(fs, "foo", []byte("foo"), 0644)

	_, err = w.Add("foo")
	c.Assert(err, IsNil)

	key := commitSignKey(c, true)
	hash, err := w.Commit("foo\n", &CommitOptions{Author: defaultSignature(), SignKey: key})
	c.Assert(err, IsNil)

	// Verify the commit.
	pks := new(bytes.Buffer)
	pkw, err := armor.Encode(pks, openpgp.PublicKeyType, nil)
	c.Assert(err, IsNil)

	err = key.Serialize(pkw)
	c.Assert(err, IsNil)
	err = pkw.Close()
	c.Assert(err, IsNil)

	expectedCommit, err := r.CommitObject(hash)
	c.Assert(err, IsNil)
	actual, err := expectedCommit.Verify(pks.String())
	c.Assert(err, IsNil)
	c.Assert(actual.PrimaryKey, DeepEquals, key.PrimaryKey)
}

func (s *WorktreeSuite) TestCommitSignBadKey(c *C) {
	fs := memfs.New()
	storage := memory.NewStorage()

	r, err := Init(storage, fs)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	util.WriteFile(fs, "foo", []byte("foo"), 0644)

	_, err = w.Add("foo")
	c.Assert(err, IsNil)

	key := commitSignKey(c, false)
	_, err = w.Commit("foo\n", &CommitOptions{Author: defaultSignature(), SignKey: key})
	c.Assert(err, Equals, errors.InvalidArgumentError("signing key is encrypted"))
}

func assertStorageStatus(
	c *C, r *Repository,
	treesCount, blobCount, commitCount int, head plumbing.Hash,
) {
	trees, err := r.Storer.IterEncodedObjects(plumbing.TreeObject)
	c.Assert(err, IsNil)
	blobs, err := r.Storer.IterEncodedObjects(plumbing.BlobObject)
	c.Assert(err, IsNil)
	commits, err := r.Storer.IterEncodedObjects(plumbing.CommitObject)
	c.Assert(err, IsNil)

	c.Assert(lenIterEncodedObjects(trees), Equals, treesCount)
	c.Assert(lenIterEncodedObjects(blobs), Equals, blobCount)
	c.Assert(lenIterEncodedObjects(commits), Equals, commitCount)

	ref, err := r.Head()
	c.Assert(err, IsNil)
	c.Assert(ref.Hash(), Equals, head)
}

func lenIterEncodedObjects(iter storer.EncodedObjectIter) int {
	count := 0
	iter.ForEach(func(plumbing.EncodedObject) error {
		count++
		return nil
	})

	return count
}

func defaultSignature() *object.Signature {
	when, _ := time.Parse(object.DateFormat, "Thu May 04 00:03:43 2017 +0200")
	return &object.Signature{
		Name:  "foo",
		Email: "foo@foo.foo",
		When:  when,
	}
}

func commitSignKey(c *C, decrypt bool) *openpgp.Entity {
	s := strings.NewReader(armoredKeyRing)
	es, err := openpgp.ReadArmoredKeyRing(s)
	c.Assert(err, IsNil)

	c.Assert(es, HasLen, 1)
	c.Assert(es[0].Identities, HasLen, 1)
	_, ok := es[0].Identities["foo bar <foo@foo.foo>"]
	c.Assert(ok, Equals, true)

	key := es[0]
	if decrypt {
		err = key.PrivateKey.Decrypt([]byte(keyPassphrase))
		c.Assert(err, IsNil)
	}

	return key
}

const armoredKeyRing = `
-----BEGIN PGP PRIVATE KEY BLOCK-----

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
=8g3t
-----END PGP PRIVATE KEY BLOCK-----
`

const keyPassphrase = "abcdef0123456789"
