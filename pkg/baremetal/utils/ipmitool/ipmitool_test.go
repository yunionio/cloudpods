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

package ipmitool

import (
	"reflect"
	"testing"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func TestIpmitoolOutputAcceptable(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{
			name: "empty",
			out:  "",
			want: false,
		},
		{
			name: "valid lan print with exit 1 style output",
			out: `Set in Progress         : Set Complete
IP Address Source       : Static Address
IP Address              : 10.127.223.102
Subnet Mask             : 255.255.255.0
MAC Address             : aa:bb:cc:dd:ee:ff
Default Gateway IP      : 10.127.223.1`,
			want: true,
		},
		{
			name: "auth failure",
			out:  "Error: Unable to establish IPMI v2 / RMCP+ session\n",
			want: false,
		},
		{
			name: "invalid channel",
			out:  "Get Channel Info command failed\nInvalid channel: 8\n",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipmitoolOutputAcceptable([]byte(tt.out))
			if got != tt.want {
				t.Errorf("ipmitoolOutputAcceptable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSysInfo(t *testing.T) {
	type args struct {
		exector IPMIExecutor
	}
	tests := []struct {
		name    string
		args    args
		want    *types.SSystemInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSysInfo(tt.args.exector)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSysInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSysInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
