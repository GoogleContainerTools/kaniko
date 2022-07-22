//go:build linux
// +build linux

package chrootuser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

)

var (
	lookupUser, lookupGroup sync.Mutex
	// override for testing
	openChrootedFileFunc = openChrootedFile
)

func parseNextPasswd(rc *bufio.Scanner) *lookupPasswdEntry {
	if !rc.Scan() {
		return nil
	}
	line := rc.Text()
	fields := strings.Split(line, ":")
	if len(fields) != 7 {
		return nil
	}
	uid, err := strconv.ParseUint(fields[2], 10, 32)
	if err != nil {
		return nil
	}
	gid, err := strconv.ParseUint(fields[3], 10, 32)
	if err != nil {
		return nil
	}
	return &lookupPasswdEntry{
		name: fields[0],
		uid:  uid,
		gid:  gid,
		home: fields[5],
	}
}

func parseNextGroup(rc *bufio.Scanner) *lookupGroupEntry {
	if !rc.Scan() {
		return nil
	}
	line := rc.Text()
	fields := strings.Split(line, ":")
	if len(fields) != 4 {
		return nil
	}
	gid, err := strconv.ParseUint(fields[2], 10, 32)
	if err != nil {
		return nil
	}
	return &lookupGroupEntry{
		name: fields[0],
		gid:  gid,
		user: fields[3],
	}
}

func lookupUserInContainer(rootdir, userStr string) (*lookupPasswdEntry, error) {
	r, err := openChrootedFileFunc(rootdir, "/etc/passwd")
	if err != nil {
		return nil, err
	}
	rc := bufio.NewScanner(r)
	defer r.Close()

	lookupUser.Lock()
	defer lookupUser.Unlock()

	pwd := parseNextPasswd(rc)
	for pwd != nil {
		// check name and uid match
		if pwd.name != userStr {
			if fmt.Sprint(pwd.uid) != userStr {
				{
					pwd = parseNextPasswd(rc)
					continue
				}
			}
		}
		return pwd, nil
	}

	return nil, user.UnknownUserError(userStr)
}

func lookupGroupInContainer(rootdir, groupname string) (*lookupGroupEntry, error) {
	r, err := openChrootedFileFunc(rootdir, "/etc/group")
	if err != nil {
		return nil, err
	}
	rc := bufio.NewScanner(r)
	defer r.Close()

	lookupGroup.Lock()
	defer lookupGroup.Unlock()

	grp := parseNextGroup(rc)
	for grp != nil {
		if grp.name != groupname {
			grp = parseNextGroup(rc)
			continue
		}
		return grp, nil
	}

	return nil, user.UnknownGroupError(groupname)
}

func lookupAdditionalGroupsForUser(rootdir string, user *user.User) (gids []uint32, err error) {
	r, err := openChrootedFileFunc(rootdir, "/etc/passwd")
	if err != nil {
		return nil, err
	}
	rc := bufio.NewScanner(r)
	defer r.Close()

	lookupGroup.Lock()
	defer lookupGroup.Unlock()

	grp := parseNextGroup(rc)
	for grp != nil {
		if strings.Contains(grp.user, user.Username) || strings.Contains(grp.user, user.Uid) {
			gids = append(gids, uint32(grp.gid))
		}
		grp = parseNextGroup(rc)
	}
	return gids, nil
}

func lookupHomedirInContainer(rootdir string, uid uint64) (string, error) {
	r, err := openChrootedFileFunc(rootdir, "/etc/passwd")
	if err != nil {
		return "", err
	}
	rc := bufio.NewScanner(r)
	defer r.Close()

	lookupUser.Lock()
	defer lookupUser.Unlock()

	pwd := parseNextPasswd(rc)
	for pwd != nil {
		if pwd.uid != uid {
			pwd = parseNextPasswd(rc)
			continue
		}
		return pwd.home, nil
	}

	return "", user.UnknownUserError(fmt.Sprint(uid))
}

func openChrootedFile(rootDir string, file string) (io.ReadCloser, error) {
	absFile := filepath.Join(rootDir, file)
	return os.OpenFile(absFile, os.O_RDONLY, 0)
}
