package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/spf13/afero"
)

func mustTag(t *testing.T, s string) name.Tag {
	tag, err := name.NewTag(s, name.StrictValidation)
	if err != nil {
		t.Fatalf("NewTag: %v", err)
	}
	return tag
}

func TestWriteImageOutputs(t *testing.T) {
	img, err := random.Image(1024, 3)
	if err != nil {
		t.Fatalf("random.Image: %v", err)
	}
	d, err := img.Digest()
	if err != nil {
		t.Fatalf("Digest: %v", err)
	}

	for _, c := range []struct {
		desc, env string
		tags      []name.Tag
		want      string
	}{{
		desc: "env unset, no output",
		env:  "",
	}, {
		desc: "env set, one tag",
		env:  "/foo",
		tags: []name.Tag{mustTag(t, "gcr.io/foo/bar:latest")},
		want: fmt.Sprintf(`{"name":"gcr.io/foo/bar:latest","digest":%q}
`, d),
	}, {
		desc: "env set, two tags",
		env:  "/foo",
		tags: []name.Tag{
			mustTag(t, "gcr.io/foo/bar:latest"),
			mustTag(t, "gcr.io/baz/qux:latest"),
		},
		want: fmt.Sprintf(`{"name":"gcr.io/foo/bar:latest","digest":%q}
{"name":"gcr.io/baz/qux:latest","digest":%q}
`, d, d),
	}} {
		t.Run(c.desc, func(t *testing.T) {
			fs = afero.NewMemMapFs()
			if c.want == "" {
				fs = afero.NewReadOnlyFs(fs) // No files should be written.
			}

			os.Setenv("BUILDER_OUTPUT", c.env)
			if err := writeImageOutputs(img, c.tags); err != nil {
				t.Fatalf("writeImageOutputs: %v", err)
			}

			if c.want == "" {
				return
			}

			b, err := afero.ReadFile(fs, filepath.Join(c.env, "images"))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}

			if got := string(b); got != c.want {
				t.Fatalf(" got: %s\nwant: %s", got, c.want)
			}
		})
	}
}
