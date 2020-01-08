package ocischema

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var expectedManifestSerialization = []byte(`{
   "schemaVersion": 2,
   "mediaType": "application/vnd.oci.image.manifest.v1+json",
   "config": {
      "mediaType": "application/vnd.oci.image.config.v1+json",
      "size": 985,
      "digest": "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b",
      "annotations": {
         "apple": "orange"
      }
   },
   "layers": [
      {
         "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
         "size": 153263,
         "digest": "sha256:62d8908bee94c202b2d35224a221aaa2058318bfa9879fa541efaecba272331b",
         "annotations": {
            "lettuce": "wrap"
         }
      }
   ],
   "annotations": {
      "hot": "potato"
   }
}`)

func makeTestManifest(mediaType string) Manifest {
	return Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 2,
			MediaType:     mediaType,
		},
		Config: distribution.Descriptor{
			Digest:      "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b",
			Size:        985,
			MediaType:   v1.MediaTypeImageConfig,
			Annotations: map[string]string{"apple": "orange"},
		},
		Layers: []distribution.Descriptor{
			{
				Digest:      "sha256:62d8908bee94c202b2d35224a221aaa2058318bfa9879fa541efaecba272331b",
				Size:        153263,
				MediaType:   v1.MediaTypeImageLayerGzip,
				Annotations: map[string]string{"lettuce": "wrap"},
			},
		},
		Annotations: map[string]string{"hot": "potato"},
	}
}

func TestManifest(t *testing.T) {
	manifest := makeTestManifest(v1.MediaTypeImageManifest)

	deserialized, err := FromStruct(manifest)
	if err != nil {
		t.Fatalf("error creating DeserializedManifest: %v", err)
	}

	mediaType, canonical, _ := deserialized.Payload()

	if mediaType != v1.MediaTypeImageManifest {
		t.Fatalf("unexpected media type: %s", mediaType)
	}

	// Check that the canonical field is the same as json.MarshalIndent
	// with these parameters.
	p, err := json.MarshalIndent(&manifest, "", "   ")
	if err != nil {
		t.Fatalf("error marshaling manifest: %v", err)
	}
	if !bytes.Equal(p, canonical) {
		t.Fatalf("manifest bytes not equal: %q != %q", string(canonical), string(p))
	}

	// Check that canonical field matches expected value.
	if !bytes.Equal(expectedManifestSerialization, canonical) {
		t.Fatalf("manifest bytes not equal: %q != %q", string(canonical), string(expectedManifestSerialization))
	}

	var unmarshalled DeserializedManifest
	if err := json.Unmarshal(deserialized.canonical, &unmarshalled); err != nil {
		t.Fatalf("error unmarshaling manifest: %v", err)
	}

	if !reflect.DeepEqual(&unmarshalled, deserialized) {
		t.Fatalf("manifests are different after unmarshaling: %v != %v", unmarshalled, *deserialized)
	}
	if deserialized.Annotations["hot"] != "potato" {
		t.Fatalf("unexpected annotation in manifest: %s", deserialized.Annotations["hot"])
	}

	target := deserialized.Target()
	if target.Digest != "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b" {
		t.Fatalf("unexpected digest in target: %s", target.Digest.String())
	}
	if target.MediaType != v1.MediaTypeImageConfig {
		t.Fatalf("unexpected media type in target: %s", target.MediaType)
	}
	if target.Size != 985 {
		t.Fatalf("unexpected size in target: %d", target.Size)
	}
	if target.Annotations["apple"] != "orange" {
		t.Fatalf("unexpected annotation in target: %s", target.Annotations["apple"])
	}

	references := deserialized.References()
	if len(references) != 2 {
		t.Fatalf("unexpected number of references: %d", len(references))
	}

	if !reflect.DeepEqual(references[0], target) {
		t.Fatalf("first reference should be target: %v != %v", references[0], target)
	}

	// Test the second reference
	if references[1].Digest != "sha256:62d8908bee94c202b2d35224a221aaa2058318bfa9879fa541efaecba272331b" {
		t.Fatalf("unexpected digest in reference: %s", references[0].Digest.String())
	}
	if references[1].MediaType != v1.MediaTypeImageLayerGzip {
		t.Fatalf("unexpected media type in reference: %s", references[0].MediaType)
	}
	if references[1].Size != 153263 {
		t.Fatalf("unexpected size in reference: %d", references[0].Size)
	}
	if references[1].Annotations["lettuce"] != "wrap" {
		t.Fatalf("unexpected annotation in reference: %s", references[1].Annotations["lettuce"])
	}
}

func mediaTypeTest(t *testing.T, mediaType string, shouldError bool) {
	manifest := makeTestManifest(mediaType)

	deserialized, err := FromStruct(manifest)
	if err != nil {
		t.Fatalf("error creating DeserializedManifest: %v", err)
	}

	unmarshalled, descriptor, err := distribution.UnmarshalManifest(
		v1.MediaTypeImageManifest,
		deserialized.canonical)

	if shouldError {
		if err == nil {
			t.Fatalf("bad content type should have produced error")
		}
	} else {
		if err != nil {
			t.Fatalf("error unmarshaling manifest, %v", err)
		}

		asManifest := unmarshalled.(*DeserializedManifest)
		if asManifest.MediaType != mediaType {
			t.Fatalf("Bad media type '%v' as unmarshalled", asManifest.MediaType)
		}

		if descriptor.MediaType != v1.MediaTypeImageManifest {
			t.Fatalf("Bad media type '%v' for descriptor", descriptor.MediaType)
		}

		unmarshalledMediaType, _, _ := unmarshalled.Payload()
		if unmarshalledMediaType != v1.MediaTypeImageManifest {
			t.Fatalf("Bad media type '%v' for payload", unmarshalledMediaType)
		}
	}
}

func TestMediaTypes(t *testing.T) {
	mediaTypeTest(t, "", false)
	mediaTypeTest(t, v1.MediaTypeImageManifest, false)
	mediaTypeTest(t, v1.MediaTypeImageManifest+"XXX", true)
}
