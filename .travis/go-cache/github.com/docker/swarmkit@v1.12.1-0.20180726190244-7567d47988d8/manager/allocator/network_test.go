package allocator

import (
	"testing"

	"github.com/docker/swarmkit/api"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePortsInHostPublishMode(t *testing.T) {
	service := api.Service{
		Spec: api.ServiceSpec{
			Endpoint: &api.EndpointSpec{
				Ports: []*api.PortConfig{
					{
						Protocol:      api.ProtocolTCP,
						TargetPort:    80,
						PublishedPort: 10000,
						PublishMode:   api.PublishModeHost,
					},
				},
			},
		},
		Endpoint: &api.Endpoint{
			Ports: []*api.PortConfig{
				{
					Protocol:      api.ProtocolTCP,
					TargetPort:    80,
					PublishedPort: 15000,
					PublishMode:   api.PublishModeHost,
				},
			},
		},
	}
	updatePortsInHostPublishMode(&service)

	assert.Equal(t, len(service.Endpoint.Ports), 1)
	assert.Equal(t, service.Endpoint.Ports[0].PublishedPort, uint32(10000))
	assert.Equal(t, service.Endpoint.Spec.Ports[0].PublishedPort, uint32(10000))
}
