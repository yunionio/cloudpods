// Copyright 2023 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package volcengine

import (
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SRouteEntry struct {
	multicloud.SResourceBase
	VolcEngineTags
	routeTable *SRouteTable

	Description          string
	DestinationCidrBlock string
	RouteEntryId         string
	RouteEntryName       string
	RouteTableId         string
	Status               string
	Type                 string
	VpcId                string
	NextHopId            string
	NextHopName          string
	NextHopType          string
}

type SRouteEntrys []*SRouteEntry

type SubnetIds []string

type SRouteTable struct {
	multicloud.SResourceBase
	VolcEngineTags
	region *SRegion
	vpc    *SVpc
	routes []cloudprovider.ICloudRoute

	Description    string
	RouteTableId   string
	RouteTableName string
	RouteTableType string
	VpcId          string
	VpcName        string
	CreationTime   time.Time
	UpdateTime     time.Time
	AccountId      string
	ProjectName    string
	SubnetIds      SubnetIds
	RouteEntrys    SRouteEntrys
}

func (route *SRouteEntry) GetId() string {
	return fmt.Sprintf("%s-%s-%s", route.RouteTableId, route.DestinationCidrBlock, route.NextHopType)
}

func (route *SRouteEntry) GetName() string {
	return route.RouteEntryName
}

func (route *SRouteEntry) GetGlobalId() string {
	return route.GetId()
}

func (route *SRouteEntry) GetStatus() string {
	return api.ROUTE_ENTRY_STATUS_AVAILIABLE
}

func (route *SRouteEntry) Refresh() error {
	return nil
}

func (route *SRouteEntry) GetType() string {
	return route.Type
}

func (route *SRouteEntry) GetCidr() string {
	return route.DestinationCidrBlock
}

func (route *SRouteEntry) GetNextHopType() string {
	switch route.NextHopType {
	case "Instance":
		return api.NEXT_HOP_TYPE_INSTANCE
	case "HaVip":
		return api.NEXT_HOP_TYPE_HAVIP
	case "VpnGW":
		return api.NEXT_HOP_TYPE_VPN
	case "NatGW":
		return api.NEXT_HOP_TYPE_NAT
	case "NetworkInterface":
		return api.NEXT_HOP_TYPE_NETWORK
	case "IPv6GW":
		return api.NEXT_HOP_TYPE_IPV6
	case "TransitRouter":
		return api.NEXT_HOP_TYPE_ROUTER
	default:
		return ""
	}
}

func (route *SRouteEntry) GetNextHop() string {
	return route.NextHopId
}

func (table *SRouteTable) GetDescription() string {
	return table.Description
}

func (table *SRouteTable) GetId() string {
	return table.RouteTableId
}

func (table *SRouteTable) GetGlobalId() string {
	return table.RouteTableId
}

func (table *SRouteTable) GetName() string {
	return table.RouteTableName
}

func (table *SRouteTable) GetRegionId() string {
	return table.region.RegionId
}

func (table *SRouteTable) GetType() cloudprovider.RouteTableType {
	switch table.RouteTableType {
	case "System":
		return cloudprovider.RouteTableTypeSystem
	case "Custom":
		return cloudprovider.RouteTableTypeCustom
	default:
		return cloudprovider.RouteTableTypeSystem
	}
}

func (table *SRouteTable) GetVpcId() string {
	return table.VpcId
}

func (table *SRouteTable) GetStatus() string {
	return api.ROUTE_TABLE_AVAILABLE
}

func (table *SRouteTable) Refresh() error {
	return nil
}

func (routeTable *SRouteTable) IsSystem() bool {
	return strings.ToLower(routeTable.RouteTableType) == "system"
}

func (table *SRouteTable) RemoteGetRoutes(pageNumber int, pageSize int) ([]*SRouteEntry, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["RouteTableId"] = table.RouteTableId
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)

	body, err := table.region.vpcRequest("DescribeRouteEntryList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "RemoteGetRoutes fail")
	}

	entries := SRouteEntrys{}
	err = body.Unmarshal(&entries, "RouteEntries")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal routeEntrys fail")
	}
	total, _ := body.Int("TotalCount")
	return entries, int(total), nil
}

func (table *SRouteTable) fetchRoutes() error {
	routes := []*SRouteEntry{}
	pageNumber := 1
	for {
		parts, total, err := table.RemoteGetRoutes(pageNumber, 50)
		if err != nil {
			return err
		}
		routes = append(routes, parts...)
		if len(routes) >= total {
			break
		}
		pageNumber += 1
	}
	table.routes = make([]cloudprovider.ICloudRoute, len(routes))
	for i := 0; i < len(routes); i++ {
		routes[i].routeTable = table
		table.routes[i] = routes[i]
	}
	return nil
}

func (table *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	if table.routes == nil {
		err := table.fetchRoutes()
		if err != nil {
			return nil, err
		}
	}
	return table.routes, nil
}

func (table *SRouteTable) GetAssociations() []cloudprovider.RouteTableAssociation {
	result := []cloudprovider.RouteTableAssociation{}
	for i := range table.SubnetIds {
		association := cloudprovider.RouteTableAssociation{
			AssociationId:        table.RouteTableId + ":" + table.SubnetIds[i],
			AssociationType:      cloudprovider.RouteTableAssociaToSubnet,
			AssociatedResourceId: table.SubnetIds[i],
		}
		result = append(result, association)
	}
	return result
}

func (table *SRouteTable) CreateRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (table *SRouteTable) UpdateRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (table *SRouteTable) RemoveRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (vpc *SVpc) RemoteGetRouteTableList(pageNumber int, pageSize int) ([]*SRouteTable, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["VpcId"] = vpc.VpcId
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)

	body, err := vpc.region.vpcRequest("DescribeRouteTableList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "RemoteGetRouteTableList fail")
	}

	routeTables := make([]*SRouteTable, 0)
	err = body.Unmarshal(&routeTables, "RouterTableList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal routeTables fail")
	}
	for _, routeTable := range routeTables {
		routeTable.region = vpc.region
	}
	total, _ := body.Int("TotalCount")
	return routeTables, int(total), nil
}

func (region *SRegion) AssociateRouteTable(rtableId string, SubnetId string) error {
	params := make(map[string]string)
	params["RouteTableId"] = rtableId
	params["SubnetId"] = SubnetId
	_, err := region.vpcRequest("AssociateRouteTable", params)
	return err
}

func (region *SRegion) UnassociateRouteTable(rtableId string, SubnetId string) error {
	params := make(map[string]string)
	params["RouteTableId"] = rtableId
	params["SubnetId"] = SubnetId
	_, err := region.vpcRequest("UnassociateRouteTable", params)
	return err
}
