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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRouteSet struct {
	multicloud.SResourceBase
	multicloud.QcloudTags
	RouteID                  int    `json:"RouteId"`
	RouteItemID              string `json:"RouteItemId"`
	DestinationIpv6CidrBlock string `json:"DestinationIpv6CidrBlock,omitempty"`
	GatewayType              string `json:"GatewayType"`
	GatewayID                string `json:"GatewayId"`
	RouteDescription         string `json:"RouteDescription"`
	DestinationCidrBlock     string `json:"DestinationCidrBlock,omitempty"`
	RouteType                string `json:"RouteType"`
}
type SRouteTableSet struct {
	multicloud.SResourceBase
	multicloud.QcloudTags
	vpc            *SVpc
	VpcID          string                     `json:"VpcId"`
	RouteTableID   string                     `json:"RouteTableId"`
	RouteTableName string                     `json:"RouteTableName"`
	AssociationSet []RouteTableAssociationSet `json:"AssociationSet"`
	RouteSet       []SRouteSet                `json:"RouteSet"`
	Main           bool                       `json:"Main"`
	CreatedTime    string                     `json:"CreatedTime"`
}

type RouteTableAssociationSet struct {
	SubnetId     string
	RouteTableId string
}

func (self *SRegion) DescribeRouteTables(vpcId string, routetables []string, offset int, limit int) ([]SRouteTableSet, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	for i := range routetables {
		params[fmt.Sprintf("RouteTableIds.%d", i)] = routetables[i]
	}

	if len(vpcId) > 0 {
		params["Filters.0.Name"] = "vpc-id"
		params["Filters.0.Values.0"] = vpcId
	}
	body, err := self.vpcRequest("DescribeRouteTables", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, ` self.vpcRequest("DescribeRouteTables", %s)`, jsonutils.Marshal(params))
	}
	routeTables := make([]SRouteTableSet, 0)
	err = body.Unmarshal(&routeTables, "RouteTableSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "body.Unmarshal(RouteTableSet)[%s]", body.String())
	}
	total, _ := body.Float("TotalCount")
	return routeTables, int(total), nil
}

func (self *SRegion) GetAllRouteTables(vpcId string, routetables []string) ([]SRouteTableSet, error) {
	routeTables := make([]SRouteTableSet, 0)
	for {
		part, total, err := self.DescribeRouteTables(vpcId, routetables, len(routeTables), 50)
		if err != nil {
			return nil, errors.Wrap(err, "self.DescribeRouteTables(vpcId, routetables, offset, limit)")
		}
		routeTables = append(routeTables, part...)
		if len(routeTables) >= total {
			break
		}
	}
	return routeTables, nil
}

func (self *SRouteTableSet) GetId() string {
	return self.RouteTableID
}

func (self *SRouteTableSet) GetName() string {
	return self.RouteTableName
}

func (self *SRouteTableSet) GetGlobalId() string {
	return self.RouteTableID
}

func (self *SRouteTableSet) GetStatus() string {
	return api.ROUTE_TABLE_AVAILABLE
}

func (self *SRouteTableSet) Refresh() error {
	return nil
}

func (self *SRouteTableSet) IsEmulated() bool {
	return false
}

func (self *SRouteTableSet) GetAssociations() []cloudprovider.RouteTableAssociation {
	result := []cloudprovider.RouteTableAssociation{}
	for i := range self.AssociationSet {
		association := cloudprovider.RouteTableAssociation{
			AssociationType:      cloudprovider.RouteTableAssociaToSubnet,
			AssociatedResourceId: self.AssociationSet[i].SubnetId,
		}
		result = append(result, association)
	}
	return result
}

func (self *SRouteTableSet) GetDescription() string {
	return ""
}

func (self *SRouteTableSet) GetRegionId() string {
	return self.vpc.GetRegion().GetId()
}

func (self *SRouteTableSet) GetVpcId() string {
	return self.vpc.GetId()
}

func (self *SRouteTableSet) GetType() cloudprovider.RouteTableType {
	if self.Main {
		return cloudprovider.RouteTableTypeSystem
	}
	return cloudprovider.RouteTableTypeCustom
}

func (self *SRouteTableSet) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	result := []cloudprovider.ICloudRoute{}
	for i := range self.RouteSet {
		result = append(result, &self.RouteSet[i])
	}
	return result, nil
}

func (self *SRouteTableSet) CreateRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRouteTableSet) UpdateRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRouteTableSet) RemoveRoute(route cloudprovider.RouteSet) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRouteSet) GetId() string {
	return self.RouteItemID
}

func (self *SRouteSet) GetName() string {
	return ""
}

func (self *SRouteSet) GetGlobalId() string {
	return self.RouteItemID
}

func (self *SRouteSet) GetStatus() string {
	return api.ROUTE_ENTRY_STATUS_AVAILIABLE
}

func (self *SRouteSet) Refresh() error {
	return nil
}

func (self *SRouteSet) IsEmulated() bool {
	return false
}

func (self *SRouteSet) GetType() string {
	switch self.RouteType {
	case "USER":
		return api.ROUTE_ENTRY_TYPE_CUSTOM
	case "NETD":
		return api.ROUTE_ENTRY_TYPE_SYSTEM
	case "CCN":
		return api.ROUTE_ENTRY_TYPE_PROPAGATE
	default:
		return api.ROUTE_ENTRY_TYPE_SYSTEM
	}
}

func (self *SRouteSet) GetCidr() string {
	return self.DestinationCidrBlock
}

func (self *SRouteSet) GetNextHopType() string {
	switch self.GatewayType {
	case "CVM":
		return api.NEXT_HOP_TYPE_INSTANCE
	case "VPN":
		return api.NEXT_HOP_TYPE_VPN
	case "DIRECTCONNECT":
		return api.NEXT_HOP_TYPE_DIRECTCONNECTION
	case "PEERCONNECTION":
		return api.NEXT_HOP_TYPE_VPCPEERING
	case "SSLVPN":
		return api.NEXT_HOP_TYPE_VPN
	case "NAT":
		return api.NEXT_HOP_TYPE_NAT
	case "NORMAL_CVM":
		return api.NEXT_HOP_TYPE_INSTANCE
	case "EIP":
		return api.NEXT_HOP_TYPE_EIP
	case "CCN":
		return api.NEXT_HOP_TYPE_INTERVPCNETWORK
	default:
		return ""
	}
}

func (self *SRouteSet) GetNextHop() string {
	return self.GatewayID
}
