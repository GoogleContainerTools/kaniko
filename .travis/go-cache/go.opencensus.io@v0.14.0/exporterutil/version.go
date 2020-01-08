// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package exporterutil contains common utilities for exporter implementations.
//
// Deprecated: Don't use this package.
package exporterutil

import opencensus "go.opencensus.io"

// Version is the current release version of OpenCensus in use. It is made
// available for exporters to include in User-Agent-like metadata.
var Version = opencensus.Version()

// TODO(jbd): Remove this package at the next release.
