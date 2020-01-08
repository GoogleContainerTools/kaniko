package queue

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/docker/go-events"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockSink struct {
	closed   bool
	holdChan chan struct{}
	data     []events.Event
	mutex    sync.Mutex
	once     sync.Once
}

func (s *mockSink) Write(event events.Event) error {
	<-s.holdChan

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return events.ErrSinkClosed
	}
	s.data = append(s.data, event)
	return nil
}

func (s *mockSink) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.once.Do(func() {
		s.closed = true
		close(s.holdChan)
	})
	return nil
}

func (s *mockSink) Len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.data)
}

func (s *mockSink) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return fmt.Sprintf("%v", s.data)
}

func TestLimitQueueNoLimit(t *testing.T) {
	require := require.New(t)
	ch := make(chan struct{})
	ms := &mockSink{
		holdChan: ch,
	}

	// Create a limit queue with no limit and store 10k events. The events
	// should be held in the queue until we unblock the sink.
	q := NewLimitQueue(ms, 0)
	defer q.Close()
	defer ms.Close()

	// Writing one event to the queue should block during the sink write phase
	require.NoError(q.Write("test event"))

	// Make sure the consumer goroutine receives the event
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && q.Len() != 0 {
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(0, q.Len())
	require.Equal(0, ms.Len())

	for i := 0; i < 9999; i++ {
		require.NoError(q.Write("test event"))
	}
	require.Equal(9999, q.Len()) // 1 event blocked in the sink, 9999 waiting in the queue
	require.Equal(0, ms.Len())

	// Unblock the sink and expect all the events to have been flushed out of
	// the queue.
	for i := 0; i < 10000; i++ {
		ch <- struct{}{}
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && ms.Len() != 10000 {
		time.Sleep(20 * time.Millisecond)
	}

	require.Equal(0, q.Len())
	require.Equal(10000, ms.Len())
}

// TestLimitQueueWithLimit ensures that the limit queue works with a limit.
func TestLimitQueueWithLimit(t *testing.T) {
	require := require.New(t)
	ch := make(chan struct{})
	ms := &mockSink{
		holdChan: ch,
	}

	// Create a limit queue with no limit and store 10k events. The events should be held in
	// the queue until we unblock the sink.
	q := NewLimitQueue(ms, 10)
	defer q.Close()
	defer ms.Close()

	// Write the first event and wait for it to block on the writer
	require.NoError(q.Write("test event"))
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && q.Len() != 0 {
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(0, ms.Len())
	require.Equal(0, q.Len())

	// Fill up the queue
	for i := 0; i < 10; i++ {
		require.NoError(q.Write("test event"))
	}
	require.Equal(0, ms.Len())
	require.Equal(10, q.Len())

	// Reading one event by the sink should allow us to write one more back
	// without closing the queue.
	ch <- struct{}{}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && q.Len() != 9 {
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(9, q.Len())
	require.Equal(1, ms.Len())
	require.NoError(q.Write("test event"))
	require.Equal(10, q.Len())
	require.Equal(1, ms.Len())

	// Trying to write a new event in the queue should flush it
	logrus.Debugf("Closing queue")
	err := q.Write("test event")
	require.Error(err)
	require.Equal(ErrQueueFull, err)
	require.Equal(10, q.Len())
	require.Equal(1, ms.Len())

	// Further writes should return the same error
	err = q.Write("test event")
	require.Error(err)
	require.Equal(ErrQueueFull, err)
	require.Equal(10, q.Len())
	require.Equal(1, ms.Len())

	// Reading one event from the sink will allow one more write to go through again
	ch <- struct{}{}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && q.Len() != 9 {
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(9, q.Len())
	require.Equal(2, ms.Len())
	require.NoError(q.Write("test event"))
	require.Equal(10, q.Len())
	require.Equal(2, ms.Len())

	err = q.Write("test event")
	require.Error(err)
	require.Equal(ErrQueueFull, err)
	require.Equal(10, q.Len())
	require.Equal(2, ms.Len())
}
