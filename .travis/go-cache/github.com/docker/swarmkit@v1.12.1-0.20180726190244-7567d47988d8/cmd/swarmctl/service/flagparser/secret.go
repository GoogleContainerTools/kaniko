package flagparser

import (
	"fmt"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

// expects secrets in the format SECRET_NAME:TARGET_NAME
func parseSecretString(secretString string) (secretName, presentName string, err error) {
	tokens := strings.Split(secretString, ":")

	secretName = strings.TrimSpace(tokens[0])

	if secretName == "" {
		err = fmt.Errorf("invalid secret name provided")
		return
	}

	if len(tokens) > 1 {
		presentName = strings.TrimSpace(tokens[1])
		if presentName == "" {
			err = fmt.Errorf("invalid presentation name provided")
			return
		}
	} else {
		presentName = secretName
	}
	return
}

// ParseAddSecret validates secrets passed on the command line
func ParseAddSecret(cmd *cobra.Command, spec *api.ServiceSpec, flagName string) error {
	flags := cmd.Flags()

	if flags.Changed(flagName) {
		secrets, err := flags.GetStringSlice(flagName)
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()
		if container == nil {
			spec.Task.Runtime = &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			}
		}

		lookupSecretNames := []string{}
		var needSecrets []*api.SecretReference

		for _, secret := range secrets {
			n, p, err := parseSecretString(secret)
			if err != nil {
				return err
			}

			// TODO(diogo): defaults to File targets, but in the future will take different types
			secretRef := &api.SecretReference{
				SecretName: n,
				Target: &api.SecretReference_File{
					File: &api.FileTarget{
						Name: p,
						Mode: 0444,
					},
				},
			}

			lookupSecretNames = append(lookupSecretNames, n)
			needSecrets = append(needSecrets, secretRef)
		}

		client, err := common.Dial(cmd)
		if err != nil {
			return err
		}

		r, err := client.ListSecrets(common.Context(cmd),
			&api.ListSecretsRequest{Filters: &api.ListSecretsRequest_Filters{Names: lookupSecretNames}})
		if err != nil {
			return err
		}

		foundSecrets := make(map[string]*api.Secret)
		for _, secret := range r.Secrets {
			foundSecrets[secret.Spec.Annotations.Name] = secret
		}

		for _, secretRef := range needSecrets {
			secret, ok := foundSecrets[secretRef.SecretName]
			if !ok {
				return fmt.Errorf("secret not found: %s", secretRef.SecretName)
			}

			secretRef.SecretID = secret.ID
			container.Secrets = append(container.Secrets, secretRef)
		}
	}

	return nil
}

// ParseRemoveSecret removes a set of secrets from the task spec's secret references
func ParseRemoveSecret(cmd *cobra.Command, spec *api.ServiceSpec, flagName string) error {
	flags := cmd.Flags()

	if flags.Changed(flagName) {
		secrets, err := flags.GetStringSlice(flagName)
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()
		if container == nil {
			return nil
		}

		wantToDelete := make(map[string]struct{})

		for _, secret := range secrets {
			n, _, err := parseSecretString(secret)
			if err != nil {
				return err
			}

			wantToDelete[n] = struct{}{}
		}

		secretRefs := []*api.SecretReference{}

		for _, secretRef := range container.Secrets {
			if _, ok := wantToDelete[secretRef.SecretName]; ok {
				continue
			}
			secretRefs = append(secretRefs, secretRef)
		}

		container.Secrets = secretRefs
	}
	return nil
}
