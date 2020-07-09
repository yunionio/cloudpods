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
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRouteEntry struct {
	Destination string `json:"destination"`
	Nexthop     string `json:"nexthop"`
}

func (route *SRouteEntry) GetType() string {
	return api.ROUTE_ENTRY_TYPE_CUSTOM
}

func (route *SRouteEntry) GetCidr() string {
	return route.Destination
}

func (route *SRouteEntry) GetNextHopType() string {
	return ""
}

func (route *SRouteEntry) GetNextHop() string {
	return route.Nexthop
}

type SRouteTable struct {
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
	return self.router.ID
}

func (self *SRouteTable) GetName() string {
	return self.router.Name
}

func (self *SRouteTable) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRouteTable) GetRegionId() string {
	return self.vpc.region.GetId()
}

func (self *SRouteTable) GetType() string {
	return ""
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
