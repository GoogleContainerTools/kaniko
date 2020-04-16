package buildcontext

import (
	"os"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestGetGitPullMethod(t *testing.T) {
	tests := []struct {
		testName string
		setEnv   func() (expectedValue string)
	}{
		{
			testName: "noEnv",
			setEnv: func() (expectedValue string) {
				expectedValue = "https"
				return
			},
		},
		{
			testName: "emptyEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "")
				expectedValue = "https"
				return
			},
		},
		{
			testName: "httpEnv",
			setEnv: func() (expectedValue string) {
				err := os.Setenv(gitPullMethodEnvKey, "http")
				if nil != err {
					expectedValue = "https"
				} else {
					expectedValue = "http"
				}
				return
			},
		},
		{
			testName: "httpsEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "https")
				expectedValue = "https"
				return
			},
		},
		{
			testName: "unknownEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "unknown")
				expectedValue = "https"
				return
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			expectedValue := tt.setEnv()
			testutil.CheckDeepEqual(t, expectedValue, getGitPullMethod())
		})
	}
}
