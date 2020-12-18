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

package v1util

import "github.com/google/go-containerregistry/pkg/v1/internal/gzip"

// GzipReadCloser reads uncompressed input data from the io.ReadCloser and
// returns an io.ReadCloser from which compressed data may be read.
// This uses gzip.BestSpeed for the compression level.
// TODO(#873): Remove this package.
// Deprecated: please move off of this API
var GzipReadCloser = gzip.ReadCloser

// GzipReadCloserLevel reads uncompressed input data from the io.ReadCloser and
// returns an io.ReadCloser from which compressed data may be read.
// Refer to compress/gzip for the level:
// https://golang.org/pkg/compress/gzip/#pkg-constants
// TODO(#873): Remove this package.
// Deprecated: please move off of this API
var GzipReadCloserLevel = gzip.ReadCloserLevel

// GunzipReadCloser reads compressed input data from the io.ReadCloser and
// returns an io.ReadCloser from which uncompessed data may be read.
// TODO(#873): Remove this package.
// Deprecated: please move off of this API
var GunzipReadCloser = gzip.UnzipReadCloser

// IsGzipped detects whether the input stream is compressed.
// TODO(#873): Remove this package.
// Deprecated: please move off of this API
var IsGzipped = gzip.Is
