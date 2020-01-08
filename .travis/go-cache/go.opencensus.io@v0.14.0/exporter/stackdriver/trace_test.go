// Copyright 2017, OpenCensus Authors
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

package stackdriver

import (
	"context"
	"testing"
	"time"

	"go.opencensus.io/trace"
)

func TestBundling(t *testing.T) {
	exporter := newTraceExporterWithClient(Options{
		ProjectID:            "fakeProjectID",
		BundleDelayThreshold: time.Second / 10,
		BundleCountThreshold: 10,
	}, nil)

	ch := make(chan []*trace.SpanData)
	exporter.uploadFn = func(spans []*trace.SpanData) {
		ch <- spans
	}
	trace.RegisterExporter(exporter)

	for i := 0; i < 35; i++ {
		_, span := trace.StartSpan(context.Background(), "span", trace.WithSampler(trace.AlwaysSample()))
		span.End()
	}

	// Read the first three bundles.
	<-ch
	<-ch
	<-ch

	// Test that the fourth bundle isn't sent early.
	select {
	case <-ch:
		t.Errorf("bundle sent too early")
	case <-time.After(time.Second / 20):
		<-ch
	}

	// Test that there aren't extra bundles.
	select {
	case <-ch:
		t.Errorf("too many bundles sent")
	case <-time.After(time.Second / 5):
	}
}
