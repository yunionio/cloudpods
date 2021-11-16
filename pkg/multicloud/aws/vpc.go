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

package aws

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SUserCIDRs struct {
	UserCidr []string
}

type SVpc struct {
	multicloud.SVpc
	multicloud.AwsTags

	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	CidrBlock               string `xml:"cidrBlock"`
	CidrBlockAssociationSet []struct {
		AssociationId  string `xml:"associationId"`
		CidrBlock      string `xml:"cidrBlock"`
		CidrBlockState struct {
			State         string `xml:"state"`
			StatusMessage string `xml:"statusMessage"`
		}
	} `xml:"cidrBlockAssociationSet>item"`
	DhcpOptionsId               string `xml:"dhcpOptionsId"`
	InstanceTenancy             string `xml:"instanceTenancy"`
	Ipv6CidrBlockAssociationSet []struct {
		AssociationId      string `xml:"associationId"`
		Ipv6CidrBlock      string `xml:"ipv6CidrBlock"`
		Ipv6CidrBlockState struct {
			State         string `xml:"state"`
			StatusMessage string `xml:"statusMessage"`
		}
		Ipv6Pool           string `xml:"ipv6Pool"`
		NetworkBorderGroup string `xml:"networkBorderGroup"`
	}
	IsDefault bool   `xml:"isDefault"`
	OwnerId   string `xml:"ownerId"`
	State     string `xml:"state"`
	VpcId     string `xml:"vpcId"`
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) GetId() string {
	return self.VpcId
}

func (self *SVpc) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.VpcId
}

func (self *SVpc) GetGlobalId() string {
	return self.VpcId
}

// pending | available
func (self *SVpc) GetStatus() string {
	// 目前不支持专用主机
	if self.InstanceTenancy == "dedicated" {
		return api.VPC_STATUS_UNAVAILABLE
	}
	switch self.State {
	case "available":
		return api.VPC_STATUS_AVAILABLE
	case "pending":
		return api.VPC_STATUS_PENDING
	}
	return self.State
}

func (self *SVpc) Refresh() error {
	vpc, err := self.region.GetVpc(self.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vpc)
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	cidrs := []string{self.CidrBlock}
	for _, block := range self.CidrBlockAssociationSet {
		if block.CidrBlockState.State == "associated" {
			cidrs = append(cidrs, block.CidrBlock)
		}
	}
	return strings.Join(cidrs, ",")
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, errors.Wrap(err, "fetchNetworks")
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.region.GetSecurityGroups(self.VpcId, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroups")
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self.region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	tables, err := self.region.GetRouteTables(self.GetId(), "", "", nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "GetRouteTables")
	}
	itables := make([]cloudprovider.ICloudRouteTable, len(tables))
	for i := range tables {
		tables[i].vpc = self
		itables[i] = &tables[i]
	}
	return itables, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	routeTable, err := self.region.GetRouteTable(routeTableId)
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetRouteTable(routeTableId:%s)", routeTableId)
	}
	routeTable.vpc = self
	return routeTable, nil
}

/*
Deletes the specified VPC. You must detach or delete all gateways and resources that are associated with
the VPC before you can delete it. For example, you must terminate all instances running in the VPC,
delete all security groups associated with the VPC (except the default one),
delete all route tables associated with the VPC (except the default one), and so on.
*/
func (self *SVpc) Delete() error {
	err := self.DeleteInternetGateways()
	if err != nil {
		return errors.Wrap(err, "DeleteInternetGateways")
	}

	// 删除路由表. todo: 3.7版本路由表开放之后，需要同步状态到平台
	rts, err := self.GetIRouteTables()
	if err != nil {
		return errors.Wrap(err, "GetIRouteTables")
	}

	for i := range rts {
		// 主路由表不允许删除
		rt := rts[i].(*SRouteTable)
		if len(rt.AssociationSet) > 0 {
			if rt.AssociationSet[0].Main {
				log.Debugf("Delete.RouteTable skipped main route table %s(%s)", rt.GetName(), rt.GetId())
				continue
			}
		}

		err = self.region.DeleteRouteTable(rts[i].GetId())
		if err != nil {
			return errors.Wrap(err, "DeleteRouteTable")
		}
	}

	return self.region.DeleteVpc(self.VpcId)
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, errors.Wrap(err, "fetchNetworks")
		}
	}
	for i := 0; i < len(self.iwires); i += 1 {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetIWireById")
}

func (self *SVpc) getWireByZoneId(zoneId string) *SWire {
	for i := 0; i < len(self.iwires); i += 1 {
		wire := self.iwires[i].(*SWire)
		if wire.zone.ZoneId == zoneId {
			return wire
		}
	}

	zone, err := self.region.getZoneById(zoneId)
	if err != nil {
		return nil
	}
	return &SWire{
		zone: zone,
		vpc:  self,
	}
}

func (self *SVpc) fetchNetworks() error {
	networks, err := self.region.GetNetwroks(nil, self.VpcId)
	if err != nil {
		return errors.Wrapf(err, "GetNetwroks(%s)", self.VpcId)
	}

	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByZoneId(networks[i].AvailabilityZone)
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
}

func (self *SVpc) revokeSecurityGroup(secgroupId string, instanceId string, keep bool) error {
	return self.region.revokeSecurityGroup(secgroupId, instanceId, keep)
}

func (self *SVpc) assignSecurityGroup(secgroupId string, instanceId string) error {
	return self.region.assignSecurityGroup(secgroupId, instanceId)
}

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	connections, err := self.region.DescribeVpcPeeringConnections("", "", self.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeVpcPeeringConnections")
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range connections {
		connections[i].vpc = self
		ret = append(ret, &connections[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	connections, err := self.region.DescribeVpcPeeringConnections("", self.VpcId, "")
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeVpcPeeringConnections")
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range connections {
		connections[i].vpc = self
		ret = append(ret, &connections[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	ret, err := self.region.GetVpcPeeringConnection(id)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeVpcPeeringConnections")
	}
	ret.vpc = self
	return ret, nil
}

func (self *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return self.createSVpcPeeringConnection(opts)
}

func (self *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	return self.acceptSVpcPeeringConnection(id)
}

func (self *SVpc) GetAuthorityOwnerId() string {
	identity, err := self.region.client.GetCallerIdentity()
	if err != nil {
		log.Errorf(err.Error() + "self.region.client.GetCallerIdentity()")
		return ""
	}
	return identity.Account
}

func (self *SVpc) createSVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeeringConnection, error) {
	peer, err := self.region.CreateVpcPeeringConnection(self.VpcId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, " self.region.CreateVpcPeeringConnection(%s,%s)", self.VpcId, jsonutils.Marshal(opts).String())
	}
	peer.vpc = self
	err = cloudprovider.WaitMultiStatus(peer, []string{api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT,
		api.VPC_PEERING_CONNECTION_STATUS_ACTIVE,
		api.VPC_PEERING_CONNECTION_STATUS_DELETING}, 5*time.Second, 60*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if peer.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_DELETING {
		return nil, errors.Wrapf(cloudprovider.ErrInvalidStatus, peer.GetStatus())
	}
	return peer, nil
}

func (self *SVpc) acceptSVpcPeeringConnection(id string) error {
	peer, err := self.region.GetVpcPeeringConnection(id)
	if err != nil {
		return errors.Wrapf(err, "GetVpcPeeringConnection(%s)", id)
	}
	//	其他region 创建的连接请求,有短暂的provisioning状态
	err = cloudprovider.WaitMultiStatus(peer, []string{api.VPC_PEERING_CONNECTION_STATUS_ACTIVE,
		api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT,
		api.VPC_PEERING_CONNECTION_STATUS_DELETING}, 5*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if peer.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_DELETING {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, peer.GetStatus())
	}

	if peer.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT {
		_, err := self.region.AcceptVpcPeeringConnection(id)
		if err != nil {
			return errors.Wrapf(err, "self.region.AcceptVpcPeeringConnection(%s)", id)
		}
	}
	err = cloudprovider.WaitMultiStatus(peer, []string{api.VPC_PEERING_CONNECTION_STATUS_ACTIVE,
		api.VPC_PEERING_CONNECTION_STATUS_DELETING}, 5*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if peer.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_DELETING {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, peer.GetStatus())
	}
	return nil
}

func (self *SVpc) IsSupportSetExternalAccess() bool {
	return true
}

func (self *SVpc) GetExternalAccessMode() string {
	igws, err := self.region.GetInternetGateways(self.GetId())
	if err != nil {
		log.Errorf("GetExternalAccessMode.GetInternetGateways %s", err)
	}

	if len(igws) > 0 {
		return api.VPC_EXTERNAL_ACCESS_MODE_EIP
	}

	return api.VPC_EXTERNAL_ACCESS_MODE_NONE
}

func (self *SVpc) AttachInternetGateway(igwId string) error {
	params := map[string]string{
		"InternetGatewayId": igwId,
		"VpcId":             self.VpcId,
	}
	err := self.region.ec2Request("AttachInternetGateway", params, nil)
	if err != nil {
		return errors.Wrapf(err, "AttachInternetGateway")
	}

	return self.AddDefaultInternetGatewayRoute(igwId)
}

func (self *SVpc) AddDefaultInternetGatewayRoute(igwId string) error {
	rt, err := self.GetMainRouteTable()
	if err != nil {
		return errors.Wrap(err, "GetMainRouteTable")
	}

	defaultRoute := cloudprovider.RouteSet{}
	defaultRoute.NextHop = igwId
	defaultRoute.Destination = "0.0.0.0/0"
	err = rt.CreateRoute(defaultRoute)
	if err != nil {
		return errors.Wrap(err, "CreateRoute")
	}

	return nil
}

func (self *SVpc) GetMainRouteTable() (*SRouteTable, error) {
	tables, err := self.region.GetRouteTables(self.GetId(), "", "", nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "GetRouteTables")
	}
	for i := range tables {
		tables[i].vpc = self
		tables[i].region = self.region
		return &tables[i], nil
	}
	return nil, errors.Wrap(cloudprovider.ErrNotSupported, "GetMainRouteTable")
}

func (self *SVpc) DetachInternetGateways() error {
	igws, err := self.region.GetInternetGateways(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetInternetGateways")
	}

	if len(igws) > 0 {
		for i := range igws {
			err = self.DetachInternetGateway(igws[i].GetId())
			if err != nil {
				return errors.Wrap(err, "DetachInternetGateway")
			}
		}
	}

	return nil
}

func (self *SVpc) DetachInternetGateway(igwId string) error {
	params := map[string]string{
		"InternetGatewayId": igwId,
		"VpcId":             self.VpcId,
	}
	return self.region.ec2Request("DetachInternetGateway", params, nil)
}

func (self *SVpc) DeleteInternetGateway(igwId string) error {
	params := map[string]string{
		"InternetGatewayId": igwId,
	}
	return self.region.ec2Request("DeleteInternetGateway", params, nil)
}

func (self *SVpc) DeleteInternetGateways() error {
	igws, err := self.region.GetInternetGateways(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetInternetGateways")
	}

	if len(igws) > 0 {
		for i := range igws {
			err = self.DetachInternetGateway(igws[i].GetId())
			if err != nil {
				return errors.Wrap(err, "DetachInternetGateway")
			}

			err = self.DeleteInternetGateway(igws[i].GetId())
			if err != nil {
				return errors.Wrap(err, "DeleteInternetGateway")
			}
		}
	}

	return nil
}

func (self *SRegion) DescribeVpcs(ids []string, nextToken string) ([]SVpc, string, error) {
	params := map[string]string{}
	for i, id := range ids {
		params[fmt.Sprintf("VpcId.%d", i+1)] = id
	}
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}
	ret := struct {
		Vpcs      []SVpc `xml:"vpcSet>item"`
		NextToken string `xml:"nextToken"`
	}{}
	return ret.Vpcs, ret.NextToken, self.ec2Request("DescribeVpcs", params, &ret)
}

func (self *SRegion) GetVpcs(ids []string) ([]SVpc, error) {
	part, nextToken, err := self.DescribeVpcs(ids, "")
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeVpcs")
	}
	vpcs := part
	for len(nextToken) > 0 && len(part) > 0 {
		part, nextToken, err = self.DescribeVpcs(ids, nextToken)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeVpcs")
		}
		vpcs = append(vpcs, part...)
	}
	return vpcs, nil
}

func (self *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	vpcs, err := self.GetVpcs([]string{vpcId})
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcs")
	}
	for i := range vpcs {
		if vpcs[i].VpcId == vpcId {
			vpcs[i].region = self
			return &vpcs[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "getVpc")
}

func (self *SRegion) revokeSecurityGroup(secgroupId, instanceId string, keep bool) error {
	// todo : keep ? 直接使用assignSecurityGroup 即可？
	return nil
}

func (self *SRegion) assignSecurityGroup(secgroupId, instanceId string) error {
	return self.assignSecurityGroups([]*string{&secgroupId}, instanceId)
}

func (self *SRegion) assignSecurityGroups(secgroupIds []*string, instanceId string) error {
	/*
		instance, err := self.GetInstance(instanceId)
		if err != nil {
			return errors.Wrap(err, "GetInstance")
		}

		ec2Client, err := self.getEc2Client()
		if err != nil {
			return errors.Wrap(err, "getEc2Client")
		}

			for _, eth := range instance.NetworkInterfaces.NetworkInterface {
				params := &ec2.ModifyNetworkInterfaceAttributeInput{}
				params.SetNetworkInterfaceId(eth.NetworkInterfaceId)
				params.SetGroups(secgroupIds)

				_, err := ec2Client.ModifyNetworkInterfaceAttribute(params)
				if err != nil {
					return err
				}
			}
	*/

	return nil
}

func (self *SRegion) DeleteSecurityGroup(secGrpId string) error {
	params := map[string]string{
		"GroupId": secGrpId,
	}
	return self.ec2Request("DeleteSecurityGroup", params, nil)
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := map[string]string{
		"VpcId": vpcId,
	}
	return self.ec2Request("DeleteVpc", params, nil)
}

func (self *SRegion) GetInternetGateways(vpcId string) ([]SInternetGateway, error) {
	params := map[string]string{}
	if len(vpcId) > 0 {
		params["Filter.1.attachment.vpc-id"] = vpcId
	}
	ret := []SInternetGateway{}
	for {
		result := struct {
			InternetGateways []SInternetGateway `xml:"internetGatewaySet>item"`
			NextToken        string             `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeInternetGateways", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeInternetGateways")
		}
		ret = append(ret, result.InternetGateways...)
		if len(result.NextToken) == 0 || len(result.InternetGateways) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}
