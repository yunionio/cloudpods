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

package huawei

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
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

/*
获取订单信息  https://support.huaweicloud.com/api-oce/api_order_00001.html
*/
func (self *SRegion) GetOrder(orderId string) (SOrder, error) {
	var order SOrder
	domain, err := self.getDomianId()
	if err != nil {
		return order, err
	}

	err = self.ecsClient.Orders.SetDomainId(domain)
	if err != nil {
		return order, err
	}

	err = DoGet(self.ecsClient.Orders.Get, orderId, nil, &order)
	return order, err
}

/*
获取订单资源详情列表 https://support.huaweicloud.com/api-oce/zh-cn_topic_0084961226.html
*/
func (self *SRegion) GetOrderResources(orderId string, resource_ids []string, only_main_resource bool) ([]SResourceDetail, error) {
	domain, err := self.getDomianId()
	if err != nil {
		return nil, err
	}

	err = self.ecsClient.Orders.SetDomainId(domain)
	if err != nil {
		return nil, err
	}

	resources := make([]SResourceDetail, 0)
	queries := map[string]string{"customer_id": domain}
	if len(orderId) > 0 {
		queries["order_id"] = orderId
	}

	if len(resource_ids) > 0 {
		queries["resource_ids"] = strings.Join(resource_ids, ",")
	}

	if only_main_resource {
		queries["only_main_resource"] = "1"
	}

	err = doListAll(self.ecsClient.Orders.GetPeriodResourceList, queries, &resources)
	return resources, err
}

/*
获取资源详情 https://support.huaweicloud.com/api-oce/zh-cn_topic_0084961226.html
*/
func (self *SRegion) GetOrderResourceDetail(resourceId string) (SResourceDetail, error) {
	var res SResourceDetail
	if len(resourceId) == 0 {
		return res, fmt.Errorf("GetOrderResourceDetail resource id should not be empty")
	}

	resources, err := self.GetOrderResources("", []string{resourceId}, false)
	if err != nil {
		return res, err
	}

	switch len(resources) {
	case 0:
		return res, cloudprovider.ErrNotFound
	case 1:
		return resources[0], nil
	default:
		return res, fmt.Errorf("%d resources with id %s found, Expect 1", len(resources), resourceId)
	}
}

func (self *SRegion) GetAllResByOrderId(orderId string) ([]SResource, error) {
	order, err := self.GetOrder(orderId)
	if err != nil {
		return nil, err
	}

	log.Debugf("GetAllResByOrderId %#v", order.Resources)
	return order.Resources, nil
}

func (self *SRegion) getAllResByType(orderId string, resourceType string) ([]SResource, error) {
	res, err := self.GetAllResByOrderId(orderId)
	if err != nil {
		return nil, err
	}

	ret := make([]SResource, 0)
	for i := range res {
		r := res[i]
		if r.ResourceType == resourceType {
			ret = append(ret, r)
		}
	}

	return ret, nil
}

func (self *SRegion) getAllResIdsByType(orderId string, resourceType string) ([]string, error) {
	res, err := self.getAllResByType(orderId, resourceType)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0)
	for _, r := range res {
		if len(r.ResourceID) > 0 {
			ids = append(ids, r.ResourceID)
		}
	}

	return ids, nil
}
