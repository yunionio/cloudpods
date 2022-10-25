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

package hcs

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

var hoptypes = map[string]string{
	api.NEXT_HOP_TYPE_VPCPEERING:      "peering",
	api.NEXT_HOP_TYPE_INSTANCE:        "ecs",
	api.NEXT_HOP_TYPE_NAT:             "nat",
	api.NEXT_HOP_TYPE_HAVIP:           "vip",
	api.NEXT_HOP_TYPE_NETWORK:         "eni",
	api.NEXT_HOP_TYPE_INTERVPCNETWORK: "cc",
}

type SRoute struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags

	Type        string `json:"type"`
	Destination string `json:"destination"`
	Nexthop     string `json:"nexthop"`
	Description string `json:"description,omitempty"`
}

func (self *SRoute) GetCidr() string {
	return self.Destination
}

func (self *SRoute) GetGlobalId() string {
	return self.GetId()
}

func (self *SRoute) GetId() string {
	return fmt.Sprintf("%s:%s:%s", self.Type, self.Nexthop, self.Destination)
}

func (self *SRoute) GetName() string {
	return ""
}

func (route *SRoute) GetStatus() string {
	return api.ROUTE_ENTRY_STATUS_AVAILIABLE
}

func (route *SRoute) GetType() string {
	if route.Type != "local" {
		return api.ROUTE_ENTRY_TYPE_CUSTOM
	}
	return api.ROUTE_ENTRY_TYPE_SYSTEM
}

func (self *SRoute) GetNextHopType() string {
	for k, v := range hoptypes {
		if v == self.Type {
			return k
		}
	}
	return self.Type
}

type Subnet struct {
	Id string `json:"id"`
}

type SRouteTable struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	vpc *SVpc

	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Routes   []SRoute `json:"routes"`
	Subnets  []Subnet `json:"subnets"`
	VpcId    string   `json:"vpc_id"`
	Default  bool     `json:"default"`
	TenantId string   `json:"tenant_id"`
}

func (self *SRouteTable) GetDescription() string {
	return ""
}

func (self *SRouteTable) GetGlobalId() string {
	return self.Id
}

func (self *SRouteTable) GetId() string {
	return self.Id
}

func (self *SRouteTable) GetName() string {
	return self.Name
}

func (self *SRouteTable) GetRegionId() string {
	return self.vpc.region.GetId()
}

func (self *SRouteTable) GetVpcId() string {
	return self.VpcId
}

func (self *SRouteTable) GetType() cloudprovider.RouteTableType {
	return cloudprovider.RouteTableTypeSystem
}

func (self *SRouteTable) GetStatus() string {
	return api.ROUTE_TABLE_AVAILABLE
}

func (self *SRouteTable) Refresh() error {
	rtb, err := self.vpc.region.GetRouteTable(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, rtb)
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	if len(self.Routes) == 0 {
		err := self.Refresh()
		if err != nil {
			return nil, err
		}
	}
	ret := []cloudprovider.ICloudRoute{}
	for i := range self.Routes {
		ret = append(ret, &self.Routes[i])
	}
	return ret, nil
}

func (self *SRoute) GetNextHop() string {
	return self.Nexthop
}

func (self *SRegion) GetRouteTables(vpcId string) ([]SRouteTable, error) {
	params := url.Values{}
	if len(vpcId) > 0 {
		params.Set("vpc_id", vpcId)
	}
	rtbs := []SRouteTable{}
	return rtbs, self.list("vpc", "v2.0", "vpc/routes", params, &rtbs)
}

func (self *SRegion) GetRouteTable(id string) (*SRouteTable, error) {
	tb := &SRouteTable{}
	return tb, self.get("vpc", "v2.0", "vpc/routes/"+id, tb)
}

func (self *SRouteTable) CreateRoute(route cloudprovider.RouteSet) error {
	routeType, ok := hoptypes[route.NextHopType]
	if !ok {
		return errors.Wrapf(cloudprovider.ErrNotSupported, route.NextHopType)
	}
	params := map[string]interface{}{
		"routes": map[string]interface{}{
			"add": []map[string]interface{}{
				{
					"type":        routeType,
					"destination": route.Destination,
					"nexthop":     route.NextHop,
				},
			},
		},
	}
	return self.vpc.region.create("vpc", "v2.0", "vpc/routes", params, nil)
}

func (self *SRouteTable) RemoveRoute(route cloudprovider.RouteSet) error {
	if len(route.RouteId) > 0 {
		return self.vpc.region.delete("vpc", "v2.0", "vpc/routes/"+route.RouteId)
	}
	return fmt.Errorf("missing route id")
}

func (self *SRouteTable) UpdateRoute(route cloudprovider.RouteSet) error {
	err := self.RemoveRoute(route)
	if err != nil {
		return errors.Wrap(err, "self.RemoveRoute(route)")
	}
	err = self.CreateRoute(route)
	if err != nil {
		return errors.Wrap(err, "self.CreateRoute(route)")
	}
	return nil
}

func (self *SRouteTable) GetAssociations() []cloudprovider.RouteTableAssociation {
	return []cloudprovider.RouteTableAssociation{}
}
