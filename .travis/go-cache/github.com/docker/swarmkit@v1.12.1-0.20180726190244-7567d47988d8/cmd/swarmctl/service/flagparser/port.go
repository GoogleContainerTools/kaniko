package flagparser

import (
	"strconv"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

func parsePorts(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if !flags.Changed("ports") {
		return nil
	}
	portConfigs, err := flags.GetStringSlice("ports")
	if err != nil {
		return err
	}

	ports := []*api.PortConfig{}
	for _, portConfig := range portConfigs {
		name, protocol, port, swarmPort, err := parsePortConfig(portConfig)
		if err != nil {
			return err
		}

		ports = append(ports, &api.PortConfig{
			Name:          name,
			Protocol:      protocol,
			TargetPort:    port,
			PublishedPort: swarmPort,
			// In swarmctl all ports are by default
			// PublishModeHost
			PublishMode: api.PublishModeHost,
		})
	}

	spec.Endpoint = &api.EndpointSpec{
		Ports: ports,
	}

	return nil
}

func parsePortConfig(portConfig string) (string, api.PortConfig_Protocol, uint32, uint32, error) {
	protocol := api.ProtocolTCP
	parts := strings.Split(portConfig, ":")
	if len(parts) < 2 {
		return "", protocol, 0, 0, errors.New("insufficient parameters in port configuration")
	}

	name := parts[0]

	portSpec := parts[1]
	protocol, port, err := parsePortSpec(portSpec)
	if err != nil {
		return "", protocol, 0, 0, errors.Wrap(err, "failed to parse port")
	}

	if len(parts) > 2 {
		var err error

		portSpec := parts[2]
		nodeProtocol, swarmPort, err := parsePortSpec(portSpec)
		if err != nil {
			return "", protocol, 0, 0, errors.Wrap(err, "failed to parse node port")
		}

		if nodeProtocol != protocol {
			return "", protocol, 0, 0, errors.New("protocol mismatch")
		}

		return name, protocol, port, swarmPort, nil
	}

	return name, protocol, port, 0, nil
}

func parsePortSpec(portSpec string) (api.PortConfig_Protocol, uint32, error) {
	parts := strings.Split(portSpec, "/")
	p := parts[0]
	port, err := strconv.ParseUint(p, 10, 32)
	if err != nil {
		return 0, 0, err
	}

	if len(parts) > 1 {
		proto := parts[1]
		protocol, ok := api.PortConfig_Protocol_value[strings.ToUpper(proto)]
		if !ok {
			return 0, 0, errors.Errorf("invalid protocol string: %s", proto)
		}

		return api.PortConfig_Protocol(protocol), uint32(port), nil
	}

	return api.ProtocolTCP, uint32(port), nil
}
