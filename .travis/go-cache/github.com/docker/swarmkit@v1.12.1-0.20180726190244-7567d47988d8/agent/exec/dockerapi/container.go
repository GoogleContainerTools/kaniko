package dockerapi

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	enginecontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	enginemount "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/docker/swarmkit/agent/exec"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
	"github.com/docker/swarmkit/api/naming"
	"github.com/docker/swarmkit/template"
	gogotypes "github.com/gogo/protobuf/types"
)

const (
	// Explicitly use the kernel's default setting for CPU quota of 100ms.
	// https://www.kernel.org/doc/Documentation/scheduler/sched-bwc.txt
	cpuQuotaPeriod = 100 * time.Millisecond

	// systemLabelPrefix represents the reserved namespace for system labels.
	systemLabelPrefix = "com.docker.swarm"
)

// containerConfig converts task properties into docker container compatible
// components.
type containerConfig struct {
	task                *api.Task
	networksAttachments map[string]*api.NetworkAttachment
}

// newContainerConfig returns a validated container config. No methods should
// return an error if this function returns without error.
func newContainerConfig(n *api.NodeDescription, t *api.Task) (*containerConfig, error) {
	var c containerConfig
	return &c, c.setTask(n, t)
}

func (c *containerConfig) setTask(n *api.NodeDescription, t *api.Task) error {
	container := t.Spec.GetContainer()
	if container == nil {
		return exec.ErrRuntimeUnsupported
	}

	if container.Image == "" {
		return ErrImageRequired
	}

	// index the networks by name
	c.networksAttachments = make(map[string]*api.NetworkAttachment, len(t.Networks))
	for _, attachment := range t.Networks {
		c.networksAttachments[attachment.Network.Spec.Annotations.Name] = attachment
	}

	c.task = t
	preparedSpec, err := template.ExpandContainerSpec(n, t)
	if err != nil {
		return err
	}
	c.task.Spec.Runtime = &api.TaskSpec_Container{
		Container: preparedSpec,
	}

	return nil
}

func (c *containerConfig) endpoint() *api.Endpoint {
	return c.task.Endpoint
}

func (c *containerConfig) spec() *api.ContainerSpec {
	return c.task.Spec.GetContainer()
}

func (c *containerConfig) name() string {
	return naming.Task(c.task)
}

func (c *containerConfig) image() string {
	return c.spec().Image
}

func portSpec(port uint32, protocol api.PortConfig_Protocol) nat.Port {
	return nat.Port(fmt.Sprintf("%d/%s", port, strings.ToLower(protocol.String())))
}

func (c *containerConfig) portBindings() nat.PortMap {
	portBindings := nat.PortMap{}
	if c.task.Endpoint == nil {
		return portBindings
	}

	for _, portConfig := range c.task.Endpoint.Ports {
		if portConfig.PublishMode != api.PublishModeHost {
			continue
		}

		port := portSpec(portConfig.TargetPort, portConfig.Protocol)
		binding := []nat.PortBinding{
			{},
		}

		if portConfig.PublishedPort != 0 {
			binding[0].HostPort = strconv.Itoa(int(portConfig.PublishedPort))
		}
		portBindings[port] = binding
	}

	return portBindings
}

func (c *containerConfig) isolation() enginecontainer.Isolation {
	switch c.spec().Isolation {
	case api.ContainerIsolationDefault:
		return enginecontainer.Isolation("default")
	case api.ContainerIsolationHyperV:
		return enginecontainer.Isolation("hyperv")
	case api.ContainerIsolationProcess:
		return enginecontainer.Isolation("process")
	}
	return enginecontainer.Isolation("")
}

func (c *containerConfig) exposedPorts() map[nat.Port]struct{} {
	exposedPorts := make(map[nat.Port]struct{})
	if c.task.Endpoint == nil {
		return exposedPorts
	}

	for _, portConfig := range c.task.Endpoint.Ports {
		if portConfig.PublishMode != api.PublishModeHost {
			continue
		}

		port := portSpec(portConfig.TargetPort, portConfig.Protocol)
		exposedPorts[port] = struct{}{}
	}

	return exposedPorts
}

func (c *containerConfig) config() *enginecontainer.Config {
	genericEnvs := genericresource.EnvFormat(c.task.AssignedGenericResources, "DOCKER_RESOURCE")
	env := append(c.spec().Env, genericEnvs...)

	config := &enginecontainer.Config{
		Labels:       c.labels(),
		StopSignal:   c.spec().StopSignal,
		User:         c.spec().User,
		Hostname:     c.spec().Hostname,
		Env:          env,
		WorkingDir:   c.spec().Dir,
		Tty:          c.spec().TTY,
		OpenStdin:    c.spec().OpenStdin,
		Image:        c.image(),
		ExposedPorts: c.exposedPorts(),
		Healthcheck:  c.healthcheck(),
	}

	if len(c.spec().Command) > 0 {
		// If Command is provided, we replace the whole invocation with Command
		// by replacing Entrypoint and specifying Cmd. Args is ignored in this
		// case.
		config.Entrypoint = append(config.Entrypoint, c.spec().Command...)
		config.Cmd = append(config.Cmd, c.spec().Args...)
	} else if len(c.spec().Args) > 0 {
		// In this case, we assume the image has an Entrypoint and Args
		// specifies the arguments for that entrypoint.
		config.Cmd = c.spec().Args
	}

	return config
}

func (c *containerConfig) healthcheck() *enginecontainer.HealthConfig {
	hcSpec := c.spec().Healthcheck
	if hcSpec == nil {
		return nil
	}
	interval, _ := gogotypes.DurationFromProto(hcSpec.Interval)
	timeout, _ := gogotypes.DurationFromProto(hcSpec.Timeout)
	startPeriod, _ := gogotypes.DurationFromProto(hcSpec.StartPeriod)
	return &enginecontainer.HealthConfig{
		Test:        hcSpec.Test,
		Interval:    interval,
		Timeout:     timeout,
		Retries:     int(hcSpec.Retries),
		StartPeriod: startPeriod,
	}
}

func (c *containerConfig) hostConfig() *enginecontainer.HostConfig {
	hc := &enginecontainer.HostConfig{
		Resources:    c.resources(),
		Mounts:       c.mounts(),
		Tmpfs:        c.tmpfs(),
		GroupAdd:     c.spec().Groups,
		PortBindings: c.portBindings(),
		Init:         c.init(),
		Isolation:    c.isolation(),
	}

	// The format of extra hosts on swarmkit is specified in:
	// http://man7.org/linux/man-pages/man5/hosts.5.html
	//    IP_address canonical_hostname [aliases...]
	// However, the format of ExtraHosts in HostConfig is
	//    <host>:<ip>
	// We need to do the conversion here
	// (Alias is ignored for now)
	for _, entry := range c.spec().Hosts {
		parts := strings.Fields(entry)
		if len(parts) > 1 {
			hc.ExtraHosts = append(hc.ExtraHosts, fmt.Sprintf("%s:%s", parts[1], parts[0]))
		}
	}

	if c.task.LogDriver != nil {
		hc.LogConfig = enginecontainer.LogConfig{
			Type:   c.task.LogDriver.Name,
			Config: c.task.LogDriver.Options,
		}
	}

	return hc
}

func (c *containerConfig) labels() map[string]string {
	var (
		system = map[string]string{
			"task":         "", // mark as cluster task
			"task.id":      c.task.ID,
			"task.name":    naming.Task(c.task),
			"node.id":      c.task.NodeID,
			"service.id":   c.task.ServiceID,
			"service.name": c.task.ServiceAnnotations.Name,
		}
		labels = make(map[string]string)
	)

	// base labels are those defined in the spec.
	for k, v := range c.spec().Labels {
		labels[k] = v
	}

	// we then apply the overrides from the task, which may be set via the
	// orchestrator.
	for k, v := range c.task.Annotations.Labels {
		labels[k] = v
	}

	// finally, we apply the system labels, which override all labels.
	for k, v := range system {
		labels[strings.Join([]string{systemLabelPrefix, k}, ".")] = v
	}

	return labels
}

func (c *containerConfig) tmpfs() map[string]string {
	r := make(map[string]string)

	for _, spec := range c.spec().Mounts {
		if spec.Type != api.MountTypeTmpfs {
			continue
		}

		r[spec.Target] = getMountMask(&spec)
	}

	return r
}

func (c *containerConfig) mounts() []enginemount.Mount {
	var r []enginemount.Mount
	for _, mount := range c.spec().Mounts {
		r = append(r, convertMount(mount))
	}
	return r
}

func convertMount(m api.Mount) enginemount.Mount {
	mount := enginemount.Mount{
		Source:   m.Source,
		Target:   m.Target,
		ReadOnly: m.ReadOnly,
	}

	switch m.Type {
	case api.MountTypeBind:
		mount.Type = enginemount.TypeBind
	case api.MountTypeVolume:
		mount.Type = enginemount.TypeVolume
	}

	if m.BindOptions != nil {
		mount.BindOptions = &enginemount.BindOptions{}
		switch m.BindOptions.Propagation {
		case api.MountPropagationRPrivate:
			mount.BindOptions.Propagation = enginemount.PropagationRPrivate
		case api.MountPropagationPrivate:
			mount.BindOptions.Propagation = enginemount.PropagationPrivate
		case api.MountPropagationRSlave:
			mount.BindOptions.Propagation = enginemount.PropagationRSlave
		case api.MountPropagationSlave:
			mount.BindOptions.Propagation = enginemount.PropagationSlave
		case api.MountPropagationRShared:
			mount.BindOptions.Propagation = enginemount.PropagationRShared
		case api.MountPropagationShared:
			mount.BindOptions.Propagation = enginemount.PropagationShared
		}
	}

	if m.VolumeOptions != nil {
		mount.VolumeOptions = &enginemount.VolumeOptions{
			NoCopy: m.VolumeOptions.NoCopy,
		}
		if m.VolumeOptions.Labels != nil {
			mount.VolumeOptions.Labels = make(map[string]string, len(m.VolumeOptions.Labels))
			for k, v := range m.VolumeOptions.Labels {
				mount.VolumeOptions.Labels[k] = v
			}
		}
		if m.VolumeOptions.DriverConfig != nil {
			mount.VolumeOptions.DriverConfig = &enginemount.Driver{
				Name: m.VolumeOptions.DriverConfig.Name,
			}
			if m.VolumeOptions.DriverConfig.Options != nil {
				mount.VolumeOptions.DriverConfig.Options = make(map[string]string, len(m.VolumeOptions.DriverConfig.Options))
				for k, v := range m.VolumeOptions.DriverConfig.Options {
					mount.VolumeOptions.DriverConfig.Options[k] = v
				}
			}
		}
	}
	return mount
}

func getMountMask(m *api.Mount) string {
	var maskOpts []string
	if m.ReadOnly {
		maskOpts = append(maskOpts, "ro")
	}

	switch m.Type {
	case api.MountTypeTmpfs:
		if m.TmpfsOptions == nil {
			break
		}

		if m.TmpfsOptions.Mode != 0 {
			maskOpts = append(maskOpts, fmt.Sprintf("mode=%o", m.TmpfsOptions.Mode))
		}

		if m.TmpfsOptions.SizeBytes != 0 {
			// calculate suffix here, making this linux specific, but that is
			// okay, since API is that way anyways.

			// we do this by finding the suffix that divides evenly into the
			// value, returning the value itself, with no suffix, if it fails.
			//
			// For the most part, we don't enforce any semantic to this values.
			// The operating system will usually align this and enforce minimum
			// and maximums.
			var (
				size   = m.TmpfsOptions.SizeBytes
				suffix string
			)
			for _, r := range []struct {
				suffix  string
				divisor int64
			}{
				{"g", 1 << 30},
				{"m", 1 << 20},
				{"k", 1 << 10},
			} {
				if size%r.divisor == 0 {
					size = size / r.divisor
					suffix = r.suffix
					break
				}
			}

			maskOpts = append(maskOpts, fmt.Sprintf("size=%d%s", size, suffix))
		}
	}

	return strings.Join(maskOpts, ",")
}

// This handles the case of volumes that are defined inside a service Mount
func (c *containerConfig) volumeCreateRequest(mount *api.Mount) *volume.VolumesCreateBody {
	var (
		driverName string
		driverOpts map[string]string
		labels     map[string]string
	)

	if mount.VolumeOptions != nil && mount.VolumeOptions.DriverConfig != nil {
		driverName = mount.VolumeOptions.DriverConfig.Name
		driverOpts = mount.VolumeOptions.DriverConfig.Options
		labels = mount.VolumeOptions.Labels
	}

	return &volume.VolumesCreateBody{
		Name:       mount.Source,
		Driver:     driverName,
		DriverOpts: driverOpts,
		Labels:     labels,
	}
}

func (c *containerConfig) resources() enginecontainer.Resources {
	resources := enginecontainer.Resources{}

	// set pids limit
	pidsLimit := c.spec().PidsLimit
	if pidsLimit > 0 {
		resources.PidsLimit = pidsLimit
	}

	// If no limits are specified let the engine use its defaults.
	//
	// TODO(aluzzardi): We might want to set some limits anyway otherwise
	// "unlimited" tasks will step over the reservation of other tasks.
	r := c.task.Spec.Resources
	if r == nil || r.Limits == nil {
		return resources
	}

	if r.Limits.MemoryBytes > 0 {
		resources.Memory = r.Limits.MemoryBytes
	}

	if r.Limits.NanoCPUs > 0 {
		// CPU Period must be set in microseconds.
		resources.CPUPeriod = int64(cpuQuotaPeriod / time.Microsecond)
		resources.CPUQuota = r.Limits.NanoCPUs * resources.CPUPeriod / 1e9
	}

	return resources
}

func (c *containerConfig) virtualIP(networkID string) string {
	if c.task.Endpoint == nil {
		return ""
	}

	for _, vip := range c.task.Endpoint.VirtualIPs {
		// We only support IPv4 VIPs for now.
		if vip.NetworkID == networkID {
			vip, _, err := net.ParseCIDR(vip.Addr)
			if err != nil {
				return ""
			}

			return vip.String()
		}
	}

	return ""
}

func (c *containerConfig) networkingConfig() *network.NetworkingConfig {
	epConfig := make(map[string]*network.EndpointSettings)
	for _, na := range c.task.Networks {
		var ipv4, ipv6 string
		for _, addr := range na.Addresses {
			ip, _, err := net.ParseCIDR(addr)
			if err != nil {
				continue
			}

			if ip.To4() != nil {
				ipv4 = ip.String()
				continue
			}

			if ip.To16() != nil {
				ipv6 = ip.String()
			}
		}

		epSettings := &network.EndpointSettings{
			IPAMConfig: &network.EndpointIPAMConfig{
				IPv4Address: ipv4,
				IPv6Address: ipv6,
			},
		}

		epConfig[na.Network.Spec.Annotations.Name] = epSettings
	}

	return &network.NetworkingConfig{EndpointsConfig: epConfig}
}

// networks returns a list of network names attached to the container. The
// returned name can be used to lookup the corresponding network create
// options.
func (c *containerConfig) networks() []string {
	var networks []string

	for name := range c.networksAttachments {
		networks = append(networks, name)
	}

	return networks
}

func (c *containerConfig) networkCreateOptions(name string) (types.NetworkCreate, error) {
	na, ok := c.networksAttachments[name]
	if !ok {
		return types.NetworkCreate{}, errors.New("container: unknown network referenced")
	}

	options := types.NetworkCreate{
		Driver: na.Network.DriverState.Name,
		IPAM: &network.IPAM{
			Driver: na.Network.IPAM.Driver.Name,
		},
		Options:        na.Network.DriverState.Options,
		CheckDuplicate: true,
	}

	for _, ic := range na.Network.IPAM.Configs {
		c := network.IPAMConfig{
			Subnet:  ic.Subnet,
			IPRange: ic.Range,
			Gateway: ic.Gateway,
		}
		options.IPAM.Config = append(options.IPAM.Config, c)
	}

	return options, nil
}

func (c containerConfig) eventFilter() filters.Args {
	filter := filters.NewArgs()
	filter.Add("type", events.ContainerEventType)
	filter.Add("name", c.name())
	filter.Add("label", fmt.Sprintf("%v.task.id=%v", systemLabelPrefix, c.task.ID))
	return filter
}

func (c *containerConfig) init() *bool {
	if c.spec().Init != nil {
		return &c.spec().Init.Value
	}
	return nil
}
