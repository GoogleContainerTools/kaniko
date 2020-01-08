package constraint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	// empty string
	_, err := Parse([]string{""})
	assert.Error(t, err)

	_, err = Parse([]string{" "})
	assert.Error(t, err)

	// no operator
	_, err = Parse([]string{"nodeabc"})
	assert.Error(t, err)

	// incorrect operator
	_, err = Parse([]string{"node ~ abc"})
	assert.Error(t, err)

	// Cannot use the leading digit for key
	_, err = Parse([]string{"1node==a2"})
	assert.Error(t, err)

	// leading and trailing white space are ignored
	_, err = Parse([]string{" node == node1"})
	assert.NoError(t, err)

	// key cannot container white space in the middle
	_, err = Parse([]string{"no de== node1"})
	assert.Error(t, err)

	// Cannot use * in key
	_, err = Parse([]string{"no*de==node1"})
	assert.Error(t, err)

	// key cannot be empty
	_, err = Parse([]string{"==node1"})
	assert.Error(t, err)

	// value cannot be empty
	_, err = Parse([]string{"node=="})
	assert.Error(t, err)

	// value cannot be an empty space
	_, err = Parse([]string{"node== "})
	assert.Error(t, err)

	// Cannot use $ in key
	_, err = Parse([]string{"no$de==node1"})
	assert.Error(t, err)

	// Allow CAPS in key
	exprs, err := Parse([]string{"NoDe==node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "NoDe")

	// Allow dot in key
	exprs, err = Parse([]string{"no.de==node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "no.de")

	// Allow leading underscore
	exprs, err = Parse([]string{"_node==_node1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "_node")

	// Allow special characters in exp
	exprs, err = Parse([]string{"node==[a-b]+c*(n|b)/"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "node")
	assert.Equal(t, exprs[0].exp, "[a-b]+c*(n|b)/")

	// Allow space in Exp
	exprs, err = Parse([]string{"node==node 1"})
	assert.NoError(t, err)
	assert.Equal(t, exprs[0].key, "node")
	assert.Equal(t, exprs[0].exp, "node 1")
}

func TestMatch(t *testing.T) {
	exprs, err := Parse([]string{"node.name==foo"})
	assert.NoError(t, err)
	e := exprs[0]
	assert.True(t, e.Match("foo"))
	assert.False(t, e.Match("fo"))
	assert.False(t, e.Match("fooE"))

	exprs, err = Parse([]string{"node.name!=foo"})
	assert.NoError(t, err)
	e = exprs[0]
	assert.False(t, e.Match("foo"))
	assert.True(t, e.Match("bar"))
	assert.True(t, e.Match("fo"))
	assert.True(t, e.Match("fooExtra"))

	exprs, err = Parse([]string{"node.name==f*o"})
	assert.NoError(t, err)
	e = exprs[0]
	assert.False(t, e.Match("fo"))
	assert.True(t, e.Match("f*o"))
	assert.True(t, e.Match("F*o"))
	assert.False(t, e.Match("foo", "fo", "bar"))
	assert.True(t, e.Match("foo", "f*o", "bar"))
	assert.False(t, e.Match("foo"))

	// test special characters
	exprs, err = Parse([]string{"node.name==f.-$o"})
	assert.NoError(t, err)
	e = exprs[0]
	assert.False(t, e.Match("fa-$o"))
	assert.True(t, e.Match("f.-$o"))
}
