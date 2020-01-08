package main

import (
	"testing"

	"github.com/moby/buildkit/util/testutil/integration"
	"github.com/stretchr/testify/require"
)

func TestCLIIntegration(t *testing.T) {
	integration.Run(t, []integration.Test{
		testDiskUsage,
		testBuildWithLocalFiles,
		testBuildLocalExporter,
		testBuildContainerdExporter,
		testPrune,
		testUsage,
	})
}

func testUsage(t *testing.T, sb integration.Sandbox) {
	t.Parallel()

	require.NoError(t, sb.Cmd().Run())

	require.NoError(t, sb.Cmd("--help").Run())
}
