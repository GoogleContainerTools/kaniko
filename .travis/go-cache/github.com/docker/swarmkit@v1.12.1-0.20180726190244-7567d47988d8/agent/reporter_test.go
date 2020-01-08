package agent

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

type uniqueStatus struct {
	taskID string
	status *api.TaskStatus
}

func TestReporter(t *testing.T) {
	const ntasks = 100

	var (
		ctx      = context.Background()
		statuses = make(map[string]*api.TaskStatus) // destination map
		unique   = make(map[uniqueStatus]struct{})  // ensure we don't receive any status twice
		mu       sync.Mutex
		expected = make(map[string]*api.TaskStatus)
		wg       sync.WaitGroup
	)

	reporter := newStatusReporter(ctx, statusReporterFunc(func(ctx context.Context, taskID string, status *api.TaskStatus) error {
		if rand.Float64() > 0.9 {
			return errors.New("status send failed")
		}

		mu.Lock()
		defer mu.Unlock()

		key := uniqueStatus{taskID, status}
		// make sure we get the status only once.
		if _, ok := unique[key]; ok {
			t.Fatal("encountered status twice")
		}

		if status.State == api.TaskStateCompleted {
			wg.Done()
		}

		unique[key] = struct{}{}
		if current, ok := statuses[taskID]; ok {
			if status.State <= current.State {
				return nil // only allow forward updates
			}
		}

		statuses[taskID] = status

		return nil
	}))

	wg.Add(ntasks) // statuses will be reported!

	for _, state := range []api.TaskState{
		api.TaskStateAccepted,
		api.TaskStatePreparing,
		api.TaskStateReady,
		api.TaskStateCompleted,
	} {
		for i := 0; i < ntasks; i++ {
			taskID, status := fmt.Sprint(i), &api.TaskStatus{State: state}
			expected[taskID] = status

			// simulate pounding this with a bunch of goroutines
			go func() {
				if err := reporter.UpdateTaskStatus(ctx, taskID, status); err != nil {
					assert.NoError(t, err, "sending should not fail")
				}
			}()

		}
	}

	wg.Wait() // wait for the propagation
	assert.NoError(t, reporter.Close())
	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, expected, statuses)
}
