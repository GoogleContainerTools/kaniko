package authprovider

import (
	"context"
	"io/ioutil"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"google.golang.org/grpc"
)

func NewDockerAuthProvider() session.Attachable {
	return &authProvider{
		config: config.LoadDefaultConfigFile(ioutil.Discard),
	}
}

type authProvider struct {
	config *configfile.ConfigFile
}

func (ap *authProvider) Register(server *grpc.Server) {
	auth.RegisterAuthServer(server, ap)
}

func (ap *authProvider) Credentials(ctx context.Context, req *auth.CredentialsRequest) (*auth.CredentialsResponse, error) {
	if req.Host == "registry-1.docker.io" {
		req.Host = "https://index.docker.io/v1/"
	}
	ac, err := ap.config.GetAuthConfig(req.Host)
	if err != nil {
		return nil, err
	}
	res := &auth.CredentialsResponse{}
	if ac.IdentityToken != "" {
		res.Secret = ac.IdentityToken
	} else {
		res.Username = ac.Username
		res.Secret = ac.Password
	}
	return res, nil
}
