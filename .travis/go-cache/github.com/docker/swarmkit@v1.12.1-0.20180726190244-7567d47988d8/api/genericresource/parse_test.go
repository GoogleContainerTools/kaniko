package genericresource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDiscrete(t *testing.T) {
	res, err := ParseCmd("apple=3")
	assert.NoError(t, err)
	assert.Equal(t, len(res), 1)

	apples := GetResource("apple", res)
	assert.Equal(t, len(apples), 1)
	assert.Equal(t, apples[0].GetDiscreteResourceSpec().Value, int64(3))

	_, err = ParseCmd("apple=3\napple=4")
	assert.Error(t, err)

	_, err = ParseCmd("apple=3,apple=4")
	assert.Error(t, err)

	_, err = ParseCmd("apple=-3")
	assert.Error(t, err)
}

func TestParseStr(t *testing.T) {
	res, err := ParseCmd("orange=red,orange=green,orange=blue")
	assert.NoError(t, err)
	assert.Equal(t, len(res), 3)

	oranges := GetResource("orange", res)
	assert.Equal(t, len(oranges), 3)
	for _, k := range []string{"red", "green", "blue"} {
		assert.True(t, HasResource(NewString("orange", k), oranges))
	}
}

func TestParseDiscreteAndStr(t *testing.T) {
	res, err := ParseCmd("orange=red,orange=green,orange=blue,apple=3")
	assert.NoError(t, err)
	assert.Equal(t, len(res), 4)

	oranges := GetResource("orange", res)
	assert.Equal(t, len(oranges), 3)
	for _, k := range []string{"red", "green", "blue"} {
		assert.True(t, HasResource(NewString("orange", k), oranges))
	}

	apples := GetResource("apple", res)
	assert.Equal(t, len(apples), 1)
	assert.Equal(t, apples[0].GetDiscreteResourceSpec().Value, int64(3))
}
