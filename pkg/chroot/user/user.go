package chrootuser

import (
	"fmt"
	"os/user"
	"strings"
)

type lookupPasswdEntry struct {
	name string
	uid  uint64
	gid  uint64
	home string
}
type lookupGroupEntry struct {
	name string
	gid  uint64
	user string
}

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

	userEntry, err := lookupUserInContainer(rootdir, userStr)
	if err != nil {
		return nil, err
	}

	var groupEntry *lookupGroupEntry
	if groupspec != "" {
		groupEntry, err = lookupGroupInContainer(rootdir, groupspec)
		if err != nil {
			return nil, err
		}
	}

	homedir, err := lookupHomedirInContainer(rootdir, userEntry.uid)
	if err != nil {
		homedir = "/"
	}
	user := &user.User{
		Uid:      fmt.Sprint(userEntry.uid),
		Gid:      fmt.Sprint(userEntry.gid),
		HomeDir:  homedir,
		Username: userStr,
	}
	if groupEntry != nil {
		user.Gid = fmt.Sprint(groupEntry.gid)
	}
	return user, nil
}

func GetAdditionalGroupIDs(rootdir string, user *user.User) ([]string, error) {
  gids, err := lookupAdditionalGroupsForUser(rootdir, user)
	if err != nil {
		return nil, err
	}
	gidsStr := make([]string, len(gids))
	for _, gid := range gids {
		gidsStr = append(gidsStr, fmt.Sprint(gid))
	}
  return gidsStr, nil
}

// GetGroup returns the gid by looking it up in the /etc/group file
// groupspec format [ group | gid ]
func GetGroup(rootdir, groupspec string) (*user.Group, error) {
	group, err := lookupGroupInContainer(rootdir, groupspec)
	if err != nil {
		return nil, err
	}
	return &user.Group{
		Gid:  fmt.Sprint(group.gid),
		Name: group.name,
	}, nil
}
