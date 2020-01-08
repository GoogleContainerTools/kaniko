package secret

import (
	"fmt"

	"github.com/docker/swarmkit/api"
	"golang.org/x/net/context"
)

func getSecret(ctx context.Context, c api.ControlClient, input string) (*api.Secret, error) {
	// not sure what it is, match by name or id prefix
	resp, err := c.ListSecrets(ctx,
		&api.ListSecretsRequest{
			Filters: &api.ListSecretsRequest_Filters{
				Names:      []string{input},
				IDPrefixes: []string{input},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	switch len(resp.Secrets) {
	case 0:
		return nil, fmt.Errorf("secret %s not found", input)
	case 1:
		return resp.Secrets[0], nil
	default:
		// ok, multiple matches.  Prefer exact ID over exact name.  If no exact matches, return an error
		for _, s := range resp.Secrets {
			if s.ID == input {
				return s, nil
			}
		}
		for _, s := range resp.Secrets {
			if s.Spec.Annotations.Name == input {
				return s, nil
			}
		}
		return nil, fmt.Errorf("secret %s is ambiguous (%d matches found)", input, len(resp.Secrets))
	}
}
