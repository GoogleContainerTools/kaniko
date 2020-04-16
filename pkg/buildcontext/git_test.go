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
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
		{
			testName: "emptyEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "")
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
		{
			testName: "httpEnv",
			setEnv: func() (expectedValue string) {
				err := os.Setenv(gitPullMethodEnvKey, gitPullMethodHTTP)
				if nil != err {
					expectedValue = gitPullMethodHTTPS
				} else {
					expectedValue = gitPullMethodHTTP
				}
				return
			},
		},
		{
			testName: "httpsEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, gitPullMethodHTTPS)
				expectedValue = gitPullMethodHTTPS
				return
			},
		},
		{
			testName: "unknownEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "unknown")
				expectedValue = gitPullMethodHTTPS
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
