// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fsdriver

import (
	"testing"

	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

func TestMergeAuthorizedKeys(t *testing.T) {
	type args struct {
		oldKeys string
		pubkeys *deployapi.SSHKeys
	}
	tests := []struct {
		name  string
		args  args
		admin bool
		want  string
	}{
		{
			name: "MergeAuthorizedKeys",
			args: args{
				oldKeys: "ssh-rsa KEY",
				pubkeys: &deployapi.SSHKeys{},
			},
			admin: true,
			want:  "ssh-rsa KEY\n",
		},
		{
			name: "MergeAuthorizedKeys",
			args: args{
				oldKeys: "ssh-rsa KEY " + sshKeySignature,
				pubkeys: &deployapi.SSHKeys{},
			},
			admin: true,
			want:  "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeAuthorizedKeys(tt.args.oldKeys, tt.args.pubkeys, tt.admin); got != tt.want {
				t.Errorf("MergeAuthorizedKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}
