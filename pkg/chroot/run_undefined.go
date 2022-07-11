//go:build !linux
// +build !linux

package chroot

import "errors"

func Chroot(newRoot string, additionalMounts ...string) (func() error, error) {
	return func() error { return nil }, errors.New("chroot is only defined when building linux")
}
