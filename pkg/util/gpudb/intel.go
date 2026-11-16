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

var intelVendorInfo = SGPUVendorInfo{
	Vendor:              "Intel",
	VendorId:            "8086",
	Country:             "US",
	ManageTool:          "xpu-smi",
	ComputeBlockName:    "Xe Core",
	VectorPeakPrecision: "fp32",
}

var intelXeHPG = SGPUGenInfo{
	SGPUVendorInfo:      intelVendorInfo,
	Arch:                "Xe-HPG",
	ComputeCap:          "DG2",
	TensorSparsity:      1,
	MatrixPeakPrecision: "int8",
}

var intelGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: intelXeHPG,
		DeviceId:    "56c0",
		Name:        "Data Center GPU Flex 170", Chip: "ACM-G10",
		ChipsOnDevice: 1,
		ComputeBlock:  32, VectorLanes: 512, MatrixEngines: 512,
		MemorySizeGB: 16, MemoryType: "GDDR6", MemoryBusBit: 256,
		BandwidthGBs: 576,
		Fp32Tflops:   16.79, Fp16Tflops: 127.8, Int8Tops: 250,
		PeakVectorTflops: 16.79, PeakMatrixTops: 250,
		TdpW: 150, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
		MigSupported: true, MaxVfs: 31,
		ReleaseDate: "2022-08",
	},
}
