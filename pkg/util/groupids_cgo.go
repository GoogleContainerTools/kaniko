// +build linux darwin
// +build cgo

package util

import (
	"os/user"
)

// groupIDs returns all of the group ID's a user is a member of
func groupIDs(u *user.User) ([]string, error) {
	return u.GroupIds()
}
