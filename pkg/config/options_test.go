package config

import (
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestKanikoGitOptions(t *testing.T) {
	t.Run("invalid pair", func(t *testing.T) {
		var g = &KanikoGitOptions{}
		testutil.CheckError(t, true, g.Set("branch"))
	})

	t.Run("sets values", func(t *testing.T) {
		var g = &KanikoGitOptions{}
		testutil.CheckNoError(t, g.Set("branch=foo"))
		testutil.CheckNoError(t, g.Set("recurse-submodules=true"))
		testutil.CheckNoError(t, g.Set("single-branch=true"))
		testutil.CheckDeepEqual(t, KanikoGitOptions{
			Branch:            "foo",
			SingleBranch:      true,
			RecurseSubmodules: true,
		}, *g)
	})

	t.Run("sets bools other than true", func(t *testing.T) {
		var g = KanikoGitOptions{}
		testutil.CheckError(t, true, g.Set("recurse-submodules="))
		testutil.CheckError(t, true, g.Set("single-branch=zaza"))
		testutil.CheckNoError(t, g.Set("recurse-submodules=false"))
		testutil.CheckDeepEqual(t, KanikoGitOptions{
			SingleBranch:      false,
			RecurseSubmodules: false,
		}, g)
	})
}
