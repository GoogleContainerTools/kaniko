package watch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestTimeoutDropErrSinkGen tests the full chain of sinks
func TestTimeoutDropErrSinkGen(t *testing.T) {
	require := require.New(t)
	doneChan := make(chan struct{})

	sinkGen := NewTimeoutDropErrSinkGen(time.Second)

	// Generate two channels to perform the following test-cases
	sink, ch := sinkGen.NewChannelSink()
	sink2, ch2 := sinkGen.NewChannelSink()

	go func() {
		for {
			select {
			case <-ch.C:
			case <-doneChan:
				return
			}
		}
	}()
	require.NoError(sink.Write("some event"))

	// Make sure the sink times out on the write operation if the channel is
	// not read from.
	err := sink2.Write("some event")
	require.Error(err)
	require.Equal(ErrSinkTimeout, err)

	// Ensure that hitting a timeout causes the sink to close
	<-ch2.Done()

	// Make sure that closing a sink closes the channel
	var errClose error
	errClose = sink.Close()
	<-ch.Done()
	require.NoError(errClose)

	// Close the leaking goroutine
	close(doneChan)
}
