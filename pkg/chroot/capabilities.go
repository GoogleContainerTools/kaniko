package chroot

import (
	"fmt"

	"github.com/syndtr/gocapability/capability"
)

// defaultCapabilities returns a Linux kernel default capabilities
var defaultCapabilities = []capability.Cap{
	capability.CAP_CHOWN,
	capability.CAP_DAC_OVERRIDE,
	capability.CAP_FSETID,
	capability.CAP_FOWNER,
	capability.CAP_MKNOD,
	capability.CAP_NET_RAW,
	capability.CAP_SETGID,
	capability.CAP_SETUID,
	capability.CAP_SETFCAP,
	capability.CAP_SETPCAP,
	capability.CAP_NET_BIND_SERVICE,
	capability.CAP_KILL,
	capability.CAP_AUDIT_WRITE,
}

// setCapabilities sets capabilities for ourselves, to be more or less inherited by any processes that we'll start.
func setCapabilities() error {
	caps, err := capability.NewPid2(0)
	if err != nil {
		return err
	}
	capMap := map[capability.CapType][]capability.Cap{
		capability.BOUNDING:    defaultCapabilities,
		capability.EFFECTIVE:   defaultCapabilities,
		capability.INHERITABLE: {},
		capability.PERMITTED:   defaultCapabilities,
	}
	for capType, capList := range capMap {
		caps.Set(capType, capList...)
	}
	err = caps.Apply(capability.CAPS | capability.BOUNDS | capability.AMBS)
	if err != nil {
		return fmt.Errorf("applying capabiliies: %w", err)
	}
	return nil
}
