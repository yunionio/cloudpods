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
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRouteEntry struct {
	multicloud.SResourceBase
	OpenStackTags
	Destination string `json:"destination"`
	Nexthop     string `json:"nexthop"`
}

func (route *SRouteEntry) GetId() string {
	return route.Destination + ":" + route.Nexthop
}
func (route *SRouteEntry) GetName() string {
	return ""
}
func (route *SRouteEntry) GetGlobalId() string {
	return route.GetId()
}

func (route *SRouteEntry) GetStatus() string {
	return ""
}

func (route *SRouteEntry) Refresh() error {
	return nil
}

func (route *SRouteEntry) IsEmulated() bool {
	return false
}

func (route *SRouteEntry) GetType() string {
	return api.ROUTE_ENTRY_TYPE_CUSTOM
}

func (route *SRouteEntry) GetCidr() string {
	return route.Destination
}

func (route *SRouteEntry) GetNextHopType() string {
	return route.Nexthop
}

func (route *SRouteEntry) GetNextHop() string {
	return route.Nexthop
}

type SRouteTable struct {
	multicloud.SResourceBase
	OpenStackTags
	vpc     *SVpc
	entries []SRouteEntry
	router  *SRouter
}

func (self *SRouteTable) GetDescription() string {
	return ""
}

func (self *SRouteTable) GetId() string {
	return self.GetGlobalId()
}

func (self *SRouteTable) GetGlobalId() string {
	return self.router.Id
}

func (self *SRouteTable) GetName() string {
	return self.router.Name
}

func (self *SRouteTable) GetRegionId() string {
	return self.vpc.region.GetId()
}

func (self *SRouteTable) GetType() cloudprovider.RouteTableType {
	return cloudprovider.RouteTableTypeSystem
}

func (self *SRouteTable) GetVpcId() string {
	return self.vpc.GetId()
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	ret := []cloudprovider.ICloudRoute{}
	for index := range self.entries {
		ret = append(ret, &self.entries[index])
	}
	return ret, nil
}

func (self *SRouteTable) GetStatus() string {
	return self.router.Status
}

func (self *SRouteTable) IsEmulated() bool {
	return false
}

func (self *SRouteTable) Refresh() error {
	return nil
}

func (self *SRouteTable) GetAssociations() []cloudprovider.RouteTableAssociation {
	result := []cloudprovider.RouteTableAssociation{}
	return result
}

func (self *SRouteTable) CreateRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRouteTable) UpdateRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRouteTable) RemoveRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}
