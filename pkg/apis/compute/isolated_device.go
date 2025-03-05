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

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ErrMsgIsolatedDeviceUsedByServer = "Isolated device used by server"
)

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

	// 设备路径
	DevicePath []string `json:"device_path"`

	// 设备VENDOE编号
	VendorDeviceId []string `json:"vendor_device_id"`

	// NUMA节点序号
	NumaNode []uint8 `json:"numa_node"`

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

	// legacy vgpu mdev id
	MdevId string `json:"mdev_id"`

	// 设备VendorId
	VendorDeviceId string `json:"vendor_device_id"`
	// PCIE information
	PCIEInfo *IsolatedDevicePCIEInfo `json:"pcie_info"`
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
	// PCIE information
	PCIEInfo *IsolatedDevicePCIEInfo `json:"pcie_info"`
}

type IsolatedDeviceJsonDesc struct {
	Id                  string `json:"id"`
	DevType             string `json:"dev_type"`
	Model               string `json:"model"`
	Addr                string `json:"addr"`
	VendorDeviceId      string `json:"vendor_device_id"`
	Vendor              string `json:"vendor"`
	NetworkIndex        int    `json:"network_index"`
	IsInfinibandNic     bool   `json:"is_infiniband_nic"`
	OvsOffloadInterface string `json:"ovs_offload_interface"`
	DiskIndex           int8   `json:"disk_index"`
	NvmeSizeMB          int    `json:"nvme_size_mb"`
	MdevId              string `json:"mdev_id"`
	NumaNode            int8   `json:"numa_node"`
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

	// 宿主机 Id
	HostId string `json:"host_id"`
}

type IsolatedDeviceModelHardwareInfo struct {
	// GPU memory size
	MemoryMB int `json:"memory_mb" help:"Memory size MB of the device"`
	// GPU bandwidth. The unit is GB/s
	Bandwidth float64 `json:"bandwidth" help:"Bandwidth of the device, and the unit is GB/s"`
	// TFLOPS stands for number of floating point operations per second.
	TFLOPS float64 `json:"tflops" help:"TFLOPS of the device, which standing for number of floating point operations per second"`
}

type IsolatedDevicePCIEInfo struct {
	// Transder rate per lane
	// Transfer rate refers to the encoded serial bit rate; 2.5 GT/s means 2.5 Gbit/s serial data rate.
	TransferRatePerLane string `json:"transfer_rate_per_lane"`
	// Lane width
	LaneWidth int `json:"lane_width,omitzero"`

	// The following attributes are calculated by TransferRatePerLane and LaneWidth

	// Throughput indicates the unencoded bandwidth (without 8b/10b, 128b/130b, or 242B/256B encoding overhead).
	// The PCIe 1.0 transfer rate of 2.5 GT/s per lane means a 2.5 Gbit/s serial bit rate corresponding to a throughput of 2.0 Gbit/s or 250 MB/s prior to 8b/10b encoding.
	Throughput string `json:"throughput"`
	// Version is the PCIE version
	Version string `json:"version"`
}

func NewIsolatedDevicePCIEInfo(transferRate string, laneWidth int) (*IsolatedDevicePCIEInfo, error) {
	info := &IsolatedDevicePCIEInfo{
		TransferRatePerLane: transferRate,
		LaneWidth:           laneWidth,
	}
	if err := info.fillData(); err != nil {
		return info, errors.Wrap(err, "fillData")
	}
	return info, nil
}

func (info *IsolatedDevicePCIEInfo) String() string {
	return jsonutils.Marshal(info).String()
}

func (info *IsolatedDevicePCIEInfo) IsZero() bool {
	if *info == (IsolatedDevicePCIEInfo{}) {
		return true
	}
	return false
}

const (
	PCIEVersion1       = "1.0"
	PCIEVersion2       = "2.0"
	PCIEVersion3       = "3.0"
	PCIEVersion4       = "4.0"
	PCIEVersion5       = "5.0"
	PCIEVersion6       = "6.0"
	PCIEVersion7       = "7.0"
	PCIEVersionUnknown = "unknown"
)

type PCIEVersionThroughput struct {
	Version    string
	Throughput float64
}

func NewPCIEVersionThroughput(version string) PCIEVersionThroughput {
	// FROM: https://en.wikipedia.org/wiki/PCI_Express
	var (
		v1 = 0.25
		v2 = 0.5
		v3 = 0.985
		v4 = 1.969
		v5 = 3.938
		v6 = 7.563
		v7 = 15.125
	)
	table := map[string]float64{
		PCIEVersion1: v1,
		PCIEVersion2: v2,
		PCIEVersion3: v3,
		PCIEVersion4: v4,
		PCIEVersion5: v5,
		PCIEVersion6: v6,
		PCIEVersion7: v7,
	}
	tp, ok := table[version]
	if ok {
		return PCIEVersionThroughput{
			Version:    version,
			Throughput: tp,
		}
	}
	return PCIEVersionThroughput{
		Version:    PCIEVersionUnknown,
		Throughput: -1,
	}
}

func (info *IsolatedDevicePCIEInfo) fillData() error {
	vTp := info.GetThroughputPerLane()
	info.Version = vTp.Version
	info.Throughput = fmt.Sprintf("%.2f GB/s", vTp.Throughput*float64(info.LaneWidth))
	return nil
}

func (info IsolatedDevicePCIEInfo) GetThroughputPerLane() PCIEVersionThroughput {
	table := map[string]PCIEVersionThroughput{
		// version 1.0: 2003
		"2.5": NewPCIEVersionThroughput(PCIEVersion1),
		// version 2.0: 2007
		"5":   NewPCIEVersionThroughput(PCIEVersion2),
		"5.0": NewPCIEVersionThroughput(PCIEVersion2),
		// version 3.0: 2010
		"8":   NewPCIEVersionThroughput(PCIEVersion3),
		"8.0": NewPCIEVersionThroughput(PCIEVersion3),
		// version 4.0: 2017
		"16":   NewPCIEVersionThroughput(PCIEVersion4),
		"16.0": NewPCIEVersionThroughput(PCIEVersion4),
		// version 5.0: 2019
		"32":   NewPCIEVersionThroughput(PCIEVersion5),
		"32.0": NewPCIEVersionThroughput(PCIEVersion5),
		// version 6.0: 2022
		"64":   NewPCIEVersionThroughput(PCIEVersion6),
		"64.0": NewPCIEVersionThroughput(PCIEVersion6),
		// version 7.0: 2025(planned)
		"128":   NewPCIEVersionThroughput(PCIEVersion7),
		"128.0": NewPCIEVersionThroughput(PCIEVersion7),
	}
	for key, val := range table {
		if fmt.Sprintf("%sGT/s", key) == info.TransferRatePerLane {
			return val
		}
	}
	return NewPCIEVersionThroughput(PCIEVersionUnknown)
}

type HostIsolatedDeviceModelDetails struct {
	SHostJointsBase
	HostJointResourceDetails
	// 宿主机Id
	HostId string `json:"host_id"`
	// 存储Id
	IsolatedDeviceModelId string `json:"isolated_device_model_id"`

	Model             string
	VendorId          string
	DeviceId          string
	DevType           string
	HotPluggable      bool
	DisableAutoDetect bool
}
