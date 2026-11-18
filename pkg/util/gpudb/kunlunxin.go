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

var kunlunxinVendorInfo = SGPUVendorInfo{
	Vendor:     "KUNLUNXIN",
	VendorId:   "1d22",
	Country:    "CN",
	ManageTool: "xpu-smi",
}

var kunlunxinKL3 = SGPUGenInfo{
	SGPUVendorInfo: kunlunxinVendorInfo,
	Arch:           "XPU-P", ComputeCap: "KL3",
}

var kunlunxinGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: kunlunxinKL3,
		DeviceId:    "3688",
		Name:        "P800 OAM", Chip: "P800",
		ChipsOnDevice: 1,
		MemorySizeGB:  96,
		TdpW:          400, FormFactor: "OAM", PcieGen: "4.0", PcieLanes: 16,
	},
}
