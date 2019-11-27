package executor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_NewCompositeCache(t *testing.T) {
	r := NewCompositeCache()
	if reflect.TypeOf(r).String() != "*executor.CompositeCache" {
		t.Errorf("expected return to be *executor.CompositeCache but was %v", reflect.TypeOf(r).String())
	}
}

func Test_CompositeCache_AddKey(t *testing.T) {
	keys := []string{
		"meow",
		"purr",
	}
	r := NewCompositeCache()
	r.AddKey(keys...)
	if len(r.keys) != 2 {
		t.Errorf("expected keys to have length 2 but was %v", len(r.keys))
	}
}

func Test_CompositeCache_Key(t *testing.T) {
	r := NewCompositeCache("meow", "purr")
	k := r.Key()
	if k != "meow-purr" {
		t.Errorf("expected result to equal meow-purr but was %v", k)
	}
}

func Test_CompositeCache_Hash(t *testing.T) {
	r := NewCompositeCache("meow", "purr")
	h, err := r.Hash()
	if err != nil {
		t.Errorf("expected error to be nil but was %v", err)
	}

	expectedHash := "b4fd5a11af812a11a79d794007c842794cc668c8e7ebaba6d1e6d021b8e06c71"
	if h != expectedHash {
		t.Errorf("expected result to equal %v but was %v", expectedHash, h)
	}
}

func Test_CompositeCache_AddPath_dir(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "foo")
	if err != nil {
		t.Errorf("got error setting up test %v", err)
	}

	content := `meow meow meow`
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "foo.txt"), []byte(content), 0777); err != nil {
		t.Errorf("got error writing temp file %v", err)
	}

	fn := func() string {
		r := NewCompositeCache()
		if err := r.AddPath(tmpDir); err != nil {
			t.Errorf("expected error to be nil but was %v", err)
		}

		if len(r.keys) != 1 {
			t.Errorf("expected len of keys to be 1 but was %v", len(r.keys))
		}
		hash, err := r.Hash()
		if err != nil {
			t.Errorf("couldnt generate hash from test cache")
		}
		return hash
	}

	hash1 := fn()
	hash2 := fn()
	if hash1 != hash2 {
		t.Errorf("expected hash %v to equal hash %v", hash1, hash2)
	}
}
func Test_CompositeCache_AddPath_file(t *testing.T) {
	tmpfile, err := ioutil.TempFile("/tmp", "foo.txt")
	if err != nil {
		t.Errorf("got error setting up test %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := `meow meow meow`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Errorf("got error writing temp file %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Errorf("got error closing temp file %v", err)
	}

	p := tmpfile.Name()
	fn := func() string {
		r := NewCompositeCache()
		if err := r.AddPath(p); err != nil {
			t.Errorf("expected error to be nil but was %v", err)
		}

		if len(r.keys) != 1 {
			t.Errorf("expected len of keys to be 1 but was %v", len(r.keys))
		}
		hash, err := r.Hash()
		if err != nil {
			t.Errorf("couldnt generate hash from test cache")
		}
		return hash
	}

	hash1 := fn()
	hash2 := fn()
	if hash1 != hash2 {
		t.Errorf("expected hash %v to equal hash %v", hash1, hash2)
	}
}
