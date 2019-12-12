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

package ctyun

import "yunion.io/x/pkg/errors"

type SOrder struct {
	OrderItemID        string            `json:"orderItemId"`
	InstanceID         string            `json:"instanceId"`
	AccountID          string            `json:"accountId"`
	UserID             string            `json:"userId"`
	InnerOrderID       string            `json:"innerOrderId"`
	InnerOrderItemID   string            `json:"innerOrderItemId"`
	ProductID          string            `json:"productId"`
	MasterOrderID      string            `json:"masterOrderId"`
	OrderID            string            `json:"orderId"`
	MasterResourceID   string            `json:"masterResourceId"`
	ResourceID         string            `json:"resourceId"`
	ServiceTag         string            `json:"serviceTag"`
	ResourceType       string            `json:"resourceType"`
	ResourceInfo       string            `json:"resourceInfo"`
	StartDate          int64             `json:"startDate"`
	ExpireDate         int64             `json:"expireDate"`
	CreateDate         int64             `json:"createDate"`
	UpdateDate         int64             `json:"updateDate"`
	Status             int64             `json:"status"`
	WorkOrderID        string            `json:"workOrderId"`
	WorkOrderItemID    string            `json:"workOrderItemId"`
	SalesEntryID       string            `json:"salesEntryId"`
	OrderStatus        int64             `json:"orderStatus"`
	ToOndemand         string            `json:"toOndemand"`
	ItemValue          string            `json:"itemValue"`
	ChargingStatus     int64             `json:"chargingStatus"`
	ChargingDate       int64             `json:"chargingDate"`
	ResourceConfig     string            `json:"resourceConfig"`
	AutoToOnDemand     bool              `json:"autoToOnDemand"`
	BuildingChannel    int64             `json:"buildingChannel"`
	IsPlatformSpecific bool              `json:"isPlatformSpecific"`
	BillingOwner       int64             `json:"billingOwner"`
	IsPackage          bool              `json:"isPackage"`
	CanRelease         bool              `json:"canRelease"`
	IsChargeOff        bool              `json:"isChargeOff"`
	IsPublicTest       int64             `json:"isPublicTest"`
	Master             bool              `json:"master"`
	ResourceConfigMap  ResourceConfigMap `json:"resourceConfigMap"`
}

type ResourceConfigMap struct {
	AvailabilityZone string            `json:"availability_zone"`
	Value            string            `json:"value"`
	Number           string            `json:"number"`
	IsSystemVolume   bool              `json:"isSystemVolume"`
	VolumeType       string            `json:"volumeType"`
	Size             int64             `json:"size"`
	ZoneID           string            `json:"zoneId"`
	RegionID         string            `json:"regionId"`
	Version          string            `json:"version"`
	CycleCnt         int64             `json:"cycleCnt"`
	CycleType        int64             `json:"cycleType"`
	ResEbsID         string            `json:"resEbsId"`
	ActualResourceID string            `json:"actualResourceId"`
	SecurityGroups   []SecurityGroupID `json:"security_groups"`
}

type SecurityGroupID struct {
	ID string `json:"id"`
}

func (self *SRegion) GetOrder(orderId string) ([]SOrder, error) {
	params := map[string]string{
		"masterOrderId": orderId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/order/queryResourceInfoByMasterOrderId", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetOrder.DoGet")
	}

	ret := make([]SOrder, 0)
	err = resp.Unmarshal(&ret, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetOrder.DoGet")
	}

	return ret, nil
}
