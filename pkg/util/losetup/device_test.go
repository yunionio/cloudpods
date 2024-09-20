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

var (
	losetupOutput string = `
NAME       BACK-FILE                                  
/dev/loop0 /disks/2b917686-2ace-4a57-a4af-44ece2303dd2 (deleted)
/dev/loop1 /disks/033d6bc0-4ce4-48c4-89d3-125077bcc28e
`
)

func Test_parseDevices(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name    string
		args    args
		want    *Devices
		wantErr bool
	}{
		{
			name: "normalOutput",
			args: args{losetupOutput},
			want: &Devices{
				[]Device{
					{
						Name:     "/dev/loop0",
						BackFile: "/disks/2b917686-2ace-4a57-a4af-44ece2303dd2 (deleted)",
					},
					{
						Name:     "/dev/loop1",
						BackFile: "/disks/033d6bc0-4ce4-48c4-89d3-125077bcc28e",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDevices(tt.args.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDevices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDevices() = %v, want %v", got, tt.want)
			}
		})
	}
}
