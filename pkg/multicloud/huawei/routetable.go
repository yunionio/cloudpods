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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// date: 2019.07.15
// In Huawei cloud, there are only two routing tables in a vpc, which are
// self-defined routing tables and peer-to-peer routing tables.
// The routing in these two tables is different, one's NextHop is a IP address and
// the other one's NextHop address is a instance ID of peer-to-peer connection.
// The former has no id and it's Type is ROUTE_TYPR_IP, and the latter's Type is ROUTE_TYPE_PEER.

const (
	ROUTE_TYPR_IP   = "IP"
	ROUTE_TYPE_PEER = "peering"
)

type SRouteEntry struct {
	routeTable *SRouteTable

	ID          string // route ID
	Type        string // route type
	Destination string // route destination
	NextHop     string // route next hop (ip or id)
}

func (route *SRouteEntry) GetType() string {
	return route.Type
}

func (route *SRouteEntry) GetCidr() string {
	return route.Destination
}

func (route *SRouteEntry) GetNextHopType() string {
	// In Huawei Cloud, NextHopType is same with itself
	return route.GetType()
}

func (route *SRouteEntry) GetNextHop() string {
	return route.NextHop
}

// SRouteTable has no ID and Name because there is no id or name of route table in huawei cloud.
// And some method such as GetId and GetName of ICloudRouteTable has no practical meaning
type SRouteTable struct {
	region *SRegion
	vpc    *SVpc

	VpcId       string
	Description string
	Type        string
	Routes      []cloudprovider.ICloudRoute
}

func NewSRouteTable(vpc *SVpc, Type string) SRouteTable {
	return SRouteTable{
		region: vpc.region,
		vpc:    vpc,
		Type:   Type,
		VpcId:  vpc.GetId(),
	}

}

func (self *SRouteTable) GetId() string {
	return self.GetGlobalId()
}

func (self *SRouteTable) GetName() string {
	return ""
}

func (self *SRouteTable) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.GetVpcId(), self.GetType())
}

func (self *SRouteTable) GetStatus() string {
	return ""
}

func (self *SRouteTable) Refresh() error {
	return nil
}

func (self *SRouteTable) IsEmulated() bool {
	return false
}

func (self *SRouteTable) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRouteTable) GetDescription() string {
	return self.Description
}

func (self *SRouteTable) GetRegionId() string {
	return self.region.GetId()
}

func (self *SRouteTable) GetVpcId() string {
	return self.VpcId
}

func (self *SRouteTable) GetType() string {
	return self.Type
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	if self.Routes == nil {
		err := self.fetchRoutes()
		if err != nil {
			return nil, err
		}
	}
	return self.Routes, nil
}

// fetchRoutes fetch Routes
func (self *SRouteTable) fetchRoutes() error {
	if self.Type == ROUTE_TYPR_IP {
		return self.fetchRoutesForIP()
	}
	return self.fetchRoutesForPeer()
}

// fetchRoutesForIP fetch the Routes which Type is ROUTE_TYPR_IP through vpc's get api
func (self *SRouteTable) fetchRoutesForIP() error {
	ret, err := self.region.ecsClient.Vpcs.Get(self.GetVpcId(), map[string]string{})
	if err != nil {
		return errors.Wrap(err, "get vpc info error")
	}
	routeArray, err := ret.GetArray("routes")
	routes := make([]cloudprovider.ICloudRoute, 0, len(routeArray))
	for i := range routeArray {
		destination, err := routeArray[i].GetString("destination")
		if err != nil {
			return errors.Wrap(err, "get destination of route error")
		}
		nextHop, err := routeArray[i].GetString("nexthop")
		if err != nil {
			return errors.Wrap(err, "get nexthop of route error")
		}
		routes = append(routes, &SRouteEntry{
			routeTable:  self,
			ID:          "",
			Type:        ROUTE_TYPR_IP,
			Destination: destination,
			NextHop:     nextHop,
		})
	}
	self.Routes = routes
	return nil
}

// fetchRoutesForPeer fetch the routes which Type is ROUTE_TYPE_PEER through vpcRoute's list api
func (self *SRouteTable) fetchRoutesForPeer() error {
	retPeer, err := self.region.ecsClient.VpcRoutes.List(map[string]string{"vpc_id": self.GetVpcId()})
	if err != nil {
		return errors.Wrap(err, "get peer route error")
	}
	routesPeer := make([]cloudprovider.ICloudRoute, 0, retPeer.Total)
	for i := range retPeer.Data {
		route := retPeer.Data[i]
		id, err := route.GetString("id")
		if err != nil {
			return errors.Wrap(err, "get id of peer route error")
		}
		destination, err := route.GetString("destination")
		if err != nil {
			return errors.Wrap(err, "get destination of peer route error")
		}
		nextHop, err := route.GetString("nexthop")
		if err != nil {
			return errors.Wrap(err, "get nexthop of peer route error")
		}
		routesPeer = append(routesPeer, &SRouteEntry{
			routeTable:  self,
			ID:          id,
			Type:        ROUTE_TYPE_PEER,
			Destination: destination,
			NextHop:     nextHop,
		})
	}
	self.Routes = routesPeer
	return nil
}

// GetRouteTables return []SRouteTable of self
func (self *SVpc) getRouteTables() ([]SRouteTable, error) {
	// every Vpc has two route table in Huawei Cloud
	routeTableIp := NewSRouteTable(self, ROUTE_TYPR_IP)
	routeTablePeer := NewSRouteTable(self, ROUTE_TYPE_PEER)
	if err := routeTableIp.fetchRoutesForIP(); err != nil {
		return nil, errors.Wrap(err, `get route table whilc type is "ip" error`)
	}
	if err := routeTablePeer.fetchRoutesForPeer(); err != nil {
		return nil, errors.Wrap(err, `get route table whilc type is "peering" error`)
	}
	ret := make([]SRouteTable, 0, 2)
	if len(routeTableIp.Routes) != 0 {
		ret = append(ret, routeTableIp)
	}
	if len(routeTablePeer.Routes) != 0 {
		ret = append(ret, routeTablePeer)
	}
	return ret, nil
}

// GetRouteTables return []SRouteTable of vpc which id is vpcId if vpcId is no-nil,
// otherwise return []SRouteTable of all vpc in this SRegion
func (self *SRegion) GetRouteTables(vpcId string) ([]SRouteTable, error) {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, errors.Wrap(err, "Get Vpcs error")
	}
	if vpcId != "" {
		for i := range vpcs {
			if vpcs[i].GetId() == vpcId {
				vpcs = vpcs[i : i+1]
				break
			}
		}
	}
	ret := make([]SRouteTable, 0, 2*len(vpcs))
	for _, vpc := range vpcs {
		routetables, err := vpc.getRouteTables()
		if err != nil {
			return nil, errors.Wrapf(err, "get vpc's route tables whilch id is %s error", vpc.GetId())
		}
		ret = append(ret, routetables...)

	}
	return ret, nil
}
