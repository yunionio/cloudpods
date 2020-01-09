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
	"yunion.io/x/onecloud/pkg/apis"
)

type HostSpec struct {
	apis.Meta

	Cpu         int            `json:"cpu"`
	Mem         int            `json:"mem"`
	NicCount    int            `json:"nic_count"`
	Manufacture string         `json:"manufacture"`
	Model       string         `json:"model"`
	Disk        DiskDriverSpec `json:"disk"`
	Driver      string         `json:"driver"`
}

type DiskDriverSpec map[string]DiskAdapterSpec

type DiskAdapterSpec map[string][]*DiskSpec

type DiskSpec struct {
	apis.Meta

	Type       string `json:"type"`
	Size       int64  `json:"size"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Count      int    `json:"count"`
}

type HostListInput struct {
	apis.EnabledStatusStandaloneResourceListInput
	apis.DomainizedResourceListInput

	ManagedResourceListInput
	ZonalFilterListInput
	WireFilterListInput
	SchedtagFilterListInput

	StorageFilterListInput
	UsableResourceListInput

	// filter by ResourceType
	ResourceType string `json:"resource_type"`
	// filter by mac of any network interface
	AnyMac string `json:"any_mac"`
	// filter storages not attached to this host
	StorageNotAttached *bool `json:"storage_not_attached"`
	// filter by Hypervisor
	Hypervisor string `json:"hypervisor"`
	// filter host that is empty
	IsEmpty *bool `json:"is_empty"`
	// filter host that is baremetal
	Baremetal *bool `json:"baremetal"`
}
