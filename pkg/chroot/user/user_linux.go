//go:build linux
// +build linux

package chrootuser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
)

var (
	lookupUser, lookupGroup sync.Mutex
	// override for testing
	openChrootedFileFunc = openChrootedFile
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

func lookupUserInContainer(rootdir, userStr string) (uid uint64, gid uint64, err error) {
	r, err := openChrootedFileFunc(rootdir, "/etc/passwd")
	if err != nil {
		return 0, 0, err
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
		return pwd.uid, pwd.gid, nil
	}

	return 0, 0, user.UnknownUserError(fmt.Sprintf("error looking up user %q", userStr))
}

func lookupGroupInContainer(rootdir, groupname string) (gid uint64, err error) {
	r, err := openChrootedFileFunc(rootdir, "/etc/group")
	if err != nil {
		return 0, err
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
		return grp.gid, nil
	}

	return 0, user.UnknownGroupError(fmt.Sprintf("error looking up group %q", groupname))
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

	return "", user.UnknownUserError(fmt.Sprintf("error looking up uid %q for homedir", uid))
}

func openChrootedFile(rootDir string, file string) (io.ReadCloser, error) {
	return os.OpenFile(rootDir, os.O_RDONLY, 0)
}
