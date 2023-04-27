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

package compute

const (
	DIRECT_PCI_TYPE = "PCI"
	GPU_HPC_TYPE    = "GPU-HPC" // # for compute
	GPU_VGA_TYPE    = "GPU-VGA" // # for display
	USB_TYPE        = "USB"
	NIC_TYPE        = "NIC"     // nic sriov
	NVME_PT_TYPE    = "NVME-PT" // nvme passthrough
	NVME_TYPE       = "NVME"    // nvme sriov

	NVIDIA_VENDOR_ID = "10de"
	AMD_VENDOR_ID    = "1002"
)

var VALID_GPU_TYPES = []string{GPU_HPC_TYPE, GPU_VGA_TYPE}

var VALID_PASSTHROUGH_TYPES = []string{DIRECT_PCI_TYPE, USB_TYPE, NIC_TYPE, GPU_HPC_TYPE, GPU_VGA_TYPE, NVME_PT_TYPE, NVME_TYPE}

var ID_VENDOR_MAP = map[string]string{
	NVIDIA_VENDOR_ID: "NVIDIA",
	AMD_VENDOR_ID:    "AMD",
}

var VENDOR_ID_MAP = map[string]string{
	"NVIDIA": NVIDIA_VENDOR_ID,
	"AMD":    AMD_VENDOR_ID,
}
