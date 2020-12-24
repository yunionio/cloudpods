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
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SkusManager struct {
	modulebase.ResourceManager
}

type OfflineCloudmetaManager struct {
	modulebase.ResourceManager
}

type ServerSkusManager struct {
	modulebase.ResourceManager
}

type ElasticcacheSkusManager struct {
	modulebase.ResourceManager
}

var (
	CloudmetaSkus    SkusManager             // meta.yunion.io
	OfflineCloudmeta OfflineCloudmetaManager // aliyun offine sku&rate data
	ServerSkus       ServerSkusManager       // region service: server sku manager
	ElasticcacheSkus ElasticcacheSkusManager // region service: elasitc cache sku manager
)

func init() {
	CloudmetaSkus = SkusManager{NewCloudmetaManager("sku", "skus",
		[]string{},
		[]string{})}

	OfflineCloudmeta = OfflineCloudmetaManager{NewOfflineCloudmetaManager("", "",
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

	ElasticcacheSkus = ElasticcacheSkusManager{NewComputeManager("elasticcachesku", "elasticcacheskus",
		[]string{},
		[]string{})}

	register(&CloudmetaSkus)
	registerCompute(&ServerSkus)
	registerCompute(&ElasticcacheSkus)
}

func (self *SkusManager) GetSkus(s *mcclient.ClientSession, providerId, regionId, zoneId string, limit, offset int) (*modulebase.ListResult, error) {
	p := strings.ToLower(providerId)
	r := strings.ToLower(regionId)
	z := strings.ToLower(zoneId)
	url := fmt.Sprintf("/providers/%s/regions/%s/zones/%s/skus?limit=%d&offset=%d", p, r, z, limit, offset)
	ret, err := modulebase.List(self.ResourceManager, s, url, self.KeywordPlural)
	if err != nil {
		return &modulebase.ListResult{}, err
	}

	return ret, nil
}

func (self *OfflineCloudmetaManager) GetSkuSourcesMeta(s *mcclient.ClientSession, client *http.Client) (jsonutils.JSONObject, error) {
	baseUrl, err := s.GetServiceVersionURL(self.ServiceType(), self.EndpointType(), self.GetApiVersion())
	if err != nil {
		return nil, err
	}
	url := strings.TrimSuffix(baseUrl, "/") + "/sku.meta"
	_, body, err := httputils.JSONRequest(client, context.TODO(), "GET", url, nil, nil, false)
	return body, err
}
