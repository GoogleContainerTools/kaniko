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
				expectedValue = gitPullMethodHttps
				return
			},
		},
		{
			testName: "emptyEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "")
				expectedValue = gitPullMethodHttps
				return
			},
		},
		{
			testName: "httpEnv",
			setEnv: func() (expectedValue string) {
				err := os.Setenv(gitPullMethodEnvKey, gitPullMethodHttp)
				if nil != err {
					expectedValue = gitPullMethodHttps
				} else {
					expectedValue = gitPullMethodHttp
				}
				return
			},
		},
		{
			testName: "httpsEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, gitPullMethodHttps)
				expectedValue = gitPullMethodHttps
				return
			},
		},
		{
			testName: "unknownEnv",
			setEnv: func() (expectedValue string) {
				_ = os.Setenv(gitPullMethodEnvKey, "unknown")
				expectedValue = gitPullMethodHttps
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
