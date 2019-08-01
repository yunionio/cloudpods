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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SFlavor struct {
	region       *SRegion
	ID           string
	Disk         int
	Ephemeral    int
	ExtraSpecs   ExtraSpecs
	OriginalName string
	Name         string
	RAM          int
	Swap         string
	Vcpus        int8
}

func (region *SRegion) GetFlavors() ([]SFlavor, error) {
	url := "/flavors/detail"
	flavors := []SFlavor{}
	for len(url) > 0 {
		_, resp, err := region.List("compute", url, "", nil)
		if err != nil {
			return nil, err
		}
		_flavors := []SFlavor{}
		err = resp.Unmarshal(&_flavors, "flavors")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_flavors, "flavors")`)
		}
		flavors = append(flavors, _flavors...)
		url = ""
		if resp.Contains("flavors_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "flavors_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "flavors_links")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}
	return flavors, nil
}

func (region *SRegion) GetFlavor(flavorId string) (*SFlavor, error) {
	_, resp, err := region.Get("compute", "/flavors/"+flavorId, "", nil)
	if err != nil {
		return nil, err
	}
	flavor := &SFlavor{region: region}
	return flavor, resp.Unmarshal(flavor, "flavor")
}

func (region *SRegion) SyncFlavor(name string, cpu, memoryMb, diskGB int) (string, error) {
	return region.syncFlavor(name, cpu, memoryMb, diskGB)
}

func (region *SRegion) syncFlavor(name string, cpu, memoryMb, diskGB int) (string, error) {
	flavors, err := region.GetFlavors()
	if err != nil {
		return "", err
	}
	if len(name) > 0 {
		for _, flavor := range flavors {
			flavorName := flavor.GetName()
			if flavorName == name {
				return flavor.ID, nil
			}
		}
	}

	if cpu == 0 && memoryMb == 0 {
		return "", fmt.Errorf("failed to find instance type %s", name)
	}

	for _, flavor := range flavors {
		if flavor.GetCpuCoreCount() == cpu && flavor.GetMemorySizeMB() == memoryMb {
			return flavor.ID, nil
		}
	}
	return "", fmt.Errorf("failed to find right flavor(name: %s cpu: %d memory: %d)", name, cpu, memoryMb)
}

func (region *SRegion) CreateFlavor(name string, cpu int, memoryMb int, diskGB int) (*SFlavor, error) {
	if diskGB < 30 {
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
	_, resp, err := region.Post("compute", "/flavors", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	flavor := &SFlavor{}
	return flavor, resp.Unmarshal(flavor, "flavor")
}

func (region *SRegion) DeleteFlavor(flavorId string) error {
	_, err := region.Delete("compute", "/flavors/"+flavorId, "")
	return err
}

func (flavor *SFlavor) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (flavor *SFlavor) IsEmulated() bool {
	return false
}

func (flavor *SFlavor) Refresh() error {
	new, err := flavor.region.GetFlavor(flavor.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(flavor, new)
}

func (flavor *SFlavor) GetName() string {
	if len(flavor.OriginalName) > 0 {
		return flavor.OriginalName
	}
	return flavor.Name
}

func (flavor *SFlavor) GetStatus() string {
	return ""
}

func (flavor *SFlavor) GetId() string {
	return flavor.ID
}

func (flavor *SFlavor) GetGlobalId() string {
	return flavor.ID
}

func (flavor *SFlavor) GetInstanceTypeFamily() string {
	return flavor.GetName()
}

func (flavor *SFlavor) GetInstanceTypeCategory() string {
	return flavor.GetName()
}

func (flavor *SFlavor) GetPrepaidStatus() string {
	return api.SkuStatusSoldout
}

func (flavor *SFlavor) GetPostpaidStatus() string {
	return api.SkuStatusAvailable
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
	return "iscsi"
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
	return 6
}

func (flavor *SFlavor) GetDataDiskTypes() string {
	return "iscsi"
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
