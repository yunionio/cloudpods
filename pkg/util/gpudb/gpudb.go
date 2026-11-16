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

type SGPUVendorInfo struct {
	Vendor                 string `json:"vendor"`
	VendorId               string `json:"vendor_id"`
	Country                string `json:"country"`
	ManageTool             string `json:"manage_tool"`
	ComputeBlockName       string `json:"compute_block_name"`
	SchedulingWidth        int    `json:"scheduling_width"`
	SchedulingExplicitTile bool   `json:"scheduling_explicit_tile"`
	VectorPeakPrecision    string `json:"vector_peak_precision"`
}

type SGPUGenInfo struct {
	SGPUVendorInfo

	Arch                string  `json:"arch"`
	ComputeCap          string  `json:"compute_cap"`
	TensorSparsity      float64 `json:"tensor_sparsity"`
	MatrixPeakPrecision string  `json:"matrix_peak_precision"`
}

type SGPUInfo struct {
	SGPUGenInfo

	DeviceId string `json:"device_id"`

	Name          string `json:"name"`
	Chip          string `json:"chip"`
	ChipsOnDevice uint   `json:"chips_on_device"`

	ComputeBlock  int `json:"compute_block"`
	VectorLanes   int `json:"vector_lanes"`
	MatrixEngines int `json:"matrix_engines"`
	ScalarUnit    int `json:"scalar_unit"`

	MemorySizeGB uint   `json:"memory_size_gb"`
	MemoryType   string `json:"memory_type"`
	MemoryBusBit int    `json:"memory_bus_bit"`
	Ecc          bool   `json:"ecc"`

	BandwidthGBs float64 `json:"bandwidth_gb_s"`

	Fp32Tflops float64 `json:"fp32_tflops"`
	Fp64Tflops float64 `json:"fp64_tflops"`

	Tf32Tflops       float64 `json:"tf32_tflops"`
	Fp16Tflops       float64 `json:"fp16_tflops"`
	Bf16Tflops       float64 `json:"bf16_tflops"`
	Fp8Tflops        float64 `json:"fp8_tflops"`
	Fp6Tflops        float64 `json:"fp6_tflops"`
	Fp4Tflops        float64 `json:"fp4_tflops"`
	Int8Tops         float64 `json:"int8_tops"`
	PeakVectorTflops float64 `json:"peak_vector_tflops"`
	PeakMatrixTops   float64 `json:"peak_matrix_tops"`

	TdpW int `json:"tdp_w"`

	FormFactor string `json:"form_factor"`
	PcieGen    string `json:"pcie_gen"`
	PcieLanes  int    `json:"pcie_lanes"`

	Interconnect             string  `json:"interconnect"`
	InterconnectBandwidthGBs float64 `json:"interconnect_bandwidth_gb_s"`

	MigSupported bool `json:"mig_supported"`
	MaxVfs       int  `json:"max_vfs"`

	ReleaseDate string `json:"release_date"`
}
