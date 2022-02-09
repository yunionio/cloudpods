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

package qcloud

import (
	"fmt"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SCcnRouteSet struct {
	multicloud.SResourceBase
	multicloud.QcloudTags

	RouteID              string `json:"RouteId"`
	DestinationCidrBlock string `json:"DestinationCidrBlock"`
	InstanceType         string `json:"InstanceType"`
	InstanceID           string `json:"InstanceId"`
	InstanceName         string `json:"InstanceName"`
	InstanceRegion       string `json:"InstanceRegion"`
	InstanceUin          string `json:"InstanceUin"`
	UpdateTime           string `json:"UpdateTime"`
	Enabled              bool   `json:"Enabled"`
	ExtraState           string `json:"ExtraState"`
}
type SCcnRouteSets struct {
	RouteSet   []SCcnRouteSet `json:"RouteSet"`
	TotalCount int            `json:"TotalCount"`
	RequestID  string         `json:"RequestId"`
}

func (self *SRegion) DescribeCcnRoutes(ccnId string, offset int, limit int) ([]SCcnRouteSet, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	params["Limit"] = strconv.Itoa(limit)
	params["CcnId"] = ccnId
	resp, err := self.vpcRequest("DescribeCcnRoutes", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, `self.vpcRequest("DescribeCcnRoutes", %s)`, jsonutils.Marshal(params).String())
	}
	routes := []SCcnRouteSet{}
	err = resp.Unmarshal(&routes, "RouteSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, `(%s).Unmarshal(&routes,"RouteSet")`, jsonutils.Marshal(resp).String())
	}
	total, _ := resp.Float("TotalCount")
	return routes, int(total), nil
}

func (self *SRegion) GetAllCcnRouteSets(ccnId string) ([]SCcnRouteSet, error) {
	routes := []SCcnRouteSet{}
	for {
		part, total, err := self.DescribeCcnRoutes(ccnId, len(routes), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "self.DescribeCcns(nil, %d, 50)", len(routes))
		}
		routes = append(routes, part...)
		if len(routes) >= total {
			break
		}
	}
	return routes, nil
}

func (self *SRegion) EnableCcnRoutes(ccnId string, routeIds []string) error {
	params := map[string]string{}
	params["CcnId"] = ccnId
	for i := range routeIds {
		params[fmt.Sprintf("RouteIds.%d", i)] = routeIds[i]
	}
	_, err := self.vpcRequest("EnableCcnRoutes", params)
	if err != nil {
		return errors.Wrapf(err, `self.vpcRequest("EnableCcnRoutes", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) DisableCcnRoutes(ccnId string, routeIds []string) error {
	params := map[string]string{}
	params["CcnId"] = ccnId
	for i := range routeIds {
		params[fmt.Sprintf("RouteIds.%d", i)] = routeIds[i]
	}
	_, err := self.vpcRequest("DisableCcnRoutes", params)
	if err != nil {
		return errors.Wrapf(err, `self.vpcRequest("DisableCcnRoutes", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SCcnRouteSet) GetId() string {
	return self.RouteID
}

func (self *SCcnRouteSet) GetName() string {
	return ""
}

func (self *SCcnRouteSet) GetGlobalId() string {
	return self.RouteID
}

func (self *SCcnRouteSet) GetStatus() string {
	switch self.ExtraState {
	case "Disable":
		return api.ROUTE_ENTRY_STATUS_DISABLED
	case "Running":
		return api.ROUTE_ENTRY_STATUS_AVAILIABLE
	default:
		return api.ROUTE_ENTRY_STATUS_UNKNOWN
	}
}

func (self *SCcnRouteSet) Refresh() error {
	return nil
}

func (self *SCcnRouteSet) IsEmulated() bool {
	return false
}

func (self *SCcnRouteSet) GetInstanceId() string {
	return self.InstanceID
}

func (self *SCcnRouteSet) GetInstanceType() string {
	switch self.InstanceType {
	case "VPC":
		return api.NEXT_HOP_TYPE_VPC
	case "DIRECTCONNECT":
		return api.NEXT_HOP_TYPE_VBR
	default:
		return ""
	}
}

func (self *SCcnRouteSet) GetInstanceRegionId() string {
	return self.InstanceRegion
}

func (self *SCcnRouteSet) GetEnabled() bool {
	return self.Enabled
}

func (self *SCcnRouteSet) GetCidr() string {
	return self.DestinationCidrBlock
}
