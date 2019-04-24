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

package modules

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SkusManager struct {
	ResourceManager
}

type ServerSkusManager struct {
	ResourceManager
}

var (
	CloudmetaSkus SkusManager
	ServerSkus    ServerSkusManager
)

func init() {
	CloudmetaSkus = SkusManager{NewCloudmetaManager("sku", "skus",
		[]string{},
		[]string{})}

	ServerSkus = ServerSkusManager{NewComputeManager("serversku", "serverskus",
		[]string{"ID", "Name", "Instance_type_family", "Instance_type_category", "Cpu_core_count",
			"Memory_size_mb", "Os_name", "Sys_disk_resizable", "Sys_disk_type",
			"Sys_disk_min_size_mb", "Sys_disk_max_size_mb", "Attached_disk_type",
			"Attached_disk_size_gb", "Attached_disk_count", "Data_disk_types",
			"Data_disk_max_count", "Nic_max_count", "Cloudregion_id", "Zone_id",
			"Provider", "Postpaid_status", "Prepaid_status", "Region", "Region_ext_id", "Zone", "Zone_ext_id"},
		[]string{"Total_guest_count"})}

	register(&CloudmetaSkus)
	registerCompute(&ServerSkus)
}

func (self *SkusManager) GetSkus(s *mcclient.ClientSession, providerId, regionId, zoneId string, limit, offset int) (*ListResult, error) {
	p := strings.ToLower(providerId)
	r := strings.ToLower(regionId)
	z := strings.ToLower(zoneId)
	url := fmt.Sprintf("/providers/%s/regions/%s/zones/%s/skus?limit=%d&offset=%d", p, r, z, limit, offset)
	ret, err := self._list(s, url, self.KeywordPlural)
	if err != nil {
		return &ListResult{}, err
	}

	return ret, nil
}
