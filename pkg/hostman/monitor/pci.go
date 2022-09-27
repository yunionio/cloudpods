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

package monitor

// PciBridgeInfo -> PCIBridgeInfo (struct)

// PCIBridgeInfo implements the "PciBridgeInfo" QMP API type.
type PCIBridgeInfo struct {
	Bus     PCIBusInfo      `json:"bus"`
	Devices []PCIDeviceInfo `json:"devices,omitempty"`
}

// PciBusInfo -> PCIBusInfo (struct)

// PCIBusInfo implements the "PciBusInfo" QMP API type.
type PCIBusInfo struct {
	Number            int64          `json:"number"`
	Secondary         int64          `json:"secondary"`
	Subordinate       int64          `json:"subordinate"`
	IORange           PCIMemoryRange `json:"io_range"`
	MemoryRange       PCIMemoryRange `json:"memory_range"`
	PrefetchableRange PCIMemoryRange `json:"prefetchable_range"`
}

// PciDeviceClass -> PCIDeviceClass (struct)

// PCIDeviceClass implements the "PciDeviceClass" QMP API type.
type PCIDeviceClass struct {
	Desc  *string `json:"desc,omitempty"`
	Class int64   `json:"class"`
}

// PciDeviceId -> PCIDeviceID (struct)

// PCIDeviceID implements the "PciDeviceId" QMP API type.
type PCIDeviceID struct {
	Device int64 `json:"device"`
	Vendor int64 `json:"vendor"`
}

// PciDeviceInfo -> PCIDeviceInfo (struct)

// PCIDeviceInfo implements the "PciDeviceInfo" QMP API type.
type PCIDeviceInfo struct {
	Bus       int64             `json:"bus"`
	Slot      int64             `json:"slot"`
	Function  int64             `json:"function"`
	ClassInfo PCIDeviceClass    `json:"class_info"`
	ID        PCIDeviceID       `json:"id"`
	Irq       *int64            `json:"irq,omitempty"`
	QdevID    string            `json:"qdev_id"`
	PCIBridge *PCIBridgeInfo    `json:"pci_bridge,omitempty"`
	Regions   []PCIMemoryRegion `json:"regions"`
}

// PciInfo -> PCIInfo (struct)

// PCIInfo implements the "PciInfo" QMP API type.
type PCIInfo struct {
	Bus     int64           `json:"bus"`
	Devices []PCIDeviceInfo `json:"devices"`
}

// PciMemoryRange -> PCIMemoryRange (struct)

// PCIMemoryRange implements the "PciMemoryRange" QMP API type.
type PCIMemoryRange struct {
	Base  int64 `json:"base"`
	Limit int64 `json:"limit"`
}

// PciMemoryRegion -> PCIMemoryRegion (struct)

// PCIMemoryRegion implements the "PciMemoryRegion" QMP API type.
type PCIMemoryRegion struct {
	Bar       int64  `json:"bar"`
	Type      string `json:"type"`
	Address   int64  `json:"address"`
	Size      int64  `json:"size"`
	Prefetch  *bool  `json:"prefetch,omitempty"`
	MemType64 *bool  `json:"mem_type_64,omitempty"`
}

type QueryPciCallback func(pciInfoList []PCIInfo, err string)

type MemoryDeviceInfo struct {
	Type string
	Data PcdimmDeviceInfo
}

type PcdimmDeviceInfo struct {
	ID           *string `json:"id,omitempty"`
	Addr         int64   `json:"addr"`
	Size         int64   `json:"size"`
	Slot         int64   `json:"slot"`
	Node         int64   `json:"node"`
	Memdev       string  `json:"memdev"`
	Hotplugged   bool    `json:"hotplugged"`
	Hotpluggable bool    `json:"hotpluggable"`
}

type QueryMemoryDevicesCallback func(memoryDevicesInfoList []MemoryDeviceInfo, err string)

// MachineInfo implements the "MachineInfo" QMP API type.
type MachineInfo struct {
	Name             string  `json:"name"`
	Alias            *string `json:"alias,omitempty"`
	IsDefault        *bool   `json:"is-default,omitempty"`
	CPUMax           int64   `json:"cpu-max"`
	HotpluggableCpus bool    `json:"hotpluggable-cpus"`
}

type QueryMachinesCallback func(machineInfoList []MachineInfo, err string)
