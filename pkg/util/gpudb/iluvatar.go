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

var iluvatarVendorInfo = SGPUVendorInfo{
	Vendor:     "ILUVATAR",
	VendorId:   "1e3e",
	Country:    "CN",
	ManageTool: "ixsmi",
}

var iluvatarGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: SGPUGenInfo{SGPUVendorInfo: iluvatarVendorInfo},
		DeviceId:    "0002",
		Name:        "MR-V100", Chip: "MR-V100",
		ChipsOnDevice: 1,
		MemorySizeGB:  32, MemoryType: "HBM2e",
		TdpW: 150, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: SGPUGenInfo{SGPUVendorInfo: iluvatarVendorInfo},
		DeviceId:    "000f",
		Name:        "BI-V150S", Chip: "BI-V150S",
		ChipsOnDevice: 1,
		MemorySizeGB:  32, MemoryType: "HBM2e",
		TdpW: 450, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
}
