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

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func TestCountDevicesWithMinMemoryFromList(t *testing.T) {
	mk := func(path string, memMb int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{DevicePath: path, MemorySize: memMb, VirtualNum: 1}
	}
	mkVirtual := func(path string, memMb, virtualNum, allocated int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{
			DevicePath:          path,
			MemorySize:          memMb,
			VirtualNum:          virtualNum,
			VirtualNumAllocated: allocated,
		}
	}
	mkHAMI := func(path string, memMb, allocatedMb int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{
			DevicePath:          path,
			MemorySize:          memMb,
			MemorySizeAllocated: allocatedMb,
		}
	}

	cases := []struct {
		name        string
		devs        []*core.IsolatedDeviceDesc
		sharingMode string
		minMemMb    int
		want        int
	}{
		{
			name: "plain GPU: 3 cards 24/40/80 GiB, request 30 GiB → 2 fit",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 24576),
				mk("/dev/nvidia1", 40960),
				mk("/dev/nvidia2", 81920),
			},
			sharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, minMemMb: 30000, want: 2,
		},
		{
			name: "plain GPU: request 0 (unconstrained) → all pass through",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 24576),
				mk("/dev/nvidia1", 40960),
			},
			sharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, minMemMb: 0, want: 2,
		},
		{
			name: "unknown MemorySize=0 → passes as unknown (avoid mass exclusion)",
			devs: []*core.IsolatedDeviceDesc{
				mk("/dev/nvidia0", 0),
				mk("/dev/nvidia1", 24576),
			},
			sharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, minMemMb: 40000, want: 1, // unknown stays in, 24GiB excluded
		},
		{
			name: "UNLIMITED share: 2 physical cards, only one card's slots meet req",
			devs: []*core.IsolatedDeviceDesc{
				mkVirtual("/dev/nvidia0", 6144, 4, 0),
				mkVirtual("/dev/nvidia1", 20480, 4, 0),
			},
			sharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED, minMemMb: 10000, want: 1,
		},
		{
			name: "UNLIMITED share: all matching virtual slots are counted",
			devs: []*core.IsolatedDeviceDesc{
				mkVirtual("/dev/nvidia0", 24576, 2, 0),
				mkVirtual("/dev/nvidia1", 24576, 2, 1),
			},
			sharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED, minMemMb: 10000, want: 2,
		},
		{
			name: "HAMI share: count cards with enough remaining memory",
			devs: []*core.IsolatedDeviceDesc{
				mkHAMI("/dev/nvidia0", 24576, 8192),
				mkHAMI("/dev/nvidia1", 24576, 20480),
			},
			sharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, minMemMb: 10000, want: 1,
		},
		{
			name:        "empty pool → 0",
			devs:        []*core.IsolatedDeviceDesc{},
			sharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			minMemMb:    1000,
			want:        0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := countDevicesWithMinMemoryFromList(c.devs, c.sharingMode, c.minMemMb)
			if got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}
