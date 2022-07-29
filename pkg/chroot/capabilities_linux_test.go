//go:build linux
// +build linux

package chroot

import (
	"os"
	"runtime"
	"testing"

	"github.com/syndtr/gocapability/capability"
)

func Test_setCapabilities(t *testing.T) {
	test := struct {
		name    string
		wanted  map[capability.CapType][]capability.Cap
		wantErr bool
	}{
		name: "default applied capabilities",
		wanted: map[capability.CapType][]capability.Cap{
			capability.BOUNDING:    defaultCapabilities,
			capability.EFFECTIVE:   defaultCapabilities,
			capability.INHERITABLE: {},
			capability.PERMITTED:   defaultCapabilities,
		},
		wantErr: false,
	}
	if os.Getuid() != 0 {
		t.Skip("calling user is not root, so can't load caps")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := setCapabilities(); (err != nil) != test.wantErr {
		t.Fatalf("setCapabilities() error = %v, wantErr %v", err, test.wantErr)
	}
	// load the current caps
	caps, err := capability.NewPid2(0)
	if err != nil {
		t.Fatal(err)
	}
	err = caps.Load()
	if err != nil {
		t.Fatal(err)
	}
	for capType, capList := range test.wanted {
		for _, cap := range capList {
			if !caps.Get(capType, cap) {
				t.Errorf("cap %s on capType %s is not set but wanted", cap, capType)
			}
		}
	}
	t.Logf(caps.String())
}
