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

var enflameVendorInfo = SGPUVendorInfo{
	Vendor:     "Enflame",
	VendorId:   "1e36",
	Country:    "CN",
	ManageTool: "efsmi",
}

var enflameGCU320 = SGPUGenInfo{
	SGPUVendorInfo: enflameVendorInfo,
	Arch:           "GCU",
	ComputeCap:     "邃思320",
}

var enflameGCU400 = SGPUGenInfo{
	SGPUVendorInfo: enflameVendorInfo,
	Arch:           "GCU",
	ComputeCap:     "邃思400",
}

var enflameGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: enflameGCU320,
		DeviceId:    "c035",
		Name:        "S60", Chip: "邃思320",
		ChipsOnDevice: 1,
		MemorySizeGB:  48, MemoryType: "GDDR6", MemoryBusBit: 384, Ecc: true,
		BandwidthGBs: 672,
		TdpW:         300, FormFactor: "PCIe", PcieGen: "5.0", PcieLanes: 16,
		MigSupported: true, MaxVfs: 4,
	},
	{
		SGPUGenInfo: enflameGCU400,
		DeviceId:    "c042",
		Name:        "L600", Chip: "邃思400",
		ChipsOnDevice: 1,
	},
}
