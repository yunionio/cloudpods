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

package diskutils

import (
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type IDisk interface {
	Connect() error
	Disconnect() error
	MountRootfs() fsdriver.IRootFsDriver
	UmountRootfs(driver fsdriver.IRootFsDriver)
}

func GetIDisk(params *apis.DeployParams) IDisk {
	hypervisor := params.GuestDesc.Hypervisor
	switch hypervisor {
	case comapi.HYPERVISOR_KVM:
		return NewKVMGuestDisk(params.DiskPath)
	case comapi.HYPERVISOR_ESXI:
		return NewVDDKDisk(params.VddkInfo, params.DiskPath)
	default:
		return NewKVMGuestDisk(params.DiskPath)
	}
}
