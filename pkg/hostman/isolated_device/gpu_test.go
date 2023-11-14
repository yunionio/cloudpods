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

package isolated_device

import (
	"reflect"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func Test_parsePCIELinkCap(t *testing.T) {
	tests := []struct {
		line    string
		want    *api.IsolatedDevicePCIEInfo
		wantErr bool
	}{
		{
			line: `LnkCap: Port #0, Speed 2.5GT/s, Width x16, ASPM L0s L1, Exit Latency L0s <1us, L1 <4us`,
			want: &api.IsolatedDevicePCIEInfo{
				TransferRatePerLane: "2.5GT/s",
				LaneWidth:           16,
				Version:             api.PCIEVersion1,
				Throughput:          "4.00 GB/s",
			},
			wantErr: false,
		},
		{
			line: `                LnkCap: Port #0, Speed 8GT/s, Width x16, ASPM L0s L1, Exit Latency L0s <1us, L1 <4us`,
			want: &api.IsolatedDevicePCIEInfo{
				TransferRatePerLane: "8GT/s",
				LaneWidth:           16,
				Version:             api.PCIEVersion3,
				Throughput:          "15.76 GB/s",
			},
			wantErr: false,
		},
		{
			line: `                LnkCap: Port #0, Speed 16GT/s, Width x4, ASPM L0s L1, Exit Latency L0s <1us, L1 <4us`,
			want: &api.IsolatedDevicePCIEInfo{
				TransferRatePerLane: "16GT/s",
				LaneWidth:           4,
				Version:             api.PCIEVersion4,
				Throughput:          "7.88 GB/s",
			},
			wantErr: false,
		},
		{
			line:    ``,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := parsePCIELinkCap(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePCIELinkCap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePCIELinkCap() = %v, want %v", got, tt.want)
			}
		})
	}
}
