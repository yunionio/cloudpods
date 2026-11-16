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

var hygonVendorInfo = SGPUVendorInfo{
	Vendor:           "HYGON",
	VendorId:         "1d94",
	Country:          "CN",
	ManageTool:       "hy-smi",
	ComputeBlockName: "CU",
}

var hygonDCU = SGPUGenInfo{
	SGPUVendorInfo: hygonVendorInfo,
	Arch:           "DCU",
	ComputeCap:     "gfx928",
}

var hygonGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: hygonDCU,
		DeviceId:    "6210",
		Name:        "K100_AI", Chip: "KONGMING",
		ChipsOnDevice: 1,
		ComputeBlock:  120,
		MemorySizeGB:  64,
		TdpW:          400, FormFactor: "PCIe",
	},
}
