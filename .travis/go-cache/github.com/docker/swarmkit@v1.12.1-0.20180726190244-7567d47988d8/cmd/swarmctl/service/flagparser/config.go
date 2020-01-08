package flagparser

import (
	"fmt"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/cmd/swarmctl/common"
	"github.com/spf13/cobra"
)

// expects configs in the format CONFIG_NAME:TARGET_NAME
func parseConfigString(configString string) (configName, presentName string, err error) {
	tokens := strings.Split(configString, ":")

	configName = strings.TrimSpace(tokens[0])

	if configName == "" {
		err = fmt.Errorf("invalid config name provided")
		return
	}

	if len(tokens) > 1 {
		presentName = strings.TrimSpace(tokens[1])
		if presentName == "" {
			err = fmt.Errorf("invalid presentation name provided")
			return
		}
	} else {
		presentName = configName
	}
	return
}

// ParseAddConfig validates configs passed on the command line
func ParseAddConfig(cmd *cobra.Command, spec *api.ServiceSpec, flagName string) error {
	flags := cmd.Flags()

	if flags.Changed(flagName) {
		configs, err := flags.GetStringSlice(flagName)
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()
		if container == nil {
			spec.Task.Runtime = &api.TaskSpec_Container{
				Container: &api.ContainerSpec{},
			}
		}

		lookupConfigNames := []string{}
		var needConfigs []*api.ConfigReference

		for _, config := range configs {
			n, p, err := parseConfigString(config)
			if err != nil {
				return err
			}

			// TODO(diogo): defaults to File targets, but in the future will take different types
			configRef := &api.ConfigReference{
				ConfigName: n,
				Target: &api.ConfigReference_File{
					File: &api.FileTarget{
						Name: p,
						Mode: 0444,
					},
				},
			}

			lookupConfigNames = append(lookupConfigNames, n)
			needConfigs = append(needConfigs, configRef)
		}

		client, err := common.Dial(cmd)
		if err != nil {
			return err
		}

		r, err := client.ListConfigs(common.Context(cmd),
			&api.ListConfigsRequest{Filters: &api.ListConfigsRequest_Filters{Names: lookupConfigNames}})
		if err != nil {
			return err
		}

		foundConfigs := make(map[string]*api.Config)
		for _, config := range r.Configs {
			foundConfigs[config.Spec.Annotations.Name] = config
		}

		for _, configRef := range needConfigs {
			config, ok := foundConfigs[configRef.ConfigName]
			if !ok {
				return fmt.Errorf("config not found: %s", configRef.ConfigName)
			}

			configRef.ConfigID = config.ID
			container.Configs = append(container.Configs, configRef)
		}
	}

	return nil
}

// ParseRemoveConfig removes a set of configs from the task spec's config references
func ParseRemoveConfig(cmd *cobra.Command, spec *api.ServiceSpec, flagName string) error {
	flags := cmd.Flags()

	if flags.Changed(flagName) {
		configs, err := flags.GetStringSlice(flagName)
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()
		if container == nil {
			return nil
		}

		wantToDelete := make(map[string]struct{})

		for _, config := range configs {
			n, _, err := parseConfigString(config)
			if err != nil {
				return err
			}

			wantToDelete[n] = struct{}{}
		}

		configRefs := []*api.ConfigReference{}

		for _, configRef := range container.Configs {
			if _, ok := wantToDelete[configRef.ConfigName]; ok {
				continue
			}
			configRefs = append(configRefs, configRef)
		}

		container.Configs = configRefs
	}
	return nil
}
