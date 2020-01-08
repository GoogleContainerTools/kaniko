package events

import (
	"fmt"
	"sync"
	"testing"
)

func TestChannel(t *testing.T) {
	const nevents = 100

	sink := NewChannel(0)

	go func() {
		var wg sync.WaitGroup
		for i := 1; i <= nevents; i++ {
			event := "event-" + fmt.Sprint(i)
			wg.Add(1)
			go func(event Event) {
				defer wg.Done()
				if err := sink.Write(event); err != nil {
					t.Fatalf("error writing event: %v", err)
				}
			}(event)
		}
		wg.Wait()
		sink.Close()

		// now send another bunch of events and ensure we stay closed
		for i := 1; i <= nevents; i++ {
			if err := sink.Write(i); err != ErrSinkClosed {
				t.Fatalf("unexpected error: %v != %v", err, ErrSinkClosed)
			}
		}
	}()

	var received int
loop:
	for {
		select {
		case <-sink.C:
			received++
		case <-sink.Done():
			break loop
		}
	}

	sink.Close()
	_, ok := <-sink.Done() // test will timeout if this hangs
	if ok {
		t.Fatalf("done should be a closed channel")
	}

	if received != nevents {
		t.Fatalf("events did not make it through sink: %v != %v", received, nevents)
	}
}
