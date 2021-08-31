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

package hcso

import (
	"fmt"
	"time"
)

type SOrder struct {
	ErrorCode *string     `json:"error_code"` // 只有失败时才返回此参数
	ErrorMsg  *string     `json:"error_msg"`  //只有失败时才返回此参数
	TotalSize int         `json:"totalSize"`  // 只有成功时才返回此参数
	Resources []SResource `json:"resources"`
}

type SResource struct {
	ResourceID       string `json:"resourceId"`
	CloudServiceType string `json:"cloudServiceType"`
	RegionCode       string `json:"regionCode"`
	ResourceType     string `json:"resourceType"`
	ResourceSpecCode string `json:"resourceSpecCode"`
	Status           int64  `json:"status"`
}

type SResourceDetail struct {
	ID                   string    `json:"id"`
	Status               int64     `json:"status"`
	ResourceID           string    `json:"resource_id"`
	ResourceName         string    `json:"resource_name"`
	RegionCode           string    `json:"region_code"`
	CloudServiceTypeCode string    `json:"cloud_service_type_code"`
	ResourceTypeCode     string    `json:"resource_type_code"`
	ResourceSpecCode     string    `json:"resource_spec_code"`
	ProjectCode          string    `json:"project_code"`
	ProductID            string    `json:"product_id"`
	MainResourceID       string    `json:"main_resource_id"`
	IsMainResource       int64     `json:"is_main_resource"`
	ValidTime            time.Time `json:"valid_time"`
	ExpireTime           time.Time `json:"expire_time"`
	NextOperationPolicy  string    `json:"next_operation_policy"`
}

func (self *SRegion) getDomianId() (string, error) {
	domains, err := self.client.getEnabledDomains()
	if err != nil {
		return "", err
	}

	if domains == nil || len(domains) == 0 {
		return "", fmt.Errorf("GetAllResByOrderId domain is empty")
	} else if len(domains) > 1 {
		// not supported??
		return "", fmt.Errorf("GetAllResByOrderId mutliple domain(%d) found", len(domains))
	}

	return domains[0].ID, nil
}
