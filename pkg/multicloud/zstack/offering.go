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

package zstack

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SInstanceOffering struct {
	multicloud.SServerSku
	region *SRegion

	ZStackBasic
	MemorySize        int    `json:"memorySize"`
	CPUNum            int    `json:"cpuNum"`
	CPUSpeed          int    `json:"cpuSpeed"`
	Type              string `json:"type"`
	AllocatorStrategy string `json:"allocatorStrategy"`
	State             string `json:"state"`

	ZStackTime
}

func (region *SRegion) GetInstanceOffering(offerId string) (*SInstanceOffering, error) {
	offer := &SInstanceOffering{region: region}
	return offer, region.client.getResource("instance-offerings", offerId, offer)
}

func (region *SRegion) GetInstanceOfferingByType(instanceType string) (*SInstanceOffering, error) {
	offerings, err := region.GetInstanceOfferings("", instanceType, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(offerings) >= 1 {
		return &offerings[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateISku(name string, vCpu int, memoryMb int) error {
	_, err := region.CreateInstanceOffering(name, vCpu, memoryMb, "UserVm")
	return err
}

func (region *SRegion) CreateInstanceOffering(name string, cpu int, memoryMb int, offeringType string) (*SInstanceOffering, error) {
	parmas := map[string]interface{}{
		"params": map[string]interface{}{
			"name":       name,
			"cpuNum":     cpu,
			"memorySize": memoryMb * 1024 * 1024,
			"type":       offeringType,
		},
	}
	resp, err := region.client.post("instance-offerings", jsonutils.Marshal(parmas))
	if err != nil {
		return nil, errors.Wrapf(err, "CreateInstanceOffering")
	}
	offering := &SInstanceOffering{region: region}
	err = resp.Unmarshal(offering, "inventory")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return offering, nil
}

func (region *SRegion) GetInstanceOfferings(offerId string, name string, cpu int, memorySizeMb int) ([]SInstanceOffering, error) {
	offerings := []SInstanceOffering{}
	params := url.Values{}
	params.Add("q", "type=UserVM")
	params.Add("q", "state=Enabled")
	if len(offerId) > 0 {
		params.Add("q", "uid="+offerId)
	}
	if len(name) > 0 {
		params.Add("q", "name="+name)
	}
	if cpu != 0 {
		params.Add("q", fmt.Sprintf("cpuNum=%d", cpu))
	}
	if memorySizeMb != 0 {
		params.Add("q", fmt.Sprintf("memorySize=%d", memorySizeMb*1024*1024))
	}
	if err := region.client.listAll("instance-offerings", params, &offerings); err != nil {
		return nil, err
	}
	for i := 0; i < len(offerings); i++ {
		offerings[i].region = region
	}
	return offerings, nil
}

func (offering *SInstanceOffering) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (offering *SInstanceOffering) IsEmulated() bool {
	return false
}

func (offering *SInstanceOffering) Refresh() error {
	new, err := offering.region.GetInstanceOffering(offering.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(offering, new)
}

func (offering *SInstanceOffering) GetName() string {
	return offering.Name
}

func (offering *SInstanceOffering) GetStatus() string {
	switch offering.State {
	case "Enabled":
		return api.SkuStatusReady
	}
	return api.SkuStatusSoldout
}

func (offering *SInstanceOffering) GetId() string {
	return offering.UUID
}

func (offering *SInstanceOffering) GetGlobalId() string {
	return offering.UUID
}

func (offering *SInstanceOffering) Delete() error {
	return offering.region.DeleteOffering(offering.UUID)
}

func (region *SRegion) DeleteOffering(offeringId string) error {
	return region.client.delete("instance-offerings", offeringId, "")
}

func (offering *SInstanceOffering) GetInstanceTypeFamily() string {
	return offering.AllocatorStrategy
}

func (offering *SInstanceOffering) GetInstanceTypeCategory() string {
	return offering.AllocatorStrategy
}

func (offering *SInstanceOffering) GetPrepaidStatus() string {
	return api.SkuStatusSoldout
}

func (offering *SInstanceOffering) GetPostpaidStatus() string {
	return api.SkuStatusAvailable
}

func (offering *SInstanceOffering) GetCpuCoreCount() int {
	return offering.CPUNum
}

func (offering *SInstanceOffering) GetMemorySizeMB() int {
	return offering.MemorySize / 1024 / 1024
}

func (offering *SInstanceOffering) GetOsName() string {
	return "Any"
}

func (offering *SInstanceOffering) GetSysDiskResizable() bool {
	return true
}

func (offering *SInstanceOffering) GetSysDiskType() string {
	return ""
}

func (offering *SInstanceOffering) GetSysDiskMinSizeGB() int {
	return 0
}

func (offering *SInstanceOffering) GetSysDiskMaxSizeGB() int {
	return 0
}

func (offering *SInstanceOffering) GetAttachedDiskType() string {
	return ""
}

func (offering *SInstanceOffering) GetAttachedDiskSizeGB() int {
	return 0
}

func (offering *SInstanceOffering) GetAttachedDiskCount() int {
	return 6
}

func (offering *SInstanceOffering) GetDataDiskTypes() string {
	return ""
}

func (offering *SInstanceOffering) GetDataDiskMaxCount() int {
	return 6
}

func (offering *SInstanceOffering) GetNicType() string {
	return "vpc"
}

func (offering *SInstanceOffering) GetNicMaxCount() int {
	return 1
}

func (offering *SInstanceOffering) GetGpuAttachable() bool {
	return false
}

func (offering *SInstanceOffering) GetGpuSpec() string {
	return ""
}

func (offering *SInstanceOffering) GetGpuCount() int {
	return 0
}

func (offering *SInstanceOffering) GetGpuMaxCount() int {
	return 0
}
