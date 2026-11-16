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

var metaxVendorInfo = SGPUVendorInfo{
	Vendor:     "MetaX",
	VendorId:   "9999",
	Country:    "CN",
	ManageTool: "mx-smi",
}

var metaxMXC500 = SGPUGenInfo{
	SGPUVendorInfo: metaxVendorInfo,
	Arch:           "GPGPU",
	ComputeCap:     "MXC500",
}

var metaxGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo:   metaxMXC500,
		DeviceId:      "4001",
		Name:          "曦云C500",
		ChipsOnDevice: 1,
		Interconnect:  "MetaXLink",
		MigSupported:  true,
	},
}
