package genericresource

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
)

func TestConsumeResourcesSingle(t *testing.T) {
	nodeAvailableResources := NewSet("apple", "red", "orange", "blue")
	res := NewSet("apple", "red")

	ConsumeNodeResources(&nodeAvailableResources, res)
	assert.Len(t, nodeAvailableResources, 2)

	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("apple", 1))
	res = []*api.GenericResource{NewDiscrete("apple", 1)}

	ConsumeNodeResources(&nodeAvailableResources, res)
	assert.Len(t, nodeAvailableResources, 2)

	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("apple", 4))
	res = []*api.GenericResource{NewDiscrete("apple", 1)}

	ConsumeNodeResources(&nodeAvailableResources, res)
	assert.Len(t, nodeAvailableResources, 3)
	assert.Equal(t, int64(3), nodeAvailableResources[2].GetDiscreteResourceSpec().Value)
}

func TestConsumeResourcesMultiple(t *testing.T) {
	nodeAvailableResources := NewSet("apple", "red", "orange", "blue", "green", "yellow")
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("orange", 5))
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("banana", 3))
	nodeAvailableResources = append(nodeAvailableResources, NewSet("grape", "red", "orange", "blue", "green", "yellow")...)
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("cakes", 3))

	res := NewSet("apple", "red")
	res = append(res, NewDiscrete("banana", 2))
	res = append(res, NewSet("apple", "green", "blue", "red")...)
	res = append(res, NewSet("grape", "red", "blue", "red")...)
	res = append(res, NewDiscrete("cakes", 3))

	ConsumeNodeResources(&nodeAvailableResources, res)
	assert.Len(t, nodeAvailableResources, 7)

	apples := GetResource("apple", nodeAvailableResources)
	oranges := GetResource("orange", nodeAvailableResources)
	bananas := GetResource("banana", nodeAvailableResources)
	grapes := GetResource("grape", nodeAvailableResources)
	assert.Len(t, apples, 2)
	assert.Len(t, oranges, 1)
	assert.Len(t, bananas, 1)
	assert.Len(t, grapes, 3)

	for _, k := range []string{"yellow", "orange"} {
		assert.True(t, HasResource(NewString("apple", k), apples))
	}

	for _, k := range []string{"yellow", "orange", "green"} {
		assert.True(t, HasResource(NewString("grape", k), grapes))
	}

	assert.Equal(t, int64(5), oranges[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(1), bananas[0].GetDiscreteResourceSpec().Value)
}
