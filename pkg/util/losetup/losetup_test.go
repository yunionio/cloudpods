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

package losetup

import (
	"reflect"
	"testing"
)

func Test_parseJsonOutput(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Devices
		wantErr bool
	}{
		{
			name: "normal input",
			content: `{
   "loopdevices": [
      {
         "name": "/dev/loop1",
         "sizelimit": 0,
         "offset": 0,
         "autoclear": true,
         "ro": false,
         "back-file": "/opt/cloud/workspace/disks/recycle_bin/20240911160650/194767a9-556f-4072-8a20-7ab1086f22ff.1726070810",
         "dio": false,
         "log-sec": 512
      },{
         "name": "/dev/loop57",
         "sizelimit": 0,
         "offset": 0,
         "autoclear": false,
         "ro": false,
         "back-file": "/opt/cloud/workspace/disks/2c10325f-7109-4cf0-8a9f-4c3724920245 (deleted)",
         "dio": false,
         "log-sec": 512
      }]}`,
			want: &Devices{
				LoopDevs: []Device{
					{
						Name:      "/dev/loop1",
						BackFile:  "/opt/cloud/workspace/disks/recycle_bin/20240911160650/194767a9-556f-4072-8a20-7ab1086f22ff.1726070810",
						SizeLimit: false,
						ReadOnly:  false,
					},
					{
						Name:      "/dev/loop57",
						BackFile:  "/opt/cloud/workspace/disks/2c10325f-7109-4cf0-8a9f-4c3724920245 (deleted)",
						SizeLimit: false,
						ReadOnly:  false,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJsonOutput(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJsonOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJsonOutput() got = %v, want %v", got, tt.want)
			}
		})
	}
}
