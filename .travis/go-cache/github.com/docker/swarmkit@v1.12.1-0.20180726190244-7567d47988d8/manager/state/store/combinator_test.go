package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrCombinator(t *testing.T) {
	s := NewMemoryStore(nil)
	assert.NotNil(t, s)

	setupTestStore(t, s)

	s.View(func(readTx ReadTx) {
		foundNodes, err := FindNodes(readTx, Or())
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 0)

		foundNodes, err = FindNodes(readTx, Or(ByName("name1")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		foundNodes, err = FindNodes(readTx, Or(ByName("name1"), ByName("name1")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		foundNodes, err = FindNodes(readTx, Or(ByName("name1"), ByName("name2")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 3)

		foundNodes, err = FindNodes(readTx, Or(ByName("name1"), ByIDPrefix("id1")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		foundNodes, err = FindNodes(readTx, Or(ByName("name1"), ByIDPrefix("id5295")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 1)

		foundNodes, err = FindNodes(readTx, Or(ByIDPrefix("id1"), ByIDPrefix("id2")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 2)

		foundNodes, err = FindNodes(readTx, Or(ByIDPrefix("id1"), ByIDPrefix("id2"), ByIDPrefix("id3")))
		assert.NoError(t, err)
		assert.Len(t, foundNodes, 3)
	})
}
