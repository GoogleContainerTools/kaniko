package events

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type tOrB interface {
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

type testSink struct {
	t tOrB

	events   []Event
	expected int
	mu       sync.Mutex
	closed   bool
}

func newTestSink(t tOrB, expected int) *testSink {
	return &testSink{
		t:        t,
		events:   make([]Event, 0, expected), // pre-allocate so we aren't benching alloc
		expected: expected,
	}
}

func (ts *testSink) Write(event Event) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return ErrSinkClosed
	}

	ts.events = append(ts.events, event)

	if len(ts.events) > ts.expected {
		ts.t.Fatalf("len(ts.events) == %v, expected %v", len(ts.events), ts.expected)
	}

	return nil
}

func (ts *testSink) Close() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.closed {
		return ErrSinkClosed
	}

	ts.closed = true

	if len(ts.events) != ts.expected {
		ts.t.Fatalf("len(ts.events) == %v, expected %v", len(ts.events), ts.expected)
	}

	return nil
}

type delayedSink struct {
	Sink
	delay time.Duration
}

func (ds *delayedSink) Write(event Event) error {
	time.Sleep(ds.delay)
	return ds.Sink.Write(event)
}

type flakySink struct {
	Sink
	rate float64
	mu   sync.Mutex
}

func (fs *flakySink) Write(event Event) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if rand.Float64() < fs.rate {
		return fmt.Errorf("error writing event: %v", event)
	}

	return fs.Sink.Write(event)
}

func checkClose(t *testing.T, sink Sink) {
	if err := sink.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	// second close should not crash but should return an error.
	if err := sink.Close(); err != nil {
		t.Fatalf("unexpected error on double close: %v", err)
	}

	// Write after closed should be an error
	if err := sink.Write("fail"); err == nil {
		t.Fatalf("write after closed did not have an error")
	} else if err != ErrSinkClosed {
		t.Fatalf("error should be ErrSinkClosed")
	}
}

func benchmarkSink(b *testing.B, sink Sink) {
	defer sink.Close()
	var event = "myevent"
	for i := 0; i < b.N; i++ {
		sink.Write(event)
	}
}
