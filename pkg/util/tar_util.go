/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"archive/tar"
	"io"
	"os"
)

// AddToTar adds the file i to tar w at path p
func AddToTar(p string, i os.FileInfo, w *tar.Writer) error {
	linkDst := ""
	if i.Mode()&os.ModeSymlink != 0 {
		var err error
		linkDst, err = os.Readlink(p)
		if err != nil {
			return err
		}
	}
	hdr, err := tar.FileInfoHeader(i, linkDst)
	if err != nil {
		return err
	}
	hdr.Name = p
	w.WriteHeader(hdr)
	if !i.Mode().IsRegular() {
		return nil
	}
	r, err := os.Open(p)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}
