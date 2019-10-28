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

package cmdline

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
)

func TestFetchServerConfigsByJSON(t *testing.T) {
	type args struct {
		obj jsonutils.JSONObject
	}
	tests := []struct {
		name    string
		args    args
		want    *compute.ServerConfigs
		wantErr bool
	}{
		{
			name: "all configs",
			args: args{jsonutils.Marshal(map[string]string{
				"disk.0":                  "1g:centos",
				"net.0":                   "192.168.222.3:inf0",
				"net.1":                   "[random]",
				"schedtag.0":              "ssd:require",
				"schedtag.1":              "container:exclude",
				"isolated_device.0":       "vendor=NVIDIA:GeForce GTX 1050 Ti",
				"baremetal_disk_config.0": "raid0:[1,2]",
			})},
			want: &compute.ServerConfigs{
				Disks: []*compute.DiskConfig{
					{
						Index:   0,
						SizeMb:  1024,
						ImageId: "centos",
						Medium:  "hybrid",
					},
				},
				Networks: []*compute.NetworkConfig{
					{
						Network: "inf0",
						Address: "192.168.222.3",
					},
					{
						Index: 1,
						Exit:  false,
					},
				},
				Schedtags: []*compute.SchedtagConfig{
					{
						Id:       "ssd",
						Strategy: "require",
					},
					{
						Id:       "container",
						Strategy: "exclude",
					},
				},
				IsolatedDevices: []*compute.IsolatedDeviceConfig{
					{
						Vendor: "NVIDIA",
						Model:  "GeForce GTX 1050 Ti",
					},
				},
				BaremetalDiskConfigs: []*compute.BaremetalDiskConfig{
					{
						Type:  "hybrid",
						Conf:  "raid0",
						Range: []int64{1, 2},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchServerConfigsByJSON(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchServerConfigsByJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotJ := jsonutils.Marshal(got).String()
			wantJ := jsonutils.Marshal(tt.want).String()
			if !reflect.DeepEqual(gotJ, wantJ) {
				t.Errorf("FetchServerConfigsByJSON() = %v, want %v", gotJ, wantJ)
			}
		})
	}
}
