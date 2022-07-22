//go:build linux
// +build linux

package chrootuser

import (
	"bufio"
	"bytes"
	"io"
	"os/user"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func Test_parseNextPasswd(t *testing.T) {
	tests := []struct {
		name   string
		reader io.Reader
		want   *lookupPasswdEntry
	}{
		{
			name: "existing user",
			want: &lookupPasswdEntry{
				name: "testuser",
				uid:  1000,
				gid:  1000,
				home: "/home/test",
			},
			reader: bytes.NewReader([]byte(passwd)),
		},
		{
			name:   "malformed passwd",
			want:   nil,
			reader: bytes.NewReader([]byte(malformedPasswd)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := bufio.NewScanner(tt.reader)
			got := parseNextPasswd(rc)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("wanted %#v, but got %#v", tt.want, got)
			}
		})
	}
}

func Test_parseNextGroup(t *testing.T) {
	tests := []struct {
		name   string
		want   *lookupGroupEntry
		reader io.Reader
	}{
		{
			name: "test group",
			want: &lookupGroupEntry{
				name: "bar",
				gid:  2001,
				user: "testuser,foo",
			},
			reader: bytes.NewReader([]byte(group)),
		},
		{
			name:   "malformed gid",
			want:   nil,
			reader: bytes.NewReader([]byte(malformedGroups)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := bufio.NewScanner(tt.reader)
			got := parseNextGroup(rc)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("wanted %#v, but got %#v", tt.want, got)
			}
		})
	}
}

func Test_lookupUserInContainer(t *testing.T) {
	type args struct {
		userStr string
	}
	tests := []struct {
		name     string
		args     args
		wantUser *lookupPasswdEntry
		wantErr  bool
	}{
		{
			name: "existing user",
			args: args{
				userStr: "foo",
			},
			wantUser: &lookupPasswdEntry{
				uid:  2000,
				gid:  2000,
				name: "foo",
				home: "/home/foo",
			},
			wantErr: false,
		},
		{
			name: "non existing user",
			args: args{
				userStr: "baz",
			},
			wantErr: true,
		},
	}
	original := openChrootedFileFunc
	openChrootedFileFunc = openPasswd
	defer func() {
		openChrootedFileFunc = original
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, err := lookupUserInContainer("", tt.args.userStr)
			testutil.CheckErrorAndDeepEqual(t, tt.wantErr, err, tt.wantUser, gotUser)
		})
	}
}

func Test_lookupGroupInContainer(t *testing.T) {
	type args struct {
		groupname string
	}
	tests := []struct {
		name           string
		args           args
		wantGroupEntry *lookupGroupEntry
		wantErr        bool
	}{
		{
			name: "existing group",
			args: args{
				groupname: "foo",
			},
			wantGroupEntry: &lookupGroupEntry{
				name: "foo",
				gid:  2000,
				user: "1000",
			},
			wantErr: false,
		},
		{
			name: "non existing group",
			args: args{
				groupname: "no group",
			},
			wantErr: true,
		},
	}
	original := openChrootedFileFunc
	openChrootedFileFunc = openGroup
	defer func() {
		openChrootedFileFunc = original
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGroupEntry, err := lookupGroupInContainer("", tt.args.groupname)
			testutil.CheckErrorAndDeepEqual(t, tt.wantErr, err, gotGroupEntry, tt.wantGroupEntry)
		})
	}
}

func Test_lookupHomedirInContainer(t *testing.T) {
	type args struct {
		uid uint64
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "existing user",
			args: args{
				uid: 1000,
			},
			want:    "/home/test",
			wantErr: false,
		},
		{
			name: "non existing user",
			args: args{
				uid: 0,
			},
			want:    "",
			wantErr: true,
		},
	}
	original := openChrootedFileFunc
	openChrootedFileFunc = openPasswd
	defer func() {
		openChrootedFileFunc = original
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lookupHomedirInContainer("", tt.args.uid)
			testutil.CheckErrorAndDeepEqual(t, tt.wantErr, err, got, tt.want)
		})
	}
}

func Test_lookupAdditionalGroupsForUser(t *testing.T) {
	type args struct {
		user *user.User
	}
	tests := []struct {
		name     string
		args     args
		wantGids []uint32
		wantErr  bool
	}{
		{
			name: "user with uid and name in groups",
			args: args{
				user: &user.User{
					Uid:      "1000",
					Username: "testuser",
				},
			},
			wantGids: []uint32{2001, 2000},
			wantErr:  false,
		},
		{
			name: "user with no additional groups",
			args: args{
				user: &user.User{
					Uid:      "2001",
					Username: "bar",
				},
			},
			wantGids: []uint32{},
			wantErr:  false,
		},
	}
	original := openChrootedFileFunc
	openChrootedFileFunc = openGroup
	defer func() {
		openChrootedFileFunc = original
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGids, err := lookupAdditionalGroupsForUser("", tt.args.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("lookupAdditionalGroupsForUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotGids, tt.wantGids) {
				t.Errorf("lookupAdditionalGroupsForUser() = %v, want %v", gotGids, tt.wantGids)
			}
		})
	}
}

var passwd = `testuser:x:1000:1000:I am test:/home/test:/bin/zsh
foo:x:2000:2000:I am foo:/home/foo:/bin/zsh
bar:x:2001:2001:I am bar:/home/bar:/bin/zsh
`
var malformedPasswd = `bar:x:awdjawdj:foo`

func openPasswd(rootDir string, file string) (io.ReadCloser, error) {
	r := bytes.NewReader([]byte(passwd))
	return io.NopCloser(r), nil
}

var group = `bar:x:2001:testuser,foo
foo:x:2000:1000
test:x:1000:
`

var malformedGroups = `bar:x:awdjawdj:foo`

func openGroup(rootDir string, file string) (io.ReadCloser, error) {
	r := bytes.NewReader([]byte(group))
	return io.NopCloser(r), nil
}
