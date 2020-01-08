// Copyright 2018 Google LLC All Rights Reserved.
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

package gcrane

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() { Root.AddCommand(NewCmdCopy()) }

// NewCmdCopy creates a new cobra.Command for the copy subcommand.
func NewCmdCopy() *cobra.Command {
	recursive := false
	cmd := &cobra.Command{
		Use:     "copy",
		Aliases: []string{"cp"},
		Short:   "Efficiently copy a remote image from src to dst",
		Args:    cobra.ExactArgs(2),
		Run: func(cc *cobra.Command, args []string) {
			doCopy(args, recursive)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Whether to recurse through repos")

	return cmd
}

func doCopy(args []string, recursive bool) {
	src, dst := args[0], args[1]

	if recursive {
		if err := recursiveCopy(src, dst); err != nil {
			log.Fatalf("failed to copy images: %v", err)
		}
	} else {
		srcAuth, dstAuth, err := parseRefAuths(src, dst)
		if err != nil {
			log.Fatal(err)
		}

		// First, try to copy as an index.
		// If that fails, try to copy as an image.
		// We have to try this second because fallback logic exists in the registry
		// to convert an index to an image.
		//
		// TODO(#407): Refactor crane so we can just call into that logic in the
		// single-image case.
		if err := copyIndex(src, dst, srcAuth, dstAuth); err != nil {
			if err := copyImage(src, dst, srcAuth, dstAuth); err != nil {
				log.Fatalf("failed to copy image: %v", err)
			}
		}
	}
}

type copier struct {
	srcRepo name.Repository
	dstRepo name.Repository

	srcAuth authn.Authenticator
	dstAuth authn.Authenticator
}

func newCopier(src, dst string) (*copier, error) {
	srcRepo, err := name.NewRepository(src)
	if err != nil {
		return nil, fmt.Errorf("parsing repo %q: %v", src, err)
	}

	dstRepo, err := name.NewRepository(dst)
	if err != nil {
		return nil, fmt.Errorf("parsing repo %q: %v", dst, err)
	}

	srcAuth, err := google.Keychain.Resolve(srcRepo.Registry)
	if err != nil {
		return nil, fmt.Errorf("getting auth for %q: %v", src, err)
	}

	dstAuth, err := google.Keychain.Resolve(dstRepo.Registry)
	if err != nil {
		return nil, fmt.Errorf("getting auth for %q: %v", dst, err)
	}

	return &copier{srcRepo, dstRepo, srcAuth, dstAuth}, nil
}

func copyImage(src, dst string, srcAuth, dstAuth authn.Authenticator) error {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", src, err)
	}

	dstRef, err := name.ParseReference(dst)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", dst, err)
	}

	img, err := remote.Image(srcRef, remote.WithAuth(srcAuth))
	if err != nil {
		return fmt.Errorf("reading image %q: %v", src, err)
	}

	if err := remote.Write(dstRef, img, remote.WithAuth(dstAuth)); err != nil {
		return fmt.Errorf("writing image %q: %v", dst, err)
	}

	return nil
}

func copyIndex(src, dst string, srcAuth, dstAuth authn.Authenticator) error {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", src, err)
	}

	dstRef, err := name.ParseReference(dst)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", dst, err)
	}

	idx, err := remote.Index(srcRef, remote.WithAuth(srcAuth))
	if err != nil {
		return fmt.Errorf("reading image %q: %v", src, err)
	}

	if err := remote.WriteIndex(dstRef, idx, remote.WithAuth(dstAuth)); err != nil {
		return fmt.Errorf("writing image %q: %v", dst, err)
	}

	return nil
}

// recursiveCopy copies images from repo src to repo dst, rather quickly. tl;dr:
//
//  for each repo in src {
//		go func {
//			for each image in repo {
//				go func {
//					for each tag in image {
//						go func {
//							copyImage(tag, rename(tag, dst))
//						}
//					}
//				}
//			}
//			for each index in repo {
//				go func {
//					for each tag in index {
//						go func {
//							copyIndex(tag, rename(tag, dst))
//						}
//					}
//				}
//			}
//		}
//	}
func recursiveCopy(src, dst string) error {
	c, err := newCopier(src, dst)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(context.Background())

	// Captures c, g, and ctx.
	walkFn := func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			return fmt.Errorf("failed walkFn for repo %s: %v", repo, err)
		}

		g.Go(func() error {
			return c.copyRepo(ctx, repo, tags)
		})

		return nil
	}

	if err := google.Walk(c.srcRepo, walkFn, google.WithAuth(c.srcAuth)); err != nil {
		return fmt.Errorf("failed to Walk: %v", err)
	}

	return g.Wait()
}

// copyRepo figures out the name for our destination repo (newRepo), lists the
// contents of newRepo, calculates the diff of what needs to be copied, then
// starts a goroutine to copy each image we need, and waits for them to finish.
func (c *copier) copyRepo(ctx context.Context, oldRepo name.Repository, tags *google.Tags) error {
	newRepo, err := c.rename(oldRepo)
	if err != nil {
		return fmt.Errorf("rename failed: %v", err)
	}

	// Figure out what we actually need to copy.
	want := tags.Manifests
	have := make(map[string]google.ManifestInfo)
	haveTags, err := google.List(newRepo, google.WithAuth(c.dstAuth))
	if err != nil {
		// Possibly, we could see a 404.  If we get an error here, log it and assume
		// we just need to copy everything.
		//
		// TODO: refactor remote.Error to expose response code?
		logs.Warn.Printf("failed to list %s: %v", newRepo, err)
	} else {
		have = haveTags.Manifests
	}
	need := diffImages(want, have)

	g, ctx := errgroup.WithContext(ctx)

	// First go through copying just manifests, skipping manifest lists, since
	// manifest lists might reference them.
	todos := make(map[string]google.ManifestInfo)
	for digest, manifest := range need {
		if manifest.MediaType == string(types.DockerManifestList) || manifest.MediaType == string(types.OCIImageIndex) {
			todos[digest] = manifest
			continue
		}

		digest, manifest := digest, manifest // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			return c.copyImages(ctx, digest, manifest, oldRepo, newRepo)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("Failed to copy %s: %v", oldRepo, err)
	}

	// Now copy the manifest lists, since it should be safe.
	for digest, manifest := range todos {
		digest, manifest := digest, manifest // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			return c.copyIndexes(ctx, digest, manifest, oldRepo, newRepo)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("Failed to copy %s: %v", oldRepo, err)
	}

	return nil
}

// copyImages starts a goroutine for each tag that points to the image
// oldRepo@digest, or just copies the image by digest if there are no tags.
func (c *copier) copyImages(ctx context.Context, digest string, manifest google.ManifestInfo, oldRepo, newRepo name.Repository) error {
	// We only have to explicitly copy by digest if there are no tags pointing to this manifest.
	if len(manifest.Tags) == 0 {
		srcImg := fmt.Sprintf("%s@%s", oldRepo, digest)
		dstImg := fmt.Sprintf("%s@%s", newRepo, digest)

		return copyImage(srcImg, dstImg, c.srcAuth, c.dstAuth)
	}

	// Copy all the tags.
	g, _ := errgroup.WithContext(ctx)
	for _, tag := range manifest.Tags {
		tag := tag // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			srcImg := fmt.Sprintf("%s:%s", oldRepo, tag)
			dstImg := fmt.Sprintf("%s:%s", newRepo, tag)

			return copyImage(srcImg, dstImg, c.srcAuth, c.dstAuth)
		})
	}
	return g.Wait()
}

// copyIndexes starts a goroutine for each tag that points to the index
// oldRepo@digest, or just copies the index by digest if there are no tags.
func (c *copier) copyIndexes(ctx context.Context, digest string, manifest google.ManifestInfo, oldRepo, newRepo name.Repository) error {
	// We only have to explicitly copy by digest if there are no tags pointing to this manifest.
	if len(manifest.Tags) == 0 {
		srcImg := fmt.Sprintf("%s@%s", oldRepo, digest)
		dstImg := fmt.Sprintf("%s@%s", newRepo, digest)

		return copyIndex(srcImg, dstImg, c.srcAuth, c.dstAuth)
	}

	// Copy all the tags.
	g, _ := errgroup.WithContext(ctx)
	for _, tag := range manifest.Tags {
		tag := tag // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			srcImg := fmt.Sprintf("%s:%s", oldRepo, tag)
			dstImg := fmt.Sprintf("%s:%s", newRepo, tag)

			// TODO: Just implement an AddTag thing.
			return copyIndex(srcImg, dstImg, c.srcAuth, c.dstAuth)
		})
	}
	return g.Wait()
}

// rename figures out the name of the new repository to copy to, e.g.:
//
// $ gcrane cp -r gcr.io/foo gcr.io/baz
//
// rename("gcr.io/foo/bar") == "gcr.io/baz/bar"
func (c *copier) rename(repo name.Repository) (name.Repository, error) {
	replaced := strings.Replace(repo.String(), c.srcRepo.String(), c.dstRepo.String(), 1)
	return name.NewRepository(replaced, name.StrictValidation)
}

// diffImages returns a map of digests to google.ManifestInfos for images or
// tags that are present in "want" but not in "have".
func diffImages(want, have map[string]google.ManifestInfo) map[string]google.ManifestInfo {
	need := make(map[string]google.ManifestInfo)

	for digest, wantManifest := range want {
		if haveManifest, ok := have[digest]; !ok {
			// Missing the whole image, we need to copy everything.
			need[digest] = wantManifest
		} else {
			missingTags := subtractStringLists(wantManifest.Tags, haveManifest.Tags)
			if len(missingTags) == 0 {
				continue
			}

			// Missing just some tags, add the ones we need to copy.
			todo := wantManifest
			todo.Tags = missingTags
			need[digest] = todo
		}
	}

	return need
}

// subtractStringLists returns a list of strings that are in minuend and not
// in subtrahend; order is unimportant.
func subtractStringLists(minuend, subtrahend []string) []string {
	bSet := toStringSet(subtrahend)
	difference := []string{}

	for _, a := range minuend {
		if _, ok := bSet[a]; !ok {
			difference = append(difference, a)
		}
	}

	return difference
}

func toStringSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	return set
}

func parseRefAuths(src, dst string) (authn.Authenticator, authn.Authenticator, error) {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference %q: %v", src, err)
	}

	dstRef, err := name.ParseReference(dst)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference %q: %v", dst, err)
	}

	srcAuth, err := authn.DefaultKeychain.Resolve(srcRef.Context().Registry)
	if err != nil {
		return nil, nil, fmt.Errorf("getting auth for %q: %v", src, err)
	}

	dstAuth, err := authn.DefaultKeychain.Resolve(dstRef.Context().Registry)
	if err != nil {
		return nil, nil, fmt.Errorf("getting auth for %q: %v", dst, err)
	}

	return srcAuth, dstAuth, nil
}
