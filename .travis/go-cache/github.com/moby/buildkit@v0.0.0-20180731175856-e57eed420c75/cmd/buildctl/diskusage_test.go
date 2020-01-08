package main

import (
	"testing"

	"github.com/moby/buildkit/util/testutil/integration"
	"github.com/stretchr/testify/assert"
)

func testDiskUsage(t *testing.T, sb integration.Sandbox) {
	t.Parallel()
	cmd := sb.Cmd("du")
	err := cmd.Run()
	assert.NoError(t, err)
}
