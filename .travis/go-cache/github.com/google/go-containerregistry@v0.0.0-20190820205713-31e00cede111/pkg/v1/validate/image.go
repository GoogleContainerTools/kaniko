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

package validate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// Image validates that img does not violate any invariants of the image format.
func Image(img v1.Image) error {
	errs := []string{}
	if err := validateLayers(img); err != nil {
		errs = append(errs, fmt.Sprintf("validating layers: %v", err))
	}

	if err := validateConfig(img); err != nil {
		errs = append(errs, fmt.Sprintf("validating config: %v", err))
	}

	if err := validateManifest(img); err != nil {
		errs = append(errs, fmt.Sprintf("validating manifest: %v", err))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n\n"))
	}
	return nil
}

func validateConfig(img v1.Image) error {
	cn, err := img.ConfigName()
	if err != nil {
		return err
	}

	rc, err := img.RawConfigFile()
	if err != nil {
		return err
	}

	hash, size, err := v1.SHA256(bytes.NewReader(rc))
	if err != nil {
		return err
	}

	m, err := img.Manifest()
	if err != nil {
		return err
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return err
	}

	pcf, err := v1.ParseConfigFile(bytes.NewReader(rc))
	if err != nil {
		return err
	}

	errs := []string{}
	if cn != hash {
		errs = append(errs, fmt.Sprintf("mismatched config digest: ConfigName()=%s, SHA256(RawConfigFile())=%s", cn, hash))
	}

	if want, got := m.Config.Size, size; want != got {
		errs = append(errs, fmt.Sprintf("mismatched config size: Manifest.Config.Size()=%d, len(RawConfigFile())=%d", want, got))
	}

	if diff := cmp.Diff(pcf, cf); diff != "" {
		errs = append(errs, fmt.Sprintf("mismatched config content: (-ParseConfigFile(RawConfigFile()) +ConfigFile()) %s", diff))
	}

	if cf.RootFS.Type != "layers" {
		errs = append(errs, fmt.Sprintf("invalid ConfigFile.RootFS.Type: %q != %q", cf.RootFS.Type, "layers"))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func validateLayers(img v1.Image) error {
	layers, err := img.Layers()
	if err != nil {
		return err
	}

	digests := []v1.Hash{}
	diffids := []v1.Hash{}
	sizes := []int64{}
	for _, layer := range layers {
		// TODO: Test layer.Uncompressed.
		compressed, err := layer.Compressed()
		if err != nil {
			return err
		}

		// Keep track of compressed digest.
		digester := sha256.New()
		// Everything read from compressed is written to digester to compute digest.
		hashCompressed := io.TeeReader(compressed, digester)

		// Call io.Copy to write from the layer Reader through to the tarReader on
		// the other side of the pipe.
		pr, pw := io.Pipe()
		var size int64
		go func() {
			n, err := io.Copy(pw, hashCompressed)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			size = n

			// Now close the compressed reader, to flush the gzip stream
			// and calculate digest/diffID/size. This will cause pr to
			// return EOF which will cause readers of the Compressed stream
			// to finish reading.
			pw.CloseWithError(compressed.Close())
		}()

		// Read the bytes through gzip.Reader to compute the DiffID.
		uncompressed, err := gzip.NewReader(pr)
		if err != nil {
			return err
		}
		diffider := sha256.New()
		hashUncompressed := io.TeeReader(uncompressed, diffider)

		// Ensure there aren't duplicate file paths.
		tarReader := tar.NewReader(hashUncompressed)
		files := make(map[string]struct{})
		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if _, ok := files[hdr.Name]; ok {
				return fmt.Errorf("duplicate file path: %s", hdr.Name)
			}
			files[hdr.Name] = struct{}{}
		}

		// Discard any trailing padding that the tar.Reader doesn't consume.
		if _, err := io.Copy(ioutil.Discard, hashUncompressed); err != nil {
			return err
		}

		if err := uncompressed.Close(); err != nil {
			return err
		}

		digest := v1.Hash{
			Algorithm: "sha256",
			Hex:       hex.EncodeToString(digester.Sum(make([]byte, 0, digester.Size()))),
		}

		diffid := v1.Hash{
			Algorithm: "sha256",
			Hex:       hex.EncodeToString(diffider.Sum(make([]byte, 0, diffider.Size()))),
		}

		// Compute all of these first before we call Config() and Manifest() to allow
		// for lazy access e.g. for stream.Layer.
		digests = append(digests, digest)
		diffids = append(diffids, diffid)
		sizes = append(sizes, size)
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return err
	}

	m, err := img.Manifest()
	if err != nil {
		return err
	}

	errs := []string{}
	for i, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			return err
		}
		diffid, err := layer.DiffID()
		if err != nil {
			return err
		}
		size, err := layer.Size()
		if err != nil {
			return err
		}

		if digest != digests[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] digest: Digest()=%s, SHA256(Compressed())=%s", i, digest, digests[i]))
		}

		if m.Layers[i].Digest != digests[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] digest: Manifest.Layers[%d].Digest=%s, SHA256(Compressed())=%s", i, i, m.Layers[i].Digest, digests[i]))
		}

		if diffid != diffids[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] diffid: DiffID()=%s, SHA256(Gunzip(Compressed()))=%s", i, diffid, diffids[i]))
		}

		if cf.RootFS.DiffIDs[i] != diffids[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] diffid: ConfigFile.RootFS.DiffIDs[%d]=%s, SHA256(Gunzip(Compressed()))=%s", i, i, cf.RootFS.DiffIDs[i], diffids[i]))
		}

		if size != sizes[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] size: Size()=%d, len(Compressed())=%d", i, size, sizes[i]))
		}

		if m.Layers[i].Size != sizes[i] {
			errs = append(errs, fmt.Sprintf("mismatched layer[%d] size: Manifest.Layers[%d].Size=%d, len(Compressed())=%d", i, i, m.Layers[i].Size, sizes[i]))
		}

	}
	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func validateManifest(img v1.Image) error {
	digest, err := img.Digest()
	if err != nil {
		return err
	}

	rm, err := img.RawManifest()
	if err != nil {
		return err
	}

	hash, _, err := v1.SHA256(bytes.NewReader(rm))
	if err != nil {
		return err
	}

	m, err := img.Manifest()
	if err != nil {
		return err
	}

	pm, err := v1.ParseManifest(bytes.NewReader(rm))
	if err != nil {
		return err
	}

	errs := []string{}
	if digest != hash {
		errs = append(errs, fmt.Sprintf("mismatched manifest digest: Digest()=%s, SHA256(RawManifest())=%s", digest, hash))
	}

	if diff := cmp.Diff(pm, m); diff != "" {
		errs = append(errs, fmt.Sprintf("mismatched manifest content: (-ParseManifest(RawManifest()) +Manifest()) %s", diff))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}
