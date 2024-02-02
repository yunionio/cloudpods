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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SUserCIDRs struct {
	UserCidr []string
}

type SVpc struct {
	multicloud.SVpc
	AwsTags

	region *SRegion

	VpcId                   string `xml:"vpcId"`
	CidrBlock               string `xml:"cidrBlock"`
	CidrBlockAssociationSet []struct {
		CidrBlock      string `xml:"cidrBlock"`
		AssociationId  string `xml:"associationId"`
		CidrBlockState struct {
			State string `xml:"state"`
		} `xml:"cidrBlockState"`
	} `xml:"cidrBlockAssociationSet>item"`
	IPv6CidrBlockAssociationSet []struct {
		IPv6CidrBlock string `xml:"ipv6CidrBlock"`
	} `xml:"ipv6CidrBlockAssociationSet>item"`
	IsDefault       bool   `xml:"isDefault"`
	Status          string `xml:"state"`
	InstanceTenancy string `xml:"instanceTenancy"`
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

func (self *SVpc) GetStatus() string {
	// 目前不支持专用主机
	if self.InstanceTenancy == "dedicated" {
		return api.VPC_STATUS_UNAVAILABLE
	}
	// pending | available
	return strings.ToLower(self.Status)
}

func (self *SVpc) Refresh() error {
	new, err := self.region.getVpc(self.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	cidr := []string{self.CidrBlock}
	for _, ip := range self.CidrBlockAssociationSet {
		if !utils.IsInStringArray(ip.CidrBlock, cidr) {
			cidr = append(cidr, ip.CidrBlock)
		}
	}
	return strings.Join(cidr, ",")
}

func (self *SVpc) GetCidrBlock6() string {
	ret := []string{}
	for _, cidr := range self.IPv6CidrBlockAssociationSet {
		if len(cidr.IPv6CidrBlock) > 0 {
			ret = append(ret, cidr.IPv6CidrBlock)
		}
	}
	return strings.Join(ret, ",")
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	zones, err := self.region.GetZones("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range zones {
		zones[i].region = self.region
		ret = append(ret, &SWire{zone: &zones[i], vpc: self})
	}
	return ret, nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.region.GetSecurityGroups(self.VpcId, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetSecurityGroups")
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self.region
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	tables, err := self.region.GetRouteTables(self.GetId(), "", "", "", false)
	if err != nil {
		return nil, errors.Wrap(err, "SVpc.GetIRouteTables")
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
		if len(rt.Associations) > 0 {
			if rt.Associations[0].Main {
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
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == wireId {
			return wires[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, wireId)
}

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	peers, err := self.region.DescribeVpcPeeringConnections("", self.VpcId, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range peers {
		peers[i].vpc = self
		ret = append(ret, &peers[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	peers, err := self.region.DescribeVpcPeeringConnections("", "", self.VpcId)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range peers {
		peers[i].vpc = self
		ret = append(ret, &peers[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	peer, err := self.region.GetVpcPeeringConnectionById(id)
	if err != nil {
		return nil, err
	}
	peer.vpc = self
	return peer, nil
}

func (self *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	peer, err := self.region.CreateVpcPeeringConnection(self.VpcId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateVpcPeeringConnection")
	}
	peer.vpc = self
	return peer, nil
}

func (self *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	_, err := self.region.AcceptVpcPeeringConnection(id)
	return errors.Wrapf(err, "AcceptVpcPeeringConnection")
}

func (self *SVpc) GetAuthorityOwnerId() string {
	identity, err := self.region.client.GetCallerIdentity()
	if err != nil {
		log.Errorf(err.Error() + "self.region.client.GetCallerIdentity()")
		return ""
	}
	return identity.Account
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
	ret := struct{}{}
	err := self.region.ec2Request("AttachInternetGateway", params, &ret)
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
	return self.region.CreateRoute(rt.RouteTableId, "0.0.0.0/0", igwId)
}

func (self *SVpc) GetMainRouteTable() (*SRouteTable, error) {
	rt, err := self.region.GetRouteTables(self.GetId(), "", "", "", true)
	if err != nil {
		return nil, errors.Wrap(err, "GetRouteTables")
	}

	if len(rt) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotSupported, "GetMainRouteTable")
	}

	return &rt[0], nil
}

func (self *SVpc) DetachInternetGateways() error {
	igws, err := self.region.GetInternetGateways(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetInternetGateways")
	}

	if len(igws) > 0 {
		for i := range igws {
			err = self.region.DetachInternetGateway(self.VpcId, igws[i].GetId())
			if err != nil {
				return errors.Wrap(err, "DetachInternetGateway")
			}
		}
	}

	return nil
}

func (self *SRegion) DetachInternetGateway(vpcId, igwId string) error {
	params := map[string]string{
		"InternetGatewayId": igwId,
		"VpcId":             vpcId,
	}
	ret := struct{}{}
	return self.ec2Request("DetachInternetGateway", params, &ret)
}

func (self *SRegion) DeleteInternetGateway(id string) error {
	params := map[string]string{
		"InternetGatewayId": id,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteInternetGateway", params, &ret)
}

func (self *SVpc) DeleteInternetGateways() error {
	igws, err := self.region.GetInternetGateways(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetInternetGateways")
	}

	for i := range igws {
		err = self.region.DetachInternetGateway(self.VpcId, igws[i].GetId())
		if err != nil {
			return errors.Wrap(err, "DetachInternetGateway")
		}

		err = self.region.DeleteInternetGateway(igws[i].GetId())
		if err != nil {
			return errors.Wrap(err, "DeleteInternetGateway")
		}
	}

	return nil
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
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
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, vpcId)
}

func (self *SRegion) assignSecurityGroups(secgroupIds []string, instanceId string) error {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return errors.Wrap(err, "GetInstance")
	}

	for _, eth := range instance.NetworkInterfaces {
		params := map[string]string{
			"NetworkInterfaceId": eth.NetworkInterfaceId,
		}
		for i, groupId := range secgroupIds {
			params[fmt.Sprintf("SecurityGroupId.%d", i+1)] = groupId
		}
		ret := struct{}{}
		err = self.ec2Request("ModifyNetworkInterfaceAttribute", params, &ret)
		if err != nil {
			return errors.Wrapf(err, "ModifyNetworkInterfaceAttribute")
		}
	}

	return nil
}

func (self *SRegion) DeleteSecurityGroup(id string) error {
	params := map[string]string{
		"GroupId": id,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteSecurityGroup", params, &ret)
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := map[string]string{
		"VpcId": vpcId,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteVpc", params, &ret)
}

func (self *SRegion) GetVpcs(vpcIds []string) ([]SVpc, error) {
	params := map[string]string{}
	for i, id := range vpcIds {
		params[fmt.Sprintf("VpcId.%d", i+1)] = id
	}
	ret := []SVpc{}
	for {
		part := struct {
			NextToken string `xml:"nextToken"`
			VpcSet    []SVpc `xml:"vpcSet>item"`
		}{}
		err := self.ec2Request("DescribeVpcs", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.VpcSet...)
		if len(part.NextToken) == 0 || len(part.VpcSet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetInternetGateways(vpcId string) ([]SInternetGateway, error) {
	params := map[string]string{}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "attachment.vpc-id"
		params[fmt.Sprintf("Filter.%d.Value", idx)] = vpcId
		idx++
	}

	ret := []SInternetGateway{}
	for {
		part := struct {
			NextToken          string             `xml:"nextToken"`
			InternetGatewaySet []SInternetGateway `xml:"internetGatewaySet>item"`
		}{}
		err := self.ec2Request("DescribeInternetGateways", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeInternetGateways")
		}
		ret = append(ret, part.InternetGatewaySet...)
		if len(part.NextToken) == 0 || len(part.InternetGatewaySet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}
