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

package tracecontext

import (
	"net/http"
	"reflect"
	"testing"

	"go.opencensus.io/trace"
)

func TestHTTPFormat_FromRequest(t *testing.T) {
	tests := []struct {
		name   string
		header string
		wantSc trace.SpanContext
		wantOk bool
	}{
		{
			name:   "unsupported version",
			header: "02-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantSc: trace.SpanContext{},
			wantOk: false,
		},
		{
			name:   "zero trace ID and span ID",
			header: "00-00000000000000000000000000000000-0000000000000000-01",
			wantSc: trace.SpanContext{},
			wantOk: false,
		},
		{
			name:   "valid header",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantSc: trace.SpanContext{
				TraceID:      trace.TraceID{75, 249, 47, 53, 119, 179, 77, 166, 163, 206, 146, 157, 14, 14, 71, 54},
				SpanID:       trace.SpanID{0, 240, 103, 170, 11, 169, 2, 183},
				TraceOptions: trace.TraceOptions(1),
			},
			wantOk: true,
		},
		{
			name:   "no options",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7",
			wantSc: trace.SpanContext{
				TraceID:      trace.TraceID{75, 249, 47, 53, 119, 179, 77, 166, 163, 206, 146, 157, 14, 14, 71, 54},
				SpanID:       trace.SpanID{0, 240, 103, 170, 11, 169, 2, 183},
				TraceOptions: trace.TraceOptions(0),
			},
			wantOk: true,
		},
		{
			name:   "missing options",
			header: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-",
			wantSc: trace.SpanContext{},
			wantOk: false,
		},
	}

	f := &HTTPFormat{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.Header.Set("Trace-Parent", tt.header)

			gotSc, gotOk := f.SpanContextFromRequest(req)
			if !reflect.DeepEqual(gotSc, tt.wantSc) {
				t.Errorf("HTTPFormat.FromRequest() gotSc = %v, want %v", gotSc, tt.wantSc)
			}
			if gotOk != tt.wantOk {
				t.Errorf("HTTPFormat.FromRequest() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestHTTPFormat_ToRequest(t *testing.T) {
	tests := []struct {
		sc         trace.SpanContext
		wantHeader string
	}{
		{
			sc: trace.SpanContext{
				TraceID:      trace.TraceID{75, 249, 47, 53, 119, 179, 77, 166, 163, 206, 146, 157, 14, 14, 71, 54},
				SpanID:       trace.SpanID{0, 240, 103, 170, 11, 169, 2, 183},
				TraceOptions: trace.TraceOptions(1),
			},
			wantHeader: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.wantHeader, func(t *testing.T) {
			f := &HTTPFormat{}
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			f.SpanContextToRequest(tt.sc, req)

			h := req.Header.Get("Trace-Parent")
			if got, want := h, tt.wantHeader; got != want {
				t.Errorf("HTTPFormat.ToRequest() header = %v, want %v", got, want)
			}
		})
	}
}
