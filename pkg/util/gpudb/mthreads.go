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

var mthreadsVendorInfo = SGPUVendorInfo{
	Vendor:     "MTHREADS",
	VendorId:   "1ed5",
	Country:    "CN",
	ManageTool: "mthreads-gmi",
}

var mthreadsQuyuan = SGPUGenInfo{
	SGPUVendorInfo: mthreadsVendorInfo,
	Arch:           "曲院",
	ComputeCap:     "2.2",
}

var mthreadsPingHu = SGPUGenInfo{
	SGPUVendorInfo: mthreadsVendorInfo,
	Arch:           "平湖",
	ComputeCap:     "3.1",
}

var mthreadsGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo:   mthreadsQuyuan,
		DeviceId:      "0323",
		Name:          "MTT S4000",
		ChipsOnDevice: 1,
		MemorySizeGB:  48,
		BandwidthGBs:  768,
		TdpW:          450, FormFactor: "PCIe", PcieGen: "5.0", PcieLanes: 16,
		Interconnect: "MTLink", InterconnectBandwidthGBs: 240,
		MigSupported: true,
	},
	{
		SGPUGenInfo: mthreadsPingHu,
		DeviceId:    "0400",
		Name:        "MTT S5000", Chip: "PH100",
		ChipsOnDevice: 1,
		MemorySizeGB:  80,
		Ecc:           true,
		FormFactor:    "OAM", PcieLanes: 16,
		Interconnect: "MTLink",
		MigSupported: true,
	},
}
