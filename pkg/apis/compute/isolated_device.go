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

import "yunion.io/x/onecloud/pkg/apis"

type IsolateDeviceDetails struct {
	apis.StandaloneResourceDetails
	HostResourceInfo

	SIsolatedDevice

	// 云主机名称
	Guest string `json:"guest"`
	// 云主机状态
	GuestStatus string `json:"guest_status"`
}

type IsolatedDeviceListInput struct {
	apis.StandaloneResourceListInput
	apis.DomainizedResourceListInput

	HostFilterListInput

	// 只列出GPU直通设备
	Gpu *bool `json:"gpu"`
	// 只列出USB直通设备
	Usb *bool `json:"usb"`
	// 只列出未使用的直通设备
	Unused *bool `json:"unused"`

	// # PCI / GPU-HPC / GPU-VGA / USB / NIC
	// 设备类型
	DevType []string `json:"dev_type"`

	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model []string `json:"model"`

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr []string `json:"addr"`

	// 设备VENDOE编号
	VendorDeviceId []string `json:"vendor_device_id"`

	// 展示物理机的上的设备
	ShowBaremetalIsolatedDevices bool `json:"show_baremetal_isolated_devices"`

	// 列出虚拟机上挂载的设备
	GuestId string `json:"guest_id"`
}

type IsolatedDeviceCreateInput struct {
	apis.StandaloneResourceCreateInput

	HostResourceInput
	IsolatedDeviceReservedResourceInput

	// 设备类型USB/GPU
	// example: GPU
	DevType string `json:"dev_type"`

	// 设备型号
	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `json:"model"`

	// PCI地址
	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr string `json:"addr"`

	// 设备VendorId
	VendorDeviceId string `json:"vendor_device_id"`
}

type IsolatedDeviceReservedResourceInput struct {
	// GPU 预留内存
	ReservedMemory *int `json:"reserved_memory"`

	// GPU 预留CPU
	ReservedCpu *int `json:"reserved_cpu"`

	// GPU 预留磁盘
	ReservedStorage *int `json:"reserved_storage"`
}

type IsolatedDeviceUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput
	IsolatedDeviceReservedResourceInput
	DevType string `json:"dev_type"`
}

type IsolatedDeviceJsonDesc struct {
	Id                  string `json:"id"`
	DevType             string `json:"dev_type"`
	Model               string `json:"model"`
	Addr                string `json:"addr"`
	VendorDeviceId      string `json:"vendor_device_id"`
	Vendor              string `json:"vendor"`
	NetworkIndex        int8   `json:"network_index"`
	OvsOffloadInterface string `json:"ovs_offload_interface"`
	DiskIndex           int8   `json:"disk_index"`
	NvmeSizeMB          int    `json:"nvme_size_mb"`
}

type IsolatedDeviceModelCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	// 设备类型
	// example: NPU
	DevType string `json:"dev_type"`

	// 设备型号
	Model string `json:"model"`

	// 设备VendorId
	VendorId string `json:"vendor_id"`

	// 设备DeviceId
	DeviceId string `json:"device_id"`

	// 支持热插拔 HotPluggable
	HotPluggable bool `json:"hot_pluggable"`

	// hosts scan isolated device after isolated_device_model created
	Hosts []string `json:"hosts"`
}

type IsolatedDeviceModelUpdateInput struct {
	apis.StandaloneAnonResourceBaseUpdateInput
	// 设备类型
	// example: NPU
	DevType string `json:"dev_type"`

	// 设备型号
	Model string `json:"model"`

	// 设备VendorId
	VendorId string `json:"vendor_id"`

	// 设备DeviceId
	DeviceId string `json:"device_id"`

	// 支持热插拔 HotPluggable
	HotPluggable bool `json:"hot_pluggable"`
}

type IsolatedDeviceModelListInput struct {
	apis.StandaloneAnonResourceListInput

	// 设备类型
	// example: NPU
	DevType []string `json:"dev_type"`

	// 设备型号
	Model []string `json:"model"`

	// 设备VendorId
	VendorId string `json:"vendor_id"`

	// 设备DeviceId
	DeviceId string `json:"device_id"`

	// 支持热插拔 HotPluggable
	HotPluggable bool `json:"hot_pluggable"`
}
