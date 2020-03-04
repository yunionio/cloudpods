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

type RegionalResourceCreateInput struct {
	Cloudregion   string `json:"cloudregion"`
	CloudregionId string `json:"cloudregion_id"`
}

type ManagedResourceCreateInput struct {
	Manager   string `json:"manager"`
	ManagerId string `json:"manager_id"`
}

type DeletePreventableCreateInput struct {
	//删除保护,创建的资源默认不允许删除
	//default: true
	DisableDelete *bool `json:"disable_delete"`
}

type KeypairListInput struct {
	apis.StandaloneResourceListInput

	apis.UserResourceListInput

	// list in admin mode
	Admin *bool `json:"admin"`
}

type CachedimageListInput struct {
	apis.StandaloneResourceListInput

	ManagedResourceListInput
	ZonalFilterListInput
}

type ExternalProjectListInput struct {
	apis.StandaloneResourceListInput
	apis.ProjectizedResourceListInput

	ManagedResourceListInput
}

type RouteTableListInput struct {
	apis.VirtualResourceListInput

	VpcFilterListInput
}

type SnapshotPolicyCacheListInput struct {
	apis.StatusStandaloneResourceListInput
	ManagedResourceListInput
	RegionalFilterListInput

	// filter by snapshotpolicy Id or Name
	Snapshotpolicy string `json:"snapshotpolicy"`
}

type NetworkInterfaceListInput struct {
	apis.StatusStandaloneResourceListInput

	ManagedResourceListInput
	RegionalFilterListInput
}

type BaremetalagentListInput struct {
	apis.StandaloneResourceListInput
	ZonalFilterListInput
}

type DnsRecordListInput struct {
	apis.AdminSharableVirtualResourceListInput
}

type DynamicschedtagListInput struct {
	apis.StandaloneResourceListInput
	SchedtagFilterListInput
}

type GuestTemplateListInput struct {
	apis.SharableVirtualResourceListInput
}

type SchedpolicyListInput struct {
	apis.StandaloneResourceListInput
	SchedtagFilterListInput
}

type ServiceCatalogListInput struct {
	apis.SharableVirtualResourceListInput
}

type SnapshotPolicyListInput struct {
	apis.VirtualResourceListInput
}

type DnsRecordDetails struct {
	apis.AdminSharableVirtualResourceDetails

	SDnsRecord
}
