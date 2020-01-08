package testutils

import (
	"errors"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/manager/state"
	"golang.org/x/net/context"
)

// MockProposer is a simple proposer implementation for use in tests.
type MockProposer struct {
	index   uint64
	changes []state.Change
}

// ProposeValue propagates a value. In this mock implementation, it just stores
// the value locally.
func (mp *MockProposer) ProposeValue(ctx context.Context, storeAction []api.StoreAction, cb func()) error {
	mp.index += 3
	mp.changes = append(mp.changes,
		state.Change{
			Version:      api.Version{Index: mp.index},
			StoreActions: storeAction,
		},
	)
	if cb != nil {
		cb()
	}
	return nil
}

// GetVersion returns the current version.
func (mp *MockProposer) GetVersion() *api.Version {
	return &api.Version{Index: mp.index}
}

// ChangesBetween returns changes after "from" up to and including "to".
func (mp *MockProposer) ChangesBetween(from, to api.Version) ([]state.Change, error) {
	var changes []state.Change

	if len(mp.changes) == 0 {
		return nil, errors.New("no history")
	}

	lastIndex := mp.changes[len(mp.changes)-1].Version.Index

	if to.Index > lastIndex || from.Index > lastIndex {
		return nil, errors.New("out of bounds")
	}

	for _, change := range mp.changes {
		if change.Version.Index > from.Index && change.Version.Index <= to.Index {
			changes = append(changes, change)
		}
	}

	return changes, nil
}
