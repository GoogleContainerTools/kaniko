// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"golang.org/x/sync/errgroup"
)

// MultiWrite writes the given Images or ImageIndexes to the given refs, as
// efficiently as possible, by deduping shared layer blobs and uploading layers
// in parallel, then uploading all manifests in parallel.
//
// Current limitations:
// - All refs must share the same repository.
// - Images cannot consist of stream.Layers.
func MultiWrite(m map[name.Reference]Taggable, options ...Option) error {
	// Determine the repository being pushed to; if asked to push to
	// multiple repositories, give up.
	var repo, zero name.Repository
	for ref := range m {
		if repo == zero {
			repo = ref.Context()
		} else if ref.Context() != repo {
			return fmt.Errorf("MultiWrite can only push to the same repository (saw %q and %q)", repo, ref.Context())
		}
	}

	// Collect unique blobs (layers and config blobs).
	blobs := map[v1.Hash]v1.Layer{}
	newManifests := []map[name.Reference]Taggable{}
	// Separate originally requested images and indexes, so we can push images first.
	images, indexes := map[name.Reference]Taggable{}, map[name.Reference]Taggable{}
	var err error
	for ref, i := range m {
		if img, ok := i.(v1.Image); ok {
			images[ref] = i
			if err := addImageBlobs(img, blobs); err != nil {
				return err
			}
			continue
		}
		if idx, ok := i.(v1.ImageIndex); ok {
			indexes[ref] = i
			newManifests, err = addIndexBlobs(idx, blobs, repo, newManifests, 0)
			if err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("pushable resource was not Image or ImageIndex: %T", i)
	}

	o, err := makeOptions(repo, options...)
	if err != nil {
		return err
	}
	// Determine if any of the layers are Mountable, because if so we need
	// to request Pull scope too.
	ls := []v1.Layer{}
	for _, l := range blobs {
		ls = append(ls, l)
	}
	scopes := scopesForUploadingImage(repo, ls)
	tr, err := transport.NewWithContext(o.context, repo.Registry, o.auth, o.transport, scopes)
	if err != nil {
		return err
	}
	w := writer{
		repo:    repo,
		client:  &http.Client{Transport: tr},
		context: o.context,
	}

	// Upload individual blobs and collect any errors.
	blobChan := make(chan v1.Layer, 2*o.jobs)
	var g errgroup.Group
	for i := 0; i < o.jobs; i++ {
		// Start N workers consuming blobs to upload.
		g.Go(func() error {
			for b := range blobChan {
				if err := w.uploadOne(b); err != nil {
					return err
				}
			}
			return nil
		})
	}
	go func() {
		for _, b := range blobs {
			blobChan <- b
		}
		close(blobChan)
	}()
	if err := g.Wait(); err != nil {
		return err
	}

	commitMany := func(m map[name.Reference]Taggable) error {
		// With all of the constituent elements uploaded, upload the manifests
		// to commit the images and indexes, and collect any errors.
		type task struct {
			i   Taggable
			ref name.Reference
		}
		taskChan := make(chan task, 2*o.jobs)
		for i := 0; i < o.jobs; i++ {
			// Start N workers consuming tasks to upload manifests.
			g.Go(func() error {
				for t := range taskChan {
					if err := w.commitManifest(t.i, t.ref); err != nil {
						return err
					}
				}
				return nil
			})
		}
		go func() {
			for ref, i := range m {
				taskChan <- task{i, ref}
			}
			close(taskChan)
		}()
		return g.Wait()
	}
	// Push originally requested image manifests. These have no
	// dependencies.
	if err := commitMany(images); err != nil {
		return err
	}
	// Push new manifests from lowest levels up.
	for i := len(newManifests) - 1; i >= 0; i-- {
		if err := commitMany(newManifests[i]); err != nil {
			return err
		}
	}
	// Push originally requested index manifests, which might depend on
	// newly discovered manifests.
	return commitMany(indexes)

}

// addIndexBlobs adds blobs to the set of blobs we intend to upload, and
// returns the latest copy of the ordered collection of manifests to upload.
func addIndexBlobs(idx v1.ImageIndex, blobs map[v1.Hash]v1.Layer, repo name.Repository, newManifests []map[name.Reference]Taggable, lvl int) ([]map[name.Reference]Taggable, error) {
	if lvl > len(newManifests)-1 {
		newManifests = append(newManifests, map[name.Reference]Taggable{})
	}

	im, err := idx.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, desc := range im.Manifests {
		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			idx, err := idx.ImageIndex(desc.Digest)
			if err != nil {
				return nil, err
			}
			newManifests, err = addIndexBlobs(idx, blobs, repo, newManifests, lvl+1)
			if err != nil {
				return nil, err
			}

			// Also track the sub-index manifest to upload later by digest.
			newManifests[lvl][repo.Digest(desc.Digest.String())] = idx
		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			img, err := idx.Image(desc.Digest)
			if err != nil {
				return nil, err
			}
			if err := addImageBlobs(img, blobs); err != nil {
				return nil, err
			}

			// Also track the sub-image manifest to upload later by digest.
			newManifests[lvl][repo.Digest(desc.Digest.String())] = img
		default:
			return nil, fmt.Errorf("unknown media type: %v", desc.MediaType)
		}
	}
	return newManifests, nil
}

func addImageBlobs(img v1.Image, blobs map[v1.Hash]v1.Layer) error {
	ls, err := img.Layers()
	if err != nil {
		return err
	}
	// Collect all layers.
	for _, l := range ls {
		d, err := l.Digest()
		if err != nil {
			return err
		}

		// Ignore foreign layers.
		mt, err := l.MediaType()
		if err != nil {
			return err
		}
		if !mt.IsDistributable() {
			// TODO(jonjohnsonjr): Add "allow-nondistributable-artifacts" option.
			continue
		}

		blobs[d] = l
	}

	// Collect config blob.
	cl, err := partial.ConfigLayer(img)
	if err != nil {
		return err
	}
	cld, err := cl.Digest()
	if err != nil {
		return err
	}
	blobs[cld] = cl
	return nil
}
