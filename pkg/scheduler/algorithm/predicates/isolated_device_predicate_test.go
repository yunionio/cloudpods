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

func TestFilterDescsByMinMemoryModelIntersection(t *testing.T) {
	mk := func(model string, memMb int) *core.IsolatedDeviceDesc {
		return &core.IsolatedDeviceDesc{
			Model:      model,
			DevType:    computeapi.GPU_TYPE,
			DevicePath: "/dev/" + model,
			MemorySize: memMb,
			VirtualNum: 1,
		}
	}
	pool := []*core.IsolatedDeviceDesc{
		mk("T4", 16384),
		mk("A100", 40960),
	}

	// Request model=T4 memory_mb=40000: type-level VRAM would see the A100
	// and pass; intersecting with model must fail.
	t4s := make([]*core.IsolatedDeviceDesc, 0)
	for _, d := range pool {
		if d.Model == "T4" {
			t4s = append(t4s, d)
		}
	}
	fit := filterDescsByMinMemory(t4s, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, 40000)
	if len(fit) != 0 {
		t.Fatalf("T4 16GiB must not satisfy 40GiB request, got %d", len(fit))
	}

	a100s := make([]*core.IsolatedDeviceDesc, 0)
	for _, d := range pool {
		if d.Model == "A100" {
			a100s = append(a100s, d)
		}
	}
	fit = filterDescsByMinMemory(a100s, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, 40000)
	if len(fit) != 1 {
		t.Fatalf("A100 40GiB should satisfy 40GiB request, got %d", len(fit))
	}
}

func TestFilterDescsByMinMemoryTwoCards24GiBRequest30(t *testing.T) {
	devs := []*core.IsolatedDeviceDesc{
		{DevicePath: "/dev/nvidia0", MemorySize: 24576, VirtualNum: 1},
		{DevicePath: "/dev/nvidia1", MemorySize: 24576, VirtualNum: 1},
	}
	got := countDevicesWithMinMemoryFromList(devs, computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, 30000)
	if got != 0 {
		t.Fatalf("two 24GiB cards must not satisfy 30GiB, got %d", got)
	}
}

func TestMaxMinMemoryForRequests(t *testing.T) {
	reqs := []*computeapi.IsolatedDeviceConfig{
		{DevType: computeapi.GPU_TYPE, SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, Model: "T4", MemoryMb: 16384},
		{DevType: computeapi.GPU_TYPE, SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, Model: "T4", MemoryMb: 40000},
		{DevType: computeapi.GPU_TYPE, SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, Model: "A100", MemoryMb: 8192},
	}
	got := maxMinMemoryForRequests(reqs, func(d *computeapi.IsolatedDeviceConfig) bool {
		return d.Model == "T4"
	})
	if got != 40000 {
		t.Fatalf("max min memory for T4 = %d want 40000", got)
	}
}

func TestIsolatedDeviceShortageMessage(t *testing.T) {
	hamiMatchGPU := func(d *computeapi.IsolatedDeviceConfig) bool {
		return d.DevType == computeapi.GPU_TYPE && d.SharingMode == computeapi.DEVICE_SHARING_MODE_HAMI
	}
	exclMatchGPU := func(d *computeapi.IsolatedDeviceConfig) bool {
		return d.DevType == computeapi.GPU_TYPE && d.SharingMode == computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
	}
	mpsMatchGPU := func(d *computeapi.IsolatedDeviceConfig) bool {
		return d.DevType == computeapi.GPU_TYPE && d.SharingMode == computeapi.DEVICE_SHARING_MODE_MPS
	}

	cases := []struct {
		name     string
		reqs     []*computeapi.IsolatedDeviceConfig
		match    func(*computeapi.IsolatedDeviceConfig) bool
		devType  string
		sharing  string
		minMem   int
		path     string
		reqCount int
		hostFree int
		pending  int
		asMem    *bool
		want     string
	}{
		{
			name: "HAMI with vendor model and MiB units",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, Vendor: "NVIDIA", Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryMb: 8192, MemoryRequest: 5815},
			},
			match:    hamiMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_HAMI,
			minMem:   8192,
			reqCount: 5815,
			hostFree: 3328,
			want:     `IsolatedDevice type "GPU" vendor "NVIDIA" model "A100" sharing_mode "HAMI" memory>=8192MiB not enough, request: 5815 MiB, hostFree: 3328 MiB`,
		},
		{
			name: "EXCLUSIVE with vendor model, count not MiB",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, Vendor: "NVIDIA", Model: "T4", SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, MemoryMb: 16384},
			},
			match:    exclMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			minMem:   16384,
			reqCount: 2,
			hostFree: 1,
			want:     `IsolatedDevice type "GPU" vendor "NVIDIA" model "T4" sharing_mode "EXCLUSIVE" memory>=16384MiB not enough, request: 2, hostFree: 1`,
		},
		{
			name: "missing vendor and model omitted",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryRequest: 4096},
			},
			match:    hamiMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_HAMI,
			reqCount: 4096,
			hostFree: 1024,
			want:     `IsolatedDevice type "GPU" sharing_mode "HAMI" not enough, request: 4096 MiB, hostFree: 1024 MiB`,
		},
		{
			name: "MPS sharing_mode and pending",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, Vendor: "NVIDIA", Model: "A10", SharingMode: computeapi.DEVICE_SHARING_MODE_MPS},
			},
			match:    mpsMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_MPS,
			reqCount: 4,
			hostFree: 2,
			pending:  1,
			want:     `IsolatedDevice type "GPU" vendor "NVIDIA" model "A10" sharing_mode "MPS" not enough, request: 4, hostFree: 2, pending: 1`,
		},
		{
			name: "mixed vendors joined",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, Vendor: "NVIDIA", Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE},
				{DevType: computeapi.GPU_TYPE, Vendor: "AMD", Model: "MI250", SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE},
			},
			match:    exclMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			reqCount: 2,
			hostFree: 0,
			want:     `IsolatedDevice type "GPU" vendor "NVIDIA,AMD" model "A100,MI250" sharing_mode "EXCLUSIVE" not enough, request: 2, hostFree: 0`,
		},
		{
			name: "VRAM fit keeps count units even for HAMI",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, Vendor: "NVIDIA", Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryMb: 8192},
			},
			match:    hamiMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_HAMI,
			minMem:   8192,
			reqCount: 2,
			hostFree: 1,
			asMem:    boolPtr(false),
			want:     `IsolatedDevice type "GPU" vendor "NVIDIA" model "A100" sharing_mode "HAMI" memory>=8192MiB not enough, request: 2, hostFree: 1`,
		},
		{
			name: "device_path included",
			reqs: []*computeapi.IsolatedDeviceConfig{
				{DevType: computeapi.GPU_TYPE, Vendor: "NVIDIA", Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE, DevicePath: "/dev/nvidia0"},
			},
			match:    exclMatchGPU,
			devType:  computeapi.GPU_TYPE,
			sharing:  computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			path:     "/dev/nvidia0",
			reqCount: 1,
			hostFree: 0,
			want:     `IsolatedDevice type "GPU" vendor "NVIDIA" model "A100" sharing_mode "EXCLUSIVE" device_path "/dev/nvidia0" not enough, request: 1, hostFree: 0`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec := shortageSpecFromRequests(c.reqs, c.match, c.devType, c.sharing, c.minMem)
			spec.devicePath = c.path
			if c.asMem != nil {
				spec.amountIsMemory = *c.asMem
			}
			got := isolatedDeviceShortageMessage(spec, c.reqCount, c.hostFree, c.pending)
			if got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}
