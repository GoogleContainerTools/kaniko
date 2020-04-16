package buildcontext

import (
	"github.com/GoogleContainerTools/kaniko/testutil"
	"os"
	"testing"
)

func TestGetGitPullMethod(t *testing.T) {
	tests := []struct {
		setEnv        func()
		expectedValue string
	}{
		{
			setEnv:        func() {},
			expectedValue: "https",
		},
		{
			setEnv: func() {
				_ = os.Setenv(gitPullMethodEnvKey, "http")
			},
			expectedValue: "http",
		},
		{
			setEnv: func() {
				_ = os.Setenv(gitPullMethodEnvKey, "https")
			},
			expectedValue: "https",
		},
		{
			setEnv: func() {
				_ = os.Setenv(gitPullMethodEnvKey, "unknown")
			},
			expectedValue: "https",
		},
	}

	for _, tt := range tests {
		tt.setEnv()
		testutil.CheckDeepEqual(t, getGitPullMethod(), tt.expectedValue)
	}
}
