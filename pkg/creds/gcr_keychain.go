package creds

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
)

type gcrKeychain struct {
	authr authn.Authenticator
}

func (g gcrKeychain) Resolve(r authn.Resource) (authn.Authenticator, error) {
	if r.RegistryStr() == "gcr.io" ||
		strings.HasSuffix(r.RegistryStr(), ".gcr.io") ||
		strings.HasSuffix(r.RegistryStr(), ".pkg.dev") {

		return g.authr, nil
	}
	return authn.Anonymous, nil
}
