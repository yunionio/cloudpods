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

var huaweiVendorInfo = SGPUVendorInfo{
	Vendor:                 "Ascend",
	VendorId:               "19e5",
	Country:                "CN",
	ManageTool:             "npu-smi",
	ComputeBlockName:       "AI Core",
	SchedulingExplicitTile: true,
	VectorPeakPrecision:    "fp16",
}

var (
	huaweiDaVinci310P = SGPUGenInfo{
		SGPUVendorInfo: huaweiVendorInfo,
		Arch:           "Da Vinci", ComputeCap: "Ascend310P",
		TensorSparsity: 1, MatrixPeakPrecision: "int8",
	}
	huaweiDaVinci910 = SGPUGenInfo{
		SGPUVendorInfo: huaweiVendorInfo,
		Arch:           "Da Vinci", ComputeCap: "Ascend910",
		TensorSparsity: 1, MatrixPeakPrecision: "fp16",
	}
	huaweiDaVinci910B = SGPUGenInfo{
		SGPUVendorInfo: huaweiVendorInfo,
		Arch:           "Da Vinci", ComputeCap: "Ascend910B",
		TensorSparsity: 1, MatrixPeakPrecision: "fp16",
	}
)

var huaweiGPUInfos = []SGPUInfo{
	{
		SGPUGenInfo: huaweiDaVinci310P,
		DeviceId:    "d500",
		Name:        "Atlas 300I Pro", Chip: "Ascend 310P",
		ChipsOnDevice: 1,
		ComputeBlock:  8,
		VectorLanes:   8, MatrixEngines: 8, ScalarUnit: 8,
		PeakMatrixTops: 140,
		MemorySizeGB:   24, MemoryType: "LPDDR4X", Ecc: true,
		BandwidthGBs: 204.8,
		Fp16Tflops:   70, Int8Tops: 140,
		TdpW: 72, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: huaweiDaVinci310P,
		DeviceId:    "d500",
		Name:        "Atlas 300I Duo", Chip: "Ascend 310P",
		ChipsOnDevice: 2,
		ComputeBlock:  16,
		VectorLanes:   16, MatrixEngines: 16, ScalarUnit: 16,
		PeakMatrixTops: 280,
		MemorySizeGB:   96, MemoryType: "LPDDR4X", Ecc: true,
		BandwidthGBs: 408,
		Fp16Tflops:   140, Int8Tops: 280,
		TdpW: 150, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: huaweiDaVinci310P,
		DeviceId:    "d500",
		Name:        "Atlas 300V Pro", Chip: "Ascend 310P",
		ChipsOnDevice: 1,
		ComputeBlock:  8,
		VectorLanes:   8, MatrixEngines: 8, ScalarUnit: 8,
		PeakMatrixTops: 140,
		MemorySizeGB:   48, MemoryType: "LPDDR4X",
		BandwidthGBs: 204.8,
		Fp16Tflops:   70, Int8Tops: 140,
		TdpW: 72, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: huaweiDaVinci910B,
		DeviceId:    "d802",
		Name:        "Atlas 300I A2 32GB", Chip: "Ascend 910B",
		ChipsOnDevice: 1,
		ComputeBlock:  20,
		VectorLanes:   20, MatrixEngines: 20, ScalarUnit: 20,
		PeakMatrixTops: 280,
		MemorySizeGB:   32, MemoryType: "HBM", Ecc: true,
		BandwidthGBs: 800,
		Fp32Tflops:   75, Fp16Tflops: 280, Int8Tops: 560,
		TdpW: 350, FormFactor: "PCIe", PcieGen: "5.0", PcieLanes: 16,
	},
	{
		SGPUGenInfo: huaweiDaVinci910B,
		DeviceId:    "d802",
		Name:        "Atlas 300I A2 64GB", Chip: "Ascend 910B",
		ChipsOnDevice: 1,
		ComputeBlock:  20,
		VectorLanes:   20, MatrixEngines: 20, ScalarUnit: 20,
		PeakMatrixTops: 280,
		MemorySizeGB:   64, MemoryType: "HBM", Ecc: true,
		BandwidthGBs: 1600,
		Fp32Tflops:   75, Fp16Tflops: 280, Int8Tops: 560,
		TdpW: 350, FormFactor: "PCIe", PcieGen: "5.0", PcieLanes: 16,
		MigSupported: true, MaxVfs: 12,
	},
	{
		SGPUGenInfo: huaweiDaVinci910,
		DeviceId:    "d801",
		Name:        "Atlas 300T Pro", Chip: "Ascend 910",
		ChipsOnDevice: 1,
		ComputeBlock:  30,
		VectorLanes:   30, MatrixEngines: 30, ScalarUnit: 30,
		PeakMatrixTops: 280,
		MemorySizeGB:   32, MemoryType: "HBM", Ecc: true,
		Fp16Tflops: 280, Int8Tops: 560,
		TdpW: 300, FormFactor: "PCIe", PcieGen: "4.0", PcieLanes: 16,
		Interconnect: "RoCE", InterconnectBandwidthGBs: 12.5,
		MigSupported: true,
	},
}
