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

package predicates

import (
	"testing"

	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func TestCountDevicesWithMinMemoryFromList(t *testing.T) {
	mk := func(path string, memMb int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{DevicePath: path, MemorySize: memMb}
	}

	cases := []struct {
		name     string
		devs     []*core.IsolatedDeviceDesc
		shared   bool
		minMemMb int
		want     int
	}{
		{
			name: "plain GPU: 3 cards 24/40/80 GiB, request 30 GiB → 2 fit",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 24576),
				mk("/dev/nvidia1", 40960),
				mk("/dev/nvidia2", 81920),
			},
			shared: false, minMemMb: 30000, want: 2,
		},
		{
			name: "plain GPU: request 0 (unconstrained) → all pass through",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 24576),
				mk("/dev/nvidia1", 40960),
			},
			shared: false, minMemMb: 0, want: 2,
		},
		{
			name: "unknown MemorySize=0 → passes as unknown (avoid mass exclusion)",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 0),
				mk("/dev/nvidia1", 24576),
			},
			shared: false, minMemMb: 40000, want: 1, // unknown stays in, 24GiB excluded
		},
		{
			name: "MPS share: 2 physical cards, 4 slices each, only 1 card meets req",
			devs: []*core.IsolatedDeviceDesc{
				// card 0: 6 GiB per slice (4 slices × same path)
				mk("/dev/nvidia0", 6144), mk("/dev/nvidia0", 6144),
				mk("/dev/nvidia0", 6144), mk("/dev/nvidia0", 6144),
				// card 1: 20 GiB per slice
				mk("/dev/nvidia1", 20480), mk("/dev/nvidia1", 20480),
				mk("/dev/nvidia1", 20480), mk("/dev/nvidia1", 20480),
			},
			shared: true, minMemMb: 10000, want: 1, // only card 1 satisfies
		},
		{
			name: "MPS share: all slices pass through dedup → count by DevicePath",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 24576), mk("/dev/nvidia0", 24576),
				mk("/dev/nvidia1", 24576),
			},
			shared: true, minMemMb: 10000, want: 2, // 2 distinct paths
		},
		{
			name:     "empty pool → 0",
			devs:     []*core.IsolatedDeviceDesc{},
			shared:   false,
			minMemMb: 1000,
			want:     0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := countDevicesWithMinMemoryFromList(c.devs, c.shared, c.minMemMb)
			if got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}
