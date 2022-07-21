package chrootuser

import (
	"fmt"
	"os/user"
	"strings"
)

// GetUser will return the uid, gid of the user specified in the userStr
// it will use the /etc/passwd and /etc/group files inside of the rootdir
// to return this information.
// userStr format [user | user:group | uid | uid:gid | user:gid | uid:group ]
func GetUser(rootdir string, userStr string) (*user.User, error) {
	spec := strings.SplitN(userStr, ":", 2)
	userStr = spec[0]
	groupspec := ""

	if userStr == "" {
		userStr = "0"
	}

	if len(spec) > 1 {
		groupspec = spec[1]
	}

	uid64, gid64, err := lookupUserInContainer(rootdir, userStr)
	if err != nil {
		return nil, err
	}

	if groupspec != "" {
		gid64, err = lookupGroupInContainer(rootdir, groupspec)
		if err != nil {
			return nil, err
		}
	}

	homedir, err := lookupHomedirInContainer(rootdir, uid64)
	if err != nil {
		homedir = "/"
	}
	return &user.User{
		Uid:      fmt.Sprint(uid64),
		Gid:      fmt.Sprint(gid64),
		HomeDir:  homedir,
		Username: userStr,
	}, nil
}

// GetGroup returns the gid by looking it up in the /etc/group file
// groupspec format [ group | gid ]
func GetGroup(rootdir, groupspec string) (*user.Group, error) {
	gid, err := lookupGroupInContainer(rootdir, groupspec)
	if err != nil {
		return nil, err
	}
	return &user.Group{
		Gid:  fmt.Sprint(gid),
		Name: groupspec,
	}, nil
}
