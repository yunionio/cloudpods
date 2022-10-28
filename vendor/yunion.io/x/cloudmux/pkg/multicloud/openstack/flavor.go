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

package openstack

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SFlavor struct {
	multicloud.SServerSku
	OpenStackTags
	region *SRegion

	Id           string
	Disk         int
	Ephemeral    int
	ExtraSpecs   ExtraSpecs
	OriginalName string
	Name         string
	RAM          int
	Swap         string
	Vcpus        int
}

func (region *SRegion) GetFlavors() ([]SFlavor, error) {
	resource := "/flavors/detail"
	flavors := []SFlavor{}
	query := url.Values{}
	for {
		resp, err := region.ecsList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "ecsList")
		}
		part := struct {
			Flavors      []SFlavor
			FlavorsLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		flavors = append(flavors, part.Flavors...)

		marker := part.FlavorsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return flavors, nil
}

func (region *SRegion) GetFlavor(flavorId string) (*SFlavor, error) {
	resource := fmt.Sprintf("/flavors/%s", flavorId)
	resp, err := region.ecsGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "ecsGet")
	}
	flavor := &SFlavor{region: region}
	err = resp.Unmarshal(flavor, "flavor")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return flavor, nil
}

func (region *SRegion) SyncFlavor(name string, cpu, memoryMb, diskGB int) (string, error) {
	flavor, err := region.syncFlavor(name, cpu, memoryMb, diskGB)
	if err != nil {
		return "", errors.Wrapf(err, "syncFlavor")
	}
	return flavor.Id, nil
}

func (region *SRegion) syncFlavor(name string, cpu, memoryMb, diskGB int) (*SFlavor, error) {
	flavors, err := region.GetFlavors()
	if err != nil {
		return nil, errors.Wrapf(err, "GetFlavors")
	}

	if cpu == 0 && memoryMb == 0 {
		return nil, fmt.Errorf("failed to find instance type %s", name)
	}

	for i := range flavors {
		flavor := flavors[i]
		flavorName := flavor.GetName()
		if (len(name) == 0 || flavorName == name || flavorName == fmt.Sprintf("%s-%d", name, diskGB)) &&
			flavor.GetCpuCoreCount() == cpu &&
			flavor.GetMemorySizeMB() == memoryMb &&
			diskGB <= flavor.GetSysDiskMaxSizeGB() {
			return &flavor, nil
		}
	}
	if len(name) == 0 {
		name = fmt.Sprintf("ecs.g1.c%dm%d", cpu, memoryMb/1024)
	}
	name = fmt.Sprintf("%s-%d", name, diskGB)
	flavor, err := region.CreateFlavor(name, cpu, memoryMb, diskGB)
	if err != nil {
		return nil, errors.Wrap(err, "CreateFlavor")
	}
	return flavor, nil
}

func (region *SRegion) CreateISku(opts *cloudprovider.SServerSkuCreateOption) (cloudprovider.ICloudSku, error) {
	flavor, err := region.CreateFlavor(opts.Name, opts.CpuCount, opts.VmemSizeMb, opts.SysDiskMaxSizeGb)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateFlavor")
	}
	return flavor, nil
}

func (region *SRegion) CreateFlavor(name string, cpu int, memoryMb int, diskGB int) (*SFlavor, error) {
	if diskGB <= 0 {
		diskGB = 30
	}
	params := map[string]map[string]interface{}{
		"flavor": {
			"name":  name,
			"ram":   memoryMb,
			"vcpus": cpu,
			"disk":  diskGB,
		},
	}
	resp, err := region.ecsPost("/flavors", params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsPost")
	}
	flavor := &SFlavor{region: region}
	err = resp.Unmarshal(flavor, "flavor")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return flavor, nil
}

func (region *SRegion) DeleteFlavor(flavorId string) error {
	_, err := region.ecsDelete("/flavors/" + flavorId)
	return err
}

func (flavor *SFlavor) Delete() error {
	return flavor.region.DeleteFlavor(flavor.Id)
}

func (flavor *SFlavor) IsEmulated() bool {
	return false
}

func (flavor *SFlavor) Refresh() error {
	_flavor, err := flavor.region.GetFlavor(flavor.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(flavor, _flavor)
}

func (flavor *SFlavor) GetName() string {
	if len(flavor.OriginalName) > 0 {
		return flavor.OriginalName
	}
	return flavor.Name
}

func (flavor *SFlavor) GetId() string {
	return flavor.Id
}

func (flavor *SFlavor) GetGlobalId() string {
	return flavor.Id
}

func (flavor *SFlavor) GetInstanceTypeFamily() string {
	return api.InstanceFamilies[api.SkuCategoryGeneralPurpose]
}

func (flavor *SFlavor) GetInstanceTypeCategory() string {
	return api.SkuCategoryGeneralPurpose
}

func (flavor *SFlavor) GetPrepaidStatus() string {
	return api.SkuStatusSoldout
}

func (flavor *SFlavor) GetPostpaidStatus() string {
	return api.SkuStatusAvailable
}

func (flavor *SFlavor) GetCpuArch() string {
	return ""
}

func (flavor *SFlavor) GetCpuCoreCount() int {
	return int(flavor.Vcpus)
}

func (flavor *SFlavor) GetMemorySizeMB() int {
	return flavor.RAM
}

func (flavor *SFlavor) GetOsName() string {
	return "Any"
}

func (flavor *SFlavor) GetSysDiskResizable() bool {
	return true
}

func (flavor *SFlavor) GetSysDiskType() string {
	return ""
}

func (flavor *SFlavor) GetSysDiskMinSizeGB() int {
	return 0
}

func (flavor *SFlavor) GetSysDiskMaxSizeGB() int {
	return flavor.Disk
}

func (flavor *SFlavor) GetAttachedDiskType() string {
	return "iscsi"
}

func (flavor *SFlavor) GetAttachedDiskSizeGB() int {
	return 0
}

func (flavor *SFlavor) GetAttachedDiskCount() int {
	return 0
}

func (flavor *SFlavor) GetDataDiskTypes() string {
	return ""
}

func (flavor *SFlavor) GetDataDiskMaxCount() int {
	return 6
}

func (flavor *SFlavor) GetNicType() string {
	return "vpc"
}

func (flavor *SFlavor) GetNicMaxCount() int {
	return 1
}

func (flavor *SFlavor) GetGpuAttachable() bool {
	return false
}

func (flavor *SFlavor) GetGpuSpec() string {
	return ""
}

func (flavor *SFlavor) GetGpuCount() int {
	return 0
}

func (flavor *SFlavor) GetGpuMaxCount() int {
	return 0
}
