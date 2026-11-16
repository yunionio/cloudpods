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

var theadVendorInfo = SGPUVendorInfo{
	Vendor:           "THEAD",
	VendorId:         "1ded",
	Country:          "CN",
	ManageTool:       "ppu-smi",
	ComputeBlockName: "CU",
}

var theadBeethoven80 = SGPUGenInfo{
	SGPUVendorInfo: theadVendorInfo,
	Arch:           "Beethoven",
	ComputeCap:     "8.0",
}

var theadBeethoven89 = SGPUGenInfo{
	SGPUVendorInfo: theadVendorInfo,
	Arch:           "Beethoven",
	ComputeCap:     "8.9",
}

var theadGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo:   theadBeethoven80,
		DeviceId:      "6001",
		Name:          "PPU-ZW810E",
		ChipsOnDevice: 1,
		MemorySizeGB:  96, MemoryType: "HBM2e", Ecc: true,
		BandwidthGBs: 2765,
		TdpW:         400, FormFactor: "OAM", PcieGen: "5.0", PcieLanes: 16,
		Interconnect: "ICN", InterconnectBandwidthGBs: 700,
		MigSupported: true, MaxVfs: 2,
	},
	{
		SGPUGenInfo:   theadBeethoven89,
		DeviceId:      "6005",
		Name:          "ZW-M890P",
		ChipsOnDevice: 1,
		MemorySizeGB:  96, Ecc: true,
		TdpW: 850, PcieGen: "5.0", PcieLanes: 16,
		Interconnect: "ICN",
		MigSupported: true,
	},
}
