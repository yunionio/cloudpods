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

package compute

import (
	"fmt"
	"testing"
)

func TestSIsolatedDevicePCIInfo_GetThroughputPerLane(t *testing.T) {
	type fields struct {
		TransferRatePerLane string
		LaneWidth           int
	}
	tests := []struct {
		fields fields
		want   float64
	}{
		{
			fields{
				"2.5GT/s",
				1,
			},
			0.25,
		},
		{
			fields{
				"2.6GT/s",
				1,
			},
			-1,
		},
		{
			fields{
				"5.0GT/s",
				2,
			},
			0.5,
		},
		{
			fields{
				"5GT/s",
				2,
			},
			0.5,
		},
		{
			fields{
				"8GT/s",
				1,
			},
			0.985,
		},
		{
			fields{
				"16GT/s",
				1,
			},
			1.969,
		},
		{
			fields{
				"32GT/s",
				1,
			},
			3.938,
		},
		{
			fields{
				"64GT/s",
				1,
			},
			7.563,
		},
		{
			fields{
				"128GT/s",
				1,
			},
			15.125,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Speed %s, Width x%d", tt.fields.TransferRatePerLane, tt.fields.LaneWidth), func(t *testing.T) {
			info := IsolatedDevicePCIEInfo{
				TransferRatePerLane: tt.fields.TransferRatePerLane,
				LaneWidth:           tt.fields.LaneWidth,
			}
			if got := info.GetThroughputPerLane(); got.Throughput != tt.want {
				t.Errorf("SIsolatedDevicePCIInfo.GetThroughputPerLane() = %v, want %v", got, tt.want)
			}
		})
	}
}
