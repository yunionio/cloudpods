package fsdriver

import (
	"testing"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
)

func TestMergeAuthorizedKeys(t *testing.T) {
	type args struct {
		oldKeys string
		pubkeys *sshkeys.SSHKeys
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "MergeAuthorizedKeys",
			args: args{
				oldKeys: "Test KEY",
				pubkeys: &sshkeys.SSHKeys{},
			},
			want: "KEY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeAuthorizedKeys(tt.args.oldKeys, tt.args.pubkeys); got != tt.want {
				t.Errorf("MergeAuthorizedKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}
