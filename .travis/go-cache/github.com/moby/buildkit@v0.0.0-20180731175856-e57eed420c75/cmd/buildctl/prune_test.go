package main

import (
	"testing"

	"github.com/moby/buildkit/util/testutil/integration"
	"github.com/stretchr/testify/assert"
)

func testPrune(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	cmd := sb.Cmd("prune")
	err := cmd.Run()
	assert.NoError(t, err)
}
