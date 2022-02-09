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

package aliyun

import (
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SCenRouteEntries struct {
	PageNumber      int             `json:"PageNumber"`
	TotalCount      int             `json:"TotalCount"`
	PageSize        int             `json:"PageSize"`
	RequestID       string          `json:"RequestId"`
	CenRouteEntries CenRouteEntries `json:"CenRouteEntries"`
}
type CenRouteMapRecord struct {
	RouteMapID string `json:"RouteMapId"`
	RegionID   string `json:"RegionId"`
}
type CenRouteMapRecords struct {
	CenRouteMapRecord []CenRouteMapRecord `json:"CenRouteMapRecord"`
}
type AsPaths struct {
	AsPath []string `json:"AsPath"`
}
type Communities struct {
	Community []string `json:"Community"`
}
type Conflicts struct {
	Conflict []Conflict `json:"Conflict"`
}

type Conflict struct {
	DestinationCidrBlock string `json:"DestinationCidrBlock"`
	InstanceId           string `json:"InstanceId"`
	InstanceType         string `json:"InstanceType"`
	RegionId             string `json:"RegionId"`
	Status               string `json:"Status"`
}

type SCenRouteEntry struct {
	multicloud.SResourceBase
	multicloud.AliyunTags
	ChildInstance        *SCenChildInstance
	NextHopInstanceID    string             `json:"NextHopInstanceId,omitempty"`
	Status               string             `json:"Status"`
	OperationalMode      bool               `json:"OperationalMode"`
	CenRouteMapRecords   CenRouteMapRecords `json:"CenRouteMapRecords"`
	AsPaths              AsPaths            `json:"AsPaths"`
	Communities          Communities        `json:"Communities"`
	Type                 string             `json:"Type"`
	NextHopType          string             `json:"NextHopType"`
	NextHopRegionID      string             `json:"NextHopRegionId,omitempty"`
	RouteTableID         string             `json:"RouteTableId"`
	DestinationCidrBlock string             `json:"DestinationCidrBlock"`
	Conflicts            Conflicts          `json:"Conflicts"`
	PublishStatus        string             `json:"PublishStatus,omitempty"`
}
type CenRouteEntries struct {
	CenRouteEntry []SCenRouteEntry `json:"CenRouteEntry"`
}

func (client *SAliyunClient) DescribeCenChildInstanceRouteEntries(cenId string, childInstanceId string, childInstanceRegion string, childInstanceType string, pageNumber int, pageSize int) (SCenRouteEntries, error) {
	routeEntries := SCenRouteEntries{}
	params := map[string]string{}
	params["CenId"] = cenId
	params["ChildInstanceId"] = childInstanceId
	params["ChildInstanceRegionId"] = childInstanceRegion
	params["ChildInstanceType"] = childInstanceType
	params["Action"] = "DescribeCenChildInstanceRouteEntries"
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.cbnRequest("DescribeCenChildInstanceRouteEntries", params)
	if err != nil {
		return routeEntries, errors.Wrapf(err, `client.cbnRequest("DescribeCenChildInstanceRouteEntries", %s)`, jsonutils.Marshal(params).String())
	}
	err = resp.Unmarshal(&routeEntries)
	if err != nil {
		return routeEntries, errors.Wrapf(err, "[%s].Unmarshal(&routeEntries)", resp.String())
	}
	return routeEntries, nil
}

func (client *SAliyunClient) GetAllCenChildInstanceRouteEntries(cenId, childInstanceId, childInstanceRegion, childInstanceType string) ([]SCenRouteEntry, error) {
	pageNumber := 0
	srouteEntries := []SCenRouteEntry{}
	for {
		pageNumber++
		routeEntries, err := client.DescribeCenChildInstanceRouteEntries(cenId, childInstanceId, childInstanceRegion, childInstanceType, pageNumber, 20)
		if err != nil {
			return nil, errors.Wrap(err, "client.DescribeCenChildInstanceRouteEntries(cenId, childInstanceId, pageNumber, 20)")
		}
		srouteEntries = append(srouteEntries, routeEntries.CenRouteEntries.CenRouteEntry...)
		if len(srouteEntries) >= routeEntries.TotalCount {
			break
		}
	}
	return srouteEntries, nil
}

func (client *SAliyunClient) PublishRouteEntries(cenId, childInstanceId, routeTableId, childInstanceRegion, childInstanceType, cidr string) error {
	params := map[string]string{}
	params["CenId"] = cenId
	params["ChildInstanceId"] = childInstanceId
	params["ChildInstanceRouteTableId"] = routeTableId
	params["ChildInstanceRegionId"] = childInstanceRegion
	params["ChildInstanceType"] = childInstanceType
	params["DestinationCidrBlock"] = cidr
	params["Action"] = "PublishRouteEntries"
	_, err := client.cbnRequest("PublishRouteEntries", params)
	if err != nil {
		return errors.Wrapf(err, `client.cbnRequest("PublishRouteEntries", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (client *SAliyunClient) WithdrawPublishedRouteEntries(cenId, childInstanceId, routeTableId, childInstanceRegion, childInstanceType, cidr string) error {
	params := map[string]string{}
	params["CenId"] = cenId
	params["ChildInstanceId"] = childInstanceId
	params["ChildInstanceRouteTableId"] = routeTableId
	params["ChildInstanceRegionId"] = childInstanceRegion
	params["ChildInstanceType"] = childInstanceType
	params["DestinationCidrBlock"] = cidr
	params["Action"] = "WithdrawPublishedRouteEntries"
	_, err := client.cbnRequest("WithdrawPublishedRouteEntries", params)
	if err != nil {
		return errors.Wrapf(err, `client.cbnRequest("WithdrawPublishedRouteEntries", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SCenRouteEntry) GetId() string {
	return self.RouteTableID + ":" + self.DestinationCidrBlock
}

func (self *SCenRouteEntry) GetName() string {
	return self.GetId()
}

func (self *SCenRouteEntry) GetGlobalId() string {
	return self.GetId()
}

func (self *SCenRouteEntry) GetStatus() string {
	if len(self.Conflicts.Conflict) > 0 {
		return api.ROUTE_ENTRY_STATUS_CONFLICT
	}
	return api.ROUTE_ENTRY_STATUS_AVAILIABLE
}

func (self *SCenRouteEntry) Refresh() error {
	return nil
}

func (self *SCenRouteEntry) IsEmulated() bool {
	return false
}

func (self *SCenRouteEntry) GetCidr() string {
	return self.DestinationCidrBlock
}

func (self *SCenRouteEntry) GetNextHopType() string {
	switch self.NextHopType {
	case "VPC":
		return api.NEXT_HOP_TYPE_VPC
	case "VBR":
		return api.NEXT_HOP_TYPE_VBR
	default:
		return ""
	}
}

func (self *SCenRouteEntry) GetNextHop() string {
	return self.NextHopInstanceID
}

func (self *SCenRouteEntry) GetNextHopRegion() string {
	return self.NextHopRegionID
}

func (self *SCenRouteEntry) GetEnabled() bool {
	if self.PublishStatus == "Published" {
		return true
	}
	return false
}

func (self *SCenRouteEntry) GetRouteTableId() string {
	return self.RouteTableID
}

func (self *SCenRouteEntry) GetInstanceId() string {
	return self.ChildInstance.ChildInstanceID
}

func (self *SCenRouteEntry) GetInstanceType() string {
	return self.ChildInstance.ChildInstanceType
}

func (self *SCenRouteEntry) GetInstanceRegionId() string {
	return self.ChildInstance.ChildInstanceRegionID
}
