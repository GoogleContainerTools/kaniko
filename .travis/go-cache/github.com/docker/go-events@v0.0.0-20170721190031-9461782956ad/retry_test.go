package events

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRetryingSinkBreaker(t *testing.T) {
	testRetryingSink(t, NewBreaker(3, 10*time.Millisecond))
}

func TestRetryingSinkExponentialBackoff(t *testing.T) {
	testRetryingSink(t, NewExponentialBackoff(ExponentialBackoffConfig{
		Base:   time.Millisecond,
		Factor: time.Millisecond,
		Max:    time.Millisecond * 5,
	}))
}

func testRetryingSink(t *testing.T, strategy RetryStrategy) {
	const nevents = 100
	ts := newTestSink(t, nevents)

	// Make a sync that fails most of the time, ensuring that all the events
	// make it through.
	flaky := &flakySink{
		rate: 1.0, // start out always failing.
		Sink: ts,
	}

	s := NewRetryingSink(flaky, strategy)

	var wg sync.WaitGroup
	for i := 1; i <= nevents; i++ {
		event := "myevent-" + fmt.Sprint(i)

		// Above 50, set the failure rate lower
		if i > 50 {
			flaky.mu.Lock()
			flaky.rate = 0.9
			flaky.mu.Unlock()
		}

		wg.Add(1)
		go func(event Event) {
			defer wg.Done()
			if err := s.Write(event); err != nil {
				t.Fatalf("error writing event: %v", err)
			}
		}(event)
	}

	wg.Wait()
	checkClose(t, s)

	ts.mu.Lock()
	defer ts.mu.Unlock()
}

func TestExponentialBackoff(t *testing.T) {
	strategy := NewExponentialBackoff(DefaultExponentialBackoffConfig)
	backoff := strategy.Proceed(nil)

	if backoff != 0 {
		t.Errorf("untouched backoff should be zero-wait: %v != 0", backoff)
	}

	expected := strategy.config.Base + strategy.config.Factor
	for i := 1; i <= 10; i++ {
		if strategy.Failure(nil, nil) {
			t.Errorf("no facilities for dropping events in ExponentialBackoff")
		}

		for j := 0; j < 1000; j++ {
			// sample this several thousand times.
			backoff := strategy.Proceed(nil)
			if backoff > expected {
				t.Fatalf("expected must be bounded by %v after %v failures: %v", expected, i, backoff)
			}
		}

		expected = strategy.config.Base + strategy.config.Factor*time.Duration(1<<uint64(i))
		if expected > strategy.config.Max {
			expected = strategy.config.Max
		}
	}

	strategy.Success(nil) // recovery!

	backoff = strategy.Proceed(nil)
	if backoff != 0 {
		t.Errorf("should have recovered: %v != 0", backoff)
	}
}
