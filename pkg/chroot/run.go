//go:build !linux
// +build !linux

package chroot

import "errors"

func PrepareMounts(newRoot string, additionalMounts ...string) (undoMount func() error, err error) {
	return func() error { return nil }, errors.New("PrepareMounts is only defined when building linux")
}
