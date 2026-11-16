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

var vastaitechVendorInfo = SGPUVendorInfo{
	Vendor:     "VASTAITECH",
	VendorId:   "1ec6",
	Country:    "CN",
	ManageTool: "vasmi",
}

var vastaitechSV100 = SGPUGenInfo{
	SGPUVendorInfo: vastaitechVendorInfo,
	Arch:           "VUCA",
}

var vastaitechGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: vastaitechSV100,
		DeviceId:    "0100",
		Name:        "SV100", Chip: "SV100",
		ChipsOnDevice: 1,
		MemorySizeGB:  32,
		TdpW:          70, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
}
