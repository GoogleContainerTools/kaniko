package genericresource

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
)

func TestClaimSingleDiscrete(t *testing.T) {
	var nodeRes, taskAssigned, taskReservations []*api.GenericResource

	nodeRes = append(nodeRes, NewDiscrete("apple", 3))
	taskReservations = append(taskReservations, NewDiscrete("apple", 2))

	err := Claim(&nodeRes, &taskAssigned, taskReservations)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 1)
	assert.Len(t, taskAssigned, 1)

	assert.Equal(t, int64(1), nodeRes[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(2), taskAssigned[0].GetDiscreteResourceSpec().Value)
}

func TestClaimMultipleDiscrete(t *testing.T) {
	var nodeRes, taskAssigned, taskReservations []*api.GenericResource

	nodeRes = append(nodeRes, NewDiscrete("apple", 3))
	nodeRes = append(nodeRes, NewDiscrete("orange", 4))
	nodeRes = append(nodeRes, NewDiscrete("banana", 2))
	nodeRes = append(nodeRes, NewDiscrete("cake", 1))

	// cake and banana should not be taken
	taskReservations = append(taskReservations, NewDiscrete("orange", 4))
	taskReservations = append(taskReservations, NewDiscrete("apple", 2))

	err := Claim(&nodeRes, &taskAssigned, taskReservations)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 3) // oranges isn't present anymore
	assert.Len(t, taskAssigned, 2)

	apples := GetResource("apple", taskAssigned)
	oranges := GetResource("orange", taskAssigned)
	assert.Len(t, apples, 1)
	assert.Len(t, oranges, 1)

	assert.Equal(t, int64(2), apples[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(4), oranges[0].GetDiscreteResourceSpec().Value)
}

func TestClaimSingleStr(t *testing.T) {
	var nodeRes, taskAssigned, taskReservations []*api.GenericResource

	nodeRes = append(nodeRes, NewSet("apple", "red", "orange", "blue", "green")...)
	taskReservations = append(taskReservations, NewDiscrete("apple", 2))

	err := Claim(&nodeRes, &taskAssigned, taskReservations)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 2)
	assert.Len(t, taskAssigned, 2)

	for _, k := range []string{"red", "orange"} {
		assert.True(t, HasResource(NewString("apple", k), taskAssigned))
	}
}

func TestClaimMultipleStr(t *testing.T) {
	var nodeRes, taskAssigned, taskReservations []*api.GenericResource

	nodeRes = append(nodeRes, NewSet("apple", "red", "orange", "blue", "green")...)
	nodeRes = append(nodeRes, NewSet("oranges", "red", "orange", "blue", "green")...)
	nodeRes = append(nodeRes, NewSet("bananas", "red", "orange", "blue", "green")...)
	taskReservations = append(taskReservations, NewDiscrete("oranges", 4))
	taskReservations = append(taskReservations, NewDiscrete("apple", 2))

	err := Claim(&nodeRes, &taskAssigned, taskReservations)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 6)
	assert.Len(t, taskAssigned, 6)

	apples := GetResource("apple", taskAssigned)
	for _, k := range []string{"red", "orange"} {
		assert.True(t, HasResource(NewString("apple", k), apples))
	}

	oranges := GetResource("oranges", taskAssigned)
	for _, k := range []string{"red", "orange", "blue", "green"} {
		assert.True(t, HasResource(NewString("oranges", k), oranges))
	}
}

func TestReclaimSingleDiscrete(t *testing.T) {
	var nodeRes, taskAssigned []*api.GenericResource

	taskAssigned = append(taskAssigned, NewDiscrete("apple", 2))

	err := reclaimResources(&nodeRes, taskAssigned)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 1)
	assert.Equal(t, int64(2), nodeRes[0].GetDiscreteResourceSpec().Value)

	err = reclaimResources(&nodeRes, taskAssigned)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 1)
	assert.Equal(t, int64(4), nodeRes[0].GetDiscreteResourceSpec().Value)
}

func TestReclaimMultipleDiscrete(t *testing.T) {
	var nodeRes, taskAssigned []*api.GenericResource

	nodeRes = append(nodeRes, NewDiscrete("apple", 3))
	nodeRes = append(nodeRes, NewDiscrete("banana", 2))

	// cake and banana should not be taken
	taskAssigned = append(taskAssigned, NewDiscrete("orange", 4))
	taskAssigned = append(taskAssigned, NewDiscrete("apple", 2))

	err := reclaimResources(&nodeRes, taskAssigned)
	assert.NoError(t, err)

	assert.Len(t, nodeRes, 3)

	apples := GetResource("apple", nodeRes)
	oranges := GetResource("orange", nodeRes)
	bananas := GetResource("banana", nodeRes)
	assert.Len(t, apples, 1)
	assert.Len(t, oranges, 1)
	assert.Len(t, bananas, 1)

	assert.Equal(t, int64(5), apples[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(4), oranges[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(2), bananas[0].GetDiscreteResourceSpec().Value)
}

func TestReclaimSingleStr(t *testing.T) {
	var nodeRes []*api.GenericResource
	taskAssigned := NewSet("apple", "red", "orange")

	err := reclaimResources(&nodeRes, taskAssigned)
	assert.NoError(t, err)
	assert.Len(t, nodeRes, 2)

	for _, k := range []string{"red", "orange"} {
		assert.True(t, HasResource(NewString("apple", k), nodeRes))
	}

	taskAssigned = NewSet("apple", "blue", "red")
	err = reclaimResources(&nodeRes, taskAssigned)

	assert.NoError(t, err)
	assert.Len(t, nodeRes, 4)

	for _, k := range []string{"red", "orange", "blue", "red"} {
		assert.True(t, HasResource(NewString("apple", k), nodeRes))
	}
}

func TestReclaimMultipleStr(t *testing.T) {
	nodeRes := NewSet("orange", "green")
	taskAssigned := NewSet("apple", "red", "orange")
	taskAssigned = append(taskAssigned, NewSet("orange", "red", "orange")...)

	err := reclaimResources(&nodeRes, taskAssigned)
	assert.NoError(t, err)
	assert.Len(t, nodeRes, 5)

	apples := GetResource("apple", nodeRes)
	oranges := GetResource("orange", nodeRes)
	assert.Len(t, apples, 2)
	assert.Len(t, oranges, 3)

	for _, k := range []string{"red", "orange"} {
		assert.True(t, HasResource(NewString("apple", k), apples))
	}

	for _, k := range []string{"red", "orange", "green"} {
		assert.True(t, HasResource(NewString("orange", k), oranges))
	}
}

func TestReclaimResources(t *testing.T) {
	nodeRes := NewSet("orange", "green", "blue")
	nodeRes = append(nodeRes, NewDiscrete("apple", 3))
	nodeRes = append(nodeRes, NewSet("banana", "red", "orange", "green")...)
	nodeRes = append(nodeRes, NewDiscrete("cake", 2))

	taskAssigned := NewSet("orange", "red", "orange")
	taskAssigned = append(taskAssigned, NewSet("grape", "red", "orange")...)
	taskAssigned = append(taskAssigned, NewDiscrete("apple", 3))
	taskAssigned = append(taskAssigned, NewDiscrete("coffe", 2))

	err := reclaimResources(&nodeRes, taskAssigned)
	assert.NoError(t, err)
	assert.Len(t, nodeRes, 12)

	apples := GetResource("apple", nodeRes)
	oranges := GetResource("orange", nodeRes)
	bananas := GetResource("banana", nodeRes)
	cakes := GetResource("cake", nodeRes)
	grapes := GetResource("grape", nodeRes)
	coffe := GetResource("coffe", nodeRes)
	assert.Len(t, apples, 1)
	assert.Len(t, oranges, 4)
	assert.Len(t, bananas, 3)
	assert.Len(t, cakes, 1)
	assert.Len(t, grapes, 2)
	assert.Len(t, coffe, 1)

	assert.Equal(t, int64(6), apples[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(2), cakes[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(2), coffe[0].GetDiscreteResourceSpec().Value)

	for _, k := range []string{"red", "orange", "green", "blue"} {
		assert.True(t, HasResource(NewString("orange", k), oranges))
	}

	for _, k := range []string{"red", "orange", "green"} {
		assert.True(t, HasResource(NewString("banana", k), bananas))
	}

	for _, k := range []string{"red", "orange"} {
		assert.True(t, HasResource(NewString("grape", k), grapes))
	}
}

func TestSanitizeDiscrete(t *testing.T) {
	var nodeRes, nodeAvailableResources []*api.GenericResource
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("orange", 4))

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 0)

	nodeRes = append(nodeRes, NewDiscrete("orange", 6))
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("orange", 4))

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 1)
	assert.Equal(t, int64(4), nodeAvailableResources[0].GetDiscreteResourceSpec().Value)

	nodeRes[0].GetDiscreteResourceSpec().Value = 4

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 1)
	assert.Equal(t, int64(4), nodeAvailableResources[0].GetDiscreteResourceSpec().Value)

	nodeRes[0].GetDiscreteResourceSpec().Value = 2

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 1)
	assert.Equal(t, int64(2), nodeAvailableResources[0].GetDiscreteResourceSpec().Value)

	nodeRes = append(nodeRes, NewDiscrete("banana", 6))
	nodeRes = append(nodeRes, NewDiscrete("cake", 6))
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("cake", 2))
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("apple", 4))
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("banana", 8))

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 3)
	assert.Equal(t, int64(2), nodeAvailableResources[0].GetDiscreteResourceSpec().Value) // oranges
	assert.Equal(t, int64(2), nodeAvailableResources[1].GetDiscreteResourceSpec().Value) // cake
	assert.Equal(t, int64(6), nodeAvailableResources[2].GetDiscreteResourceSpec().Value) // banana
}

func TestSanitizeStr(t *testing.T) {
	var nodeRes []*api.GenericResource
	nodeAvailableResources := NewSet("apple", "red", "orange", "blue")

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 0)

	nodeAvailableResources = NewSet("apple", "red", "orange", "blue")
	nodeRes = NewSet("apple", "red", "orange", "blue", "green")
	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 3)

	nodeRes = NewSet("apple", "red", "orange", "blue")
	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 3)

	nodeRes = NewSet("apple", "red", "orange")
	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 2)
}

func TestSanitizeChangeDiscreteToSet(t *testing.T) {
	nodeRes := NewSet("apple", "red")
	nodeAvailableResources := []*api.GenericResource{NewDiscrete("apple", 5)}

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 1)
	assert.Equal(t, "red", nodeAvailableResources[0].GetNamedResourceSpec().Value)

	nodeRes = NewSet("apple", "red", "orange", "green")
	nodeAvailableResources = []*api.GenericResource{NewDiscrete("apple", 5)}

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 3)

	for _, k := range []string{"red", "orange", "green"} {
		assert.True(t, HasResource(NewString("apple", k), nodeAvailableResources))
	}

	nodeRes = append(nodeRes, NewSet("orange", "red", "orange", "green")...)
	nodeRes = append(nodeRes, NewSet("cake", "red", "orange", "green")...)

	nodeAvailableResources = NewSet("apple", "green")
	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("cake", 3))
	nodeAvailableResources = append(nodeAvailableResources, NewSet("orange", "orange", "blue")...)

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 5)

	apples := GetResource("apple", nodeAvailableResources)
	oranges := GetResource("orange", nodeAvailableResources)
	cakes := GetResource("cake", nodeAvailableResources)
	assert.Len(t, apples, 1)
	assert.Len(t, oranges, 1)
	assert.Len(t, cakes, 3)

	for _, k := range []string{"green"} {
		assert.True(t, HasResource(NewString("apple", k), apples))
	}

	for _, k := range []string{"orange"} {
		assert.True(t, HasResource(NewString("orange", k), oranges))
	}

	for _, k := range []string{"red", "orange", "green"} {
		assert.True(t, HasResource(NewString("cake", k), cakes))
	}
}

func TestSanitizeChangeSetToDiscrete(t *testing.T) {
	nodeRes := []*api.GenericResource{NewDiscrete("apple", 5)}
	nodeAvailableResources := NewSet("apple", "red")

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 1)
	assert.Equal(t, int64(5), nodeAvailableResources[0].GetDiscreteResourceSpec().Value)

	nodeRes = []*api.GenericResource{NewDiscrete("apple", 5)}
	nodeAvailableResources = NewSet("apple", "red", "orange", "green")

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 1)
	assert.Equal(t, int64(5), nodeAvailableResources[0].GetDiscreteResourceSpec().Value)

	nodeRes = append(nodeRes, NewDiscrete("orange", 3))
	nodeRes = append(nodeRes, NewDiscrete("cake", 1))

	nodeAvailableResources = append(nodeAvailableResources, NewDiscrete("cake", 2))
	nodeAvailableResources = append(nodeAvailableResources, NewSet("orange", "orange", "blue")...)

	sanitize(nodeRes, &nodeAvailableResources)
	assert.Len(t, nodeAvailableResources, 3)

	apples := GetResource("apple", nodeAvailableResources)
	oranges := GetResource("orange", nodeAvailableResources)
	cakes := GetResource("cake", nodeAvailableResources)
	assert.Len(t, apples, 1)
	assert.Len(t, oranges, 1)
	assert.Len(t, cakes, 1)

	assert.Equal(t, int64(5), apples[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(3), oranges[0].GetDiscreteResourceSpec().Value)
	assert.Equal(t, int64(1), cakes[0].GetDiscreteResourceSpec().Value)
}
