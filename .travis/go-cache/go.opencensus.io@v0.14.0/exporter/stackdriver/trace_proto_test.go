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
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"go.opencensus.io/internal"

	"github.com/golang/protobuf/proto"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	"go.opencensus.io/trace"
	tracepb "google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	codepb "google.golang.org/genproto/googleapis/rpc/code"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
)

const projectID = "testproject"

var (
	traceID = trace.TraceID{0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f}
	spanID  = trace.SpanID{0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1}
)

type spans []*tracepb.Span

func (s spans) Len() int           { return len(s) }
func (s spans) Less(x, y int) bool { return s[x].DisplayName.Value < s[y].DisplayName.Value }
func (s spans) Swap(x, y int)      { s[x], s[y] = s[y], s[x] }

type testExporter struct {
	spans []*trace.SpanData
}

func (t *testExporter) ExportSpan(s *trace.SpanData) {
	t.spans = append(t.spans, s)
}

func TestExportTrace(t *testing.T) {
	ctx := context.Background()

	var te testExporter
	trace.RegisterExporter(&te)
	defer trace.UnregisterExporter(&te)

	ctx, span0 := trace.StartSpanWithRemoteParent(
		ctx,
		"span0",
		trace.SpanContext{
			TraceID:      traceID,
			SpanID:       spanID,
			TraceOptions: 1,
		},
	)
	{
		ctx1, span1 := trace.StartSpan(ctx, "span1")
		{
			_, span2 := trace.StartSpan(ctx1, "span2")
			span2.AddMessageSendEvent(0x123, 1024, 512)
			span2.Annotatef(nil, "in span%d", 2)
			span2.Annotate(nil, big.NewRat(2, 4).String())
			span2.AddAttributes(
				trace.StringAttribute("key1", "value1"),
				trace.StringAttribute("key2", "value2"))
			span2.AddAttributes(trace.Int64Attribute("key1", 100))
			span2.End()
		}
		{
			ctx3, span3 := trace.StartSpan(ctx1, "span3")
			span3.Annotate(nil, "in span3")
			span3.AddMessageReceiveEvent(0x456, 2048, 1536)
			span3.SetStatus(trace.Status{Code: int32(codepb.Code_UNAVAILABLE)})
			span3.End()
			{
				_, span4 := trace.StartSpan(ctx3, "span4")
				x := 42
				a1 := []trace.Attribute{trace.StringAttribute("k1", "v1")}
				a2 := []trace.Attribute{trace.StringAttribute("k2", "v2")}
				a3 := []trace.Attribute{trace.StringAttribute("k3", "v3")}
				a4 := map[string]interface{}{"k4": "v4"}
				r := big.NewRat(2, 4)
				span4.Annotate(a1, r.String())
				span4.Annotatef(a2, "foo %d", x)
				span4.Annotate(a3, "in span4")
				span4.AddLink(trace.Link{TraceID: trace.TraceID{1, 2}, SpanID: trace.SpanID{3}, Type: trace.LinkTypeParent, Attributes: a4})
				span4.End()
			}
		}
		span1.End()
	}
	span0.End()
	if len(te.spans) != 5 {
		t.Errorf("got %d exported spans, want 5", len(te.spans))
	}

	var spbs spans
	for _, s := range te.spans {
		spbs = append(spbs, protoFromSpanData(s, "testproject"))
	}
	sort.Sort(spbs)

	for i, want := range []string{
		spanID.String(),
		spbs[0].SpanId,
		spbs[1].SpanId,
		spbs[1].SpanId,
		spbs[3].SpanId,
	} {
		if got := spbs[i].ParentSpanId; got != want {
			t.Errorf("span %d: got ParentSpanID %q want %q", i, got, want)
		}
	}
	checkTime := func(ts **timestamppb.Timestamp) {
		if *ts == nil {
			t.Error("expected timestamp")
		}
		*ts = nil
	}
	for _, span := range spbs {
		checkTime(&span.StartTime)
		checkTime(&span.EndTime)
		if span.TimeEvents != nil {
			for _, te := range span.TimeEvents.TimeEvent {
				checkTime(&te.Time)
			}
		}
		if want := fmt.Sprintf("projects/testproject/traces/%s/spans/%s", traceID, span.SpanId); span.Name != want {
			t.Errorf("got span name %q want %q", span.Name, want)
		}
		span.Name, span.SpanId, span.ParentSpanId = "", "", ""
	}

	expectedSpans := spans{
		&tracepb.Span{
			DisplayName:             trunc("span0", 128),
			SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: false},
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					agentLabel: {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc(internal.UserAgent, len(internal.UserAgent))}},
				},
			},
		},
		&tracepb.Span{
			DisplayName:             trunc("span1", 128),
			SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					agentLabel: {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc(internal.UserAgent, len(internal.UserAgent))}},
				},
			},
		},
		&tracepb.Span{
			DisplayName: trunc("span2", 128),
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"key2":     {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc("value2", 256)}},
					"key1":     {Value: &tracepb.AttributeValue_IntValue{IntValue: 100}},
					agentLabel: {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc(internal.UserAgent, len(internal.UserAgent))}},
				},
			},
			TimeEvents: &tracepb.Span_TimeEvents{
				TimeEvent: []*tracepb.Span_TimeEvent{
					{
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Description: trunc("in span2", 256),
							},
						},
					},
					{
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Description: trunc("1/2", 256),
							},
						},
					},
					{
						Value: &tracepb.Span_TimeEvent_MessageEvent_{
							MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
								Type: tracepb.Span_TimeEvent_MessageEvent_SENT,
								Id:   0x123,
								UncompressedSizeBytes: 1024,
								CompressedSizeBytes:   512,
							},
						},
					},
				},
			},
			SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
		},
		&tracepb.Span{
			DisplayName: trunc("span3", 128),
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					agentLabel: {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc(internal.UserAgent, len(internal.UserAgent))}},
				},
			},
			TimeEvents: &tracepb.Span_TimeEvents{
				TimeEvent: []*tracepb.Span_TimeEvent{
					{
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Description: trunc("in span3", 256),
							},
						},
					},
					{
						Value: &tracepb.Span_TimeEvent_MessageEvent_{
							MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
								Type: tracepb.Span_TimeEvent_MessageEvent_RECEIVED,
								Id:   0x456,
								UncompressedSizeBytes: 2048,
								CompressedSizeBytes:   1536,
							},
						},
					},
				},
			},
			Status: &statuspb.Status{
				Code: 14,
			},
			SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
		},
		&tracepb.Span{
			DisplayName: trunc("span4", 128),
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					agentLabel: {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc(internal.UserAgent, len(internal.UserAgent))}},
				},
			},
			TimeEvents: &tracepb.Span_TimeEvents{
				TimeEvent: []*tracepb.Span_TimeEvent{
					{
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Description: trunc("1/2", 256),
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"k1": {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc("v1", 256)}},
									},
								},
							},
						},
					},
					{
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Description: trunc("foo 42", 256),
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"k2": {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc("v2", 256)}},
									},
								},
							},
						},
					},
					{
						Value: &tracepb.Span_TimeEvent_Annotation_{
							Annotation: &tracepb.Span_TimeEvent_Annotation{
								Description: trunc("in span4", 256),
								Attributes: &tracepb.Span_Attributes{
									AttributeMap: map[string]*tracepb.AttributeValue{
										"k3": {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc("v3", 256)}},
									},
								},
							},
						},
					},
				},
			},
			Links: &tracepb.Span_Links{
				Link: []*tracepb.Span_Link{
					{
						TraceId: "01020000000000000000000000000000",
						SpanId:  "0300000000000000",
						Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
						Attributes: &tracepb.Span_Attributes{
							AttributeMap: map[string]*tracepb.AttributeValue{
								"k4": {Value: &tracepb.AttributeValue_StringValue{StringValue: trunc("v4", 256)}},
							},
						},
					},
				},
			},
			SameProcessAsParentSpan: &wrapperspb.BoolValue{Value: true},
		},
	}

	if !reflect.DeepEqual(spbs, expectedSpans) {
		var got, want []string
		for _, s := range spbs {
			got = append(got, proto.MarshalTextString(s))
		}
		for _, s := range expectedSpans {
			want = append(want, proto.MarshalTextString(s))
		}
		t.Errorf("got spans:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEnums(t *testing.T) {
	for _, test := range []struct {
		x trace.LinkType
		y tracepb.Span_Link_Type
	}{
		{trace.LinkTypeUnspecified, tracepb.Span_Link_TYPE_UNSPECIFIED},
		{trace.LinkTypeChild, tracepb.Span_Link_CHILD_LINKED_SPAN},
		{trace.LinkTypeParent, tracepb.Span_Link_PARENT_LINKED_SPAN},
	} {
		if test.x != trace.LinkType(test.y) {
			t.Errorf("got link type values %d and %d, want equal", test.x, test.y)
		}
	}

	for _, test := range []struct {
		x trace.MessageEventType
		y tracepb.Span_TimeEvent_MessageEvent_Type
	}{
		{trace.MessageEventTypeUnspecified, tracepb.Span_TimeEvent_MessageEvent_TYPE_UNSPECIFIED},
		{trace.MessageEventTypeSent, tracepb.Span_TimeEvent_MessageEvent_SENT},
		{trace.MessageEventTypeRecv, tracepb.Span_TimeEvent_MessageEvent_RECEIVED},
	} {
		if test.x != trace.MessageEventType(test.y) {
			t.Errorf("got network event type values %d and %d, want equal", test.x, test.y)
		}
	}
}

func BenchmarkProto(b *testing.B) {
	sd := &trace.SpanData{
		SpanContext: trace.SpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
		Name:       "foo",
		StartTime:  time.Now().Add(-time.Second),
		EndTime:    time.Now(),
		Attributes: map[string]interface{}{"foo": "bar"},
		Annotations: []trace.Annotation{
			{
				Time:       time.Now().Add(-time.Millisecond),
				Message:    "hello, world",
				Attributes: map[string]interface{}{"foo": "bar"},
			},
		},
		MessageEvents: []trace.MessageEvent{
			{
				Time:                 time.Now().Add(-time.Microsecond),
				EventType:            1,
				MessageID:            2,
				UncompressedByteSize: 4,
				CompressedByteSize:   3,
			},
		},
		Status: trace.Status{
			Code:    42,
			Message: "failed",
		},
		HasRemoteParent: true,
	}
	var x int
	for i := 0; i < b.N; i++ {
		s := protoFromSpanData(sd, `testproject`)
		x += len(s.Name)
	}
	if x == 0 {
		fmt.Println(x)
	}
}
