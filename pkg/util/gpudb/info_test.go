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

package gpudb

import "testing"

func TestGPUInfosIncludesEveryVendor(t *testing.T) {
	sum := 0
	for _, infos := range gpuInfoList {
		if len(infos) == 0 {
			t.Fatal("gpuInfoList contains an empty vendor slice")
		}
		sum += len(infos)
	}
	if len(gpuInfoList) != 14 {
		t.Fatalf("vendor slices = %d, want 14", len(gpuInfoList))
	}
	if len(GPUInfos) != sum {
		t.Fatalf("GPUInfos = %d, vendor slices sum = %d", len(GPUInfos), sum)
	}
}

func TestGetGPUInfoRequiresVendorID(t *testing.T) {
	cases := []struct {
		vendorId string
		device   string
		name     string
	}{
		{"10de", "20b2", "A100-SXM4-80GB"},
		{"10DE", "20B2", "A100-SXM4-80GB"},
		{"19e5", "d500", "Atlas 300I Duo"},
		{"1d94", "6210", "K100_AI"},
		{"1d94", "6320", "BW1000"},
		{"cabc", "0370", "MLU370-S8"},
		{"1e36", "c035", "S60"},
		{"9999", "4001", "曦云C500"},
		{"1ed5", "0323", "MTT S4000"},
		{"1ee0", "0004", "Biren 0004"},
	}
	for _, tc := range cases {
		gpu := GetGPUInfo(tc.vendorId, tc.device, 0, 0)
		if gpu == nil {
			t.Errorf("GetGPUInfo(%q, %q) = nil", tc.vendorId, tc.device)
			continue
		}
		if gpu.Name != tc.name {
			t.Errorf("GetGPUInfo(%q, %q) = %q, want %q", tc.vendorId, tc.device, gpu.Name, tc.name)
		}
	}
	for _, vendor := range []string{"nvidia", "NVIDIA", "ascend", "huawei"} {
		if gpu := GetGPUInfo(vendor, "20b2", 0, 0); gpu != nil {
			t.Errorf("GetGPUInfo(%q) = %s, want nil", vendor, gpu.Name)
		}
	}
	gpu, err := GetGPUInfo2("10de:20b2", 0, 0)
	if err != nil || gpu == nil || gpu.Name != "A100-SXM4-80GB" {
		t.Fatalf("GetGPUInfo2(10de:20b2) = %v, %v", gpu, err)
	}
	gpu, err = GetGPUInfo2("nvidia:20b2", 0, 0)
	if err != nil || gpu != nil {
		t.Fatalf("GetGPUInfo2(nvidia:20b2) = %v, %v", gpu, err)
	}
	gpu, err = GetGPUInfo2("19e5:d500", 0, 0)
	if err != nil || gpu == nil || gpu.Name != "Atlas 300I Duo" {
		t.Fatalf("GetGPUInfo2(19e5:d500) = %v, %v", gpu, err)
	}
	gpu, err = GetGPUInfo2("10de:2d04", 16, 0)
	if err != nil || gpu == nil || gpu.Name != "GeForce RTX 5060 Ti 16GB" {
		t.Fatalf("GetGPUInfo2(10de:2d04) = %v, %v", gpu, err)
	}
	if _, err := GetGPUInfo2("10de", 0, 0); err == nil {
		t.Fatal("GetGPUInfo2(10de) error = nil, want invalid format")
	}
	gpu = GetGPUInfo("10de", "20b2", 1, 9)
	if gpu == nil || gpu.Name != "A100-SXM4-80GB" {
		t.Fatalf("GetGPUInfo(10de, 20b2, 1, 9) = %v", gpu)
	}
}

func TestGetGPUInfoReturnsCopy(t *testing.T) {
	gpu := GetGPUInfo("10de", "20b2", 0, 0)
	gpu.Name = "changed"
	again := GetGPUInfo("10de", "20b2", 0, 0)
	if again.Name != "A100-SXM4-80GB" {
		t.Fatalf("catalog name changed to %q", again.Name)
	}
}

func TestGetGPUInfoSelectsMemoryAndChips(t *testing.T) {
	saved := gpuInfosMap
	gpuInfosMap = make(map[string]map[string][]*SGPUInfo)
	t.Cleanup(func() { gpuInfosMap = saved })

	entry := func(name string, memory uint, chips uint) *SGPUInfo {
		return &SGPUInfo{
			SGPUGenInfo: SGPUGenInfo{
				SGPUVendorInfo: SGPUVendorInfo{Vendor: "TestVendor", VendorId: "ffff"},
			},
			DeviceId:      "aaaa",
			Name:          name,
			MemorySizeGB:  memory,
			ChipsOnDevice: chips,
		}
	}
	indexGPUInfo(entry("small", 48, 1))
	indexGPUInfo(entry("mid-more", 64, 4))
	indexGPUInfo(entry("large", 80, 2))
	indexGPUInfo(entry("mid", 64, 2))
	indexGPUInfo(entry("large-single", 80, 1))
	indexGPUInfo(&SGPUInfo{
		SGPUGenInfo: SGPUGenInfo{
			SGPUVendorInfo: SGPUVendorInfo{VendorId: "ffff"},
		},
		DeviceId:      "bbbb",
		Name:          "only",
		MemorySizeGB:  24,
		ChipsOnDevice: 1,
	})
	sortGPUInfosMap()

	list := gpuInfosMap["ffff"]["aaaa"]
	wantOrder := []string{"large-single", "large", "mid", "mid-more", "small"}
	if len(list) != len(wantOrder) {
		t.Fatalf("sorted list length = %d, want %d", len(list), len(wantOrder))
	}
	for i, name := range wantOrder {
		if list[i].Name != name {
			t.Fatalf("sorted[%d] = %q, want %q", i, list[i].Name, name)
		}
	}

	cases := []struct {
		mem   uint
		chips uint
		want  string
	}{
		{0, 0, "large-single"},
		{0, 2, "large-single"},
		{96, 1, "large-single"},
		{80, 0, "large-single"},
		{80, 2, "large-single"},
		{79, 0, "mid"},
		{64, 0, "mid"},
		{64, 4, "mid"},
		{63, 1, "small"},
		{48, 1, "small"},
		{32, 1, "large-single"},
	}
	for _, tc := range cases {
		gpu := GetGPUInfo("ffff", "aaaa", tc.mem, tc.chips)
		got := ""
		if gpu != nil {
			got = gpu.Name
		}
		if got != tc.want {
			t.Errorf("GetGPUInfo(mem=%d, chips=%d) = %q, want %q", tc.mem, tc.chips, got, tc.want)
		}
	}
	if gpu := GetGPUInfo("ffff", "bbbb", 80, 2); gpu == nil || gpu.Name != "only" {
		t.Fatalf("single record = %v, want only", gpu)
	}
	if gpu := GetGPUInfo("testvendor", "aaaa", 0, 0); gpu != nil {
		t.Fatalf("vendor name lookup = %s, want nil", gpu.Name)
	}
}

func TestDuplicateIdentityPanics(t *testing.T) {
	saved := gpuInfosMap
	gpuInfosMap = make(map[string]map[string][]*SGPUInfo)
	t.Cleanup(func() { gpuInfosMap = saved })

	gpu := &SGPUInfo{
		SGPUGenInfo: SGPUGenInfo{
			SGPUVendorInfo: SGPUVendorInfo{VendorId: "ffff"},
		},
		DeviceId:      "aaaa",
		Name:          "first",
		MemorySizeGB:  80,
		ChipsOnDevice: 1,
	}
	indexGPUInfo(gpu)
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate VendorId+DeviceId+ChipsOnDevice+MemorySizeGB did not panic")
		}
	}()
	indexGPUInfo(gpu)
}
