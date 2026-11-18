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

var cambriconVendorInfo = SGPUVendorInfo{
	Vendor:     "Cambricon",
	VendorId:   "cabc",
	Country:    "CN",
	ManageTool: "cnmon",
}

var cambriconMLUarch03 = SGPUGenInfo{
	SGPUVendorInfo:      cambriconVendorInfo,
	Arch:                "MLUarch03",
	ComputeCap:          "MLU370",
	MatrixPeakPrecision: "int8",
}

var cambriconGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: cambriconMLUarch03,
		DeviceId:    "0370",
		Name:        "MLU370-S8", Chip: "思元370",
		ChipsOnDevice: 1,
		MemorySizeGB:  48, MemoryType: "LPDDR5",
		BandwidthGBs: 307.2,
		Fp32Tflops:   18, Fp16Tflops: 72, Bf16Tflops: 72, Int8Tops: 192,
		PeakMatrixTops: 192,
		TdpW:           75, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: cambriconMLUarch03,
		DeviceId:    "0370",
		Name:        "MLU370-X4", Chip: "思元370",
		ChipsOnDevice: 1,
		MemorySizeGB:  24, MemoryType: "LPDDR5",
		BandwidthGBs: 307.2,
		Fp32Tflops:   24, Fp16Tflops: 96, Bf16Tflops: 96, Int8Tops: 256,
		PeakMatrixTops: 256,
		TdpW:           150, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: cambriconMLUarch03,
		DeviceId:    "0370",
		Name:        "MLU370-X8", Chip: "思元370",
		ChipsOnDevice: 2,
		MemorySizeGB:  48, MemoryType: "LPDDR5",
		BandwidthGBs: 614.4,
		Fp32Tflops:   24, Fp16Tflops: 96, Bf16Tflops: 96, Int8Tops: 256,
		PeakMatrixTops: 256,
		TdpW:           250, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
		Interconnect: "MLU-Link", InterconnectBandwidthGBs: 200,
	},
}
