/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"strconv"
	"syscall"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

func TestSyscallCredentials(t *testing.T) {
	currentUser := testutil.GetCurrentUser(t)
	uid, _ := strconv.ParseUint(currentUser.Uid, 10, 32)
	currentUserUid32 := uint32(uid)
	gid, _ := strconv.ParseUint(currentUser.Gid, 10, 32)
	currentUserGid32 := uint32(gid)

	type args struct {
		userStr string
	}
	tests := []struct {
		name    string
		args    args
		want    *syscall.Credential
		wantErr bool
	}{
		{
			name: "non-existing user without group",
			args: args{
				userStr: "helloworld-user",
			},
			wantErr: true,
		},
		{
			name: "non-existing uid without group",
			args: args{
				userStr: "1001",
			},
			want: &syscall.Credential{
				Uid: 1001,
				// because fallback is enabled
				Gid:    1001,
				Groups: []uint32{},
			},
		},
		{
			name: "non-existing uid with existing gid",
			args: args{
				userStr: fmt.Sprintf("1001:%d", currentUserGid32),
			},
			want: &syscall.Credential{
				Uid:    1001,
				Gid:    currentUserGid32,
				Groups: []uint32{},
			},
		},
		{
			name: "existing username with non-existing gid",
			args: args{
				userStr: fmt.Sprintf("%s:50000", currentUser.Username),
			},
			want: &syscall.Credential{
				Uid:    currentUserUid32,
				Gid:    50000,
				Groups: []uint32{currentUserGid32},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SyscallCredentials(tt.args.userStr)
			testutil.CheckErrorAndDeepEqual(t, tt.wantErr, err, tt.want, got)
		})
	}
}
