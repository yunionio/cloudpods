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
	"github.com/vmware/govmomi/vim25/types"
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
