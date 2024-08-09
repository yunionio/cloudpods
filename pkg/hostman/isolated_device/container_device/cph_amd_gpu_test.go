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

package container_device

import "testing"

func Test_getCphAMDGPUPCIAddr(t *testing.T) {
	tests := []struct {
		linkPartName string
		want         string
		wantErr      bool
	}{
		{
			linkPartName: "pci-0000:03:00.0-card",
			want:         "0000:03:00.0",
			wantErr:      false,
		},
		{
			linkPartName: "",
			want:         "",
			wantErr:      true,
		},
		{
			linkPartName: "pci-0000:83:00.0-render",
			want:         "0000:83:00.0",
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.linkPartName, func(t *testing.T) {
			got, err := getGPUPCIAddr(tt.linkPartName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCphAMDGPUPCIAddr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCphAMDGPUPCIAddr() got = %v, want %v", got, tt.want)
			}
		})
	}
}
