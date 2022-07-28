//go:build linux
// +build linux

package idtools

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_getMappingFromSubFile(t *testing.T) {
	type args struct {
		uidOrGid      uint32
		userOrGroup   string
		idFileContent string
	}
	tests := []struct {
		name    string
		args    args
		want    []Mapping
		wantErr bool
	}{
		{
			name: "default id file",
			args: args{
				uidOrGid:      0,
				userOrGroup:   "foo",
				idFileContent: sampleIDContent,
			},
			want: []Mapping{
				{
					ContainerID: 100000,
					HostID:      0,
					Size:        65536,
				},
			},
			wantErr: false,
		},
		{
			name: "user not in file",
			args: args{
				uidOrGid:      0,
				userOrGroup:   "baz",
				idFileContent: sampleIDContent,
			},
			wantErr: false,
			want:    []Mapping{},
		},
		{
			name: "malformed id file",
			args: args{
				uidOrGid:      0,
				userOrGroup:   "foo",
				idFileContent: malformedIDContent,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(tt.args.idFileContent))
			got, err := getMappingFromSubFile(tt.args.uidOrGid, tt.args.userOrGroup, reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMappingFromSubFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMappingFromSubFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

var sampleIDContent = `
foo:100000:65536
`

var malformedIDContent = `
:100000:::65536
`
