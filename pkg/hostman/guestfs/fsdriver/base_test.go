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
