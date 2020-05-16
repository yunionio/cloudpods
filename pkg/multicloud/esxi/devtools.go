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

package esxi

import (
	"fmt"
	"reflect"

	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/pkg/errors"
)

func NewDiskDev(sizeMb int64, templatePath string, uuid string, index int32, key int32, controlKey int32) *types.VirtualDisk {
	device := types.VirtualDisk{}

	var backFile *types.VirtualDiskFlatVer2BackingInfo
	if len(templatePath) > 0 {
		backFile = &types.VirtualDiskFlatVer2BackingInfo{}
		backFile.FileName = templatePath
	}

	diskFile := types.VirtualDiskFlatVer2BackingInfo{}
	diskFile.DiskMode = "persistent"
	thinProvisioned := true
	diskFile.ThinProvisioned = &thinProvisioned
	diskFile.Uuid = uuid
	if backFile != nil {
		diskFile.Parent = backFile
	}

	device.Backing = &diskFile

	if sizeMb > 0 {
		device.CapacityInKB = sizeMb * 1024
	}

	device.ControllerKey = controlKey
	device.Key = key + index
	device.UnitNumber = &index

	return &device
}

func addDevSpec(device types.BaseVirtualDevice) *types.VirtualDeviceConfigSpec {
	spec := types.VirtualDeviceConfigSpec{}
	spec.Operation = types.VirtualDeviceConfigSpecOperationAdd
	spec.Device = device
	return &spec
}

func NewSCSIDev(key, ctlKey int32, driver string) types.BaseVirtualDevice {
	desc := types.Description{Label: "SCSI controller 0", Summary: "VMware virtual SCSI"}

	if driver == "pvscsi" {
		device := types.ParaVirtualSCSIController{}
		device.DeviceInfo = &desc
		device.Key = key
		device.ControllerKey = ctlKey
		device.SharedBus = "noSharing"
		return &device
	}
	device := types.VirtualLsiLogicController{}
	device.DeviceInfo = &desc
	device.Key = key
	device.ControllerKey = ctlKey
	device.SharedBus = "noSharing"
	return &device
}

func NewAHCIDev(key, ctlKey int32) types.BaseVirtualDevice {
	device := types.VirtualAHCIController{}
	device.DeviceInfo = &types.Description{Label: "SATA controller 0", Summary: "AHCI"}
	device.ControllerKey = ctlKey
	device.Key = key
	return &device
}

func NewSVGADev(key, ctlKey int32) types.BaseVirtualDevice {
	device := types.VirtualMachineVideoCard{}
	device.DeviceInfo = &types.Description{Label: "Video card", Summary: "Video card"}
	device.ControllerKey = ctlKey
	device.Key = key
	device.VideoRamSizeInKB = 16 * 1024
	return &device
}

func NewIDEDev(key, index int32) types.BaseVirtualDevice {
	device := types.VirtualIDEController{}
	s := fmt.Sprintf("IDE %d", index)
	device.DeviceInfo = &types.Description{Label: s, Summary: s}
	device.Key = key + index
	device.BusNumber = index
	return &device
}

func NewCDROMDev(path string, key, ctlKey int32) types.BaseVirtualDevice {
	device := types.VirtualCdrom{}
	device.DeviceInfo = &types.Description{Label: "CD/DVD drive 1", Summary: "Local ISO Emulated CD-ROM"}
	device.ControllerKey = ctlKey
	device.Key = key

	connectable := types.VirtualDeviceConnectInfo{AllowGuestControl: true, Status: "untried"}
	if len(path) != 0 {
		device.Backing = &types.VirtualCdromIsoBackingInfo{types.VirtualDeviceFileBackingInfo{FileName: path}}
		connectable.StartConnected = true
	} else {
		device.Backing = &types.VirtualCdromRemoteAtapiBackingInfo{}
		connectable.StartConnected = false
	}

	device.Connectable = &connectable
	return &device
}

func NewUSBController(key *int32) types.BaseVirtualDevice {
	device := types.VirtualUSBController{}
	device.DeviceInfo = &types.Description{
		Label: "USB controller",
	}
	if key != nil {
		device.Key = *key
	}
	return &device
}

func NewVNICDev(host *SHost, mac, driver string, vlanId int32, key, ctlKey, index int32) (types.BaseVirtualDevice, error) {
	desc := types.Description{Label: fmt.Sprintf("Network adapter %d", index+1), Summary: "VM Network"}

	inet, err := host.FindNetworkByVlanID(vlanId)
	if err != nil {
		return nil, errors.Wrap(err, "SHost.FindNetworkByVlanID")
	}
	if inet == nil || reflect.ValueOf(inet).IsNil() {
		return nil, errors.Error(fmt.Sprintf("VLAN %d not found", vlanId))
	}

	var (
		False = false
		True  = true
	)
	var backing types.BaseVirtualDeviceBackingInfo
	switch inet.(type) {
	case *SDistributedVirtualPortgroup:
		net := inet.(*SDistributedVirtualPortgroup)
		port, err := net.FindPort()
		if err != nil {
			return nil, errors.Wrap(err, "net.FindPort")
		}
		if port == nil {
			return nil, errors.Error("no valid port on DVS, exhausted")
		}
		portCon := types.DistributedVirtualSwitchPortConnection{
			PortgroupKey: port.PortgroupKey,
			SwitchUuid:   port.DvsUuid,
			PortKey:      port.Key,
		}
		backing = &types.VirtualEthernetCardDistributedVirtualPortBackingInfo{Port: portCon}
	case *SNetwork:
		monet := inet.(*SNetwork).getMONetwork()
		backing = &types.VirtualEthernetCardNetworkBackingInfo{
			VirtualDeviceDeviceBackingInfo: types.VirtualDeviceDeviceBackingInfo{
				DeviceName:    monet.Name,
				UseAutoDetect: &False,
			},
			Network: &monet.Self,
		}
	default:
		return nil, errors.Error(fmt.Sprintf("Unsuppport network type %s", reflect.TypeOf(inet).Name()))
	}

	connectable := types.VirtualDeviceConnectInfo{
		StartConnected:    true,
		AllowGuestControl: true,
		Connected:         false,
		Status:            "untried",
	}

	nic := types.VirtualEthernetCard{
		VirtualDevice: types.VirtualDevice{
			DeviceInfo: &desc,
			Backing:    backing,
		},
		WakeOnLanEnabled: &True,
	}
	nic.Connectable = &connectable
	nic.ControllerKey = ctlKey
	nic.Key = key + index
	if len(mac) != 0 {
		nic.AddressType = "Manual"
		nic.MacAddress = mac
	} else {
		nic.AddressType = "Generated"
	}
	if driver == "e1000" {
		return &types.VirtualE1000{nic}, nil
	}

	return &types.VirtualVmxnet3{types.VirtualVmxnet{nic}}, nil
}
