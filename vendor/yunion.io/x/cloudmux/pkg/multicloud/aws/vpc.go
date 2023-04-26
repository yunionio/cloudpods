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

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

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

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	RegionId                string
	VpcId                   string
	VpcName                 string
	CidrBlock               string
	CidrBlockAssociationSet []string
	IsDefault               bool
	Status                  string
	InstanceTenancy         string
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
	if len(self.VpcName) > 0 {
		return self.VpcName
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
	return strings.ToLower(self.Status)
}

func (self *SVpc) Refresh() error {
	new, err := self.region.getVpc(self.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) IsPublic() bool {
	return false
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	return strings.Join(self.CidrBlockAssociationSet, ",")
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
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, errors.Wrap(err, "fetchSecurityGroups")
		}
	}
	return self.secgroups, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	tables, err := self.region.GetRouteTables(self.GetId(), false)
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
		wire := self.getWireByZoneId(networks[i].ZoneId)
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

func (self *SVpc) fetchSecurityGroups() error {
	if len(self.VpcId) == 0 {
		return fmt.Errorf("fetchSecurityGroups vpc id is empty")
	}

	secgroups, _, err := self.region.GetSecurityGroups(self.VpcId, "", "", 0, 0)
	if err != nil {
		return errors.Wrap(err, "GetSecurityGroups")
	}

	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		self.secgroups[i] = &secgroups[i]
	}
	return nil
}

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	svpcPCs, err := self.getRequesterVpcPeeringConnections()
	if err != nil {
		return nil, errors.Wrap(err, "self.getSVpcPeeringConnections()")
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range svpcPCs {
		ret = append(ret, svpcPCs[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	svpcPCs, err := self.getAccepterVpcPeeringConnections()
	if err != nil {
		return nil, errors.Wrap(err, "self.getAccepterVpcPeeringConnections()")
	}
	ret := []cloudprovider.ICloudVpcPeeringConnection{}
	for i := range svpcPCs {
		ret = append(ret, svpcPCs[i])
	}
	return ret, nil
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	vpcPc, err := self.getSVpcPeeringConnectionById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "getSVpcPeeringConnectionById(%s)", id)
	}
	return vpcPc, nil
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

func (self *SVpc) getSVpcPeeringConnectionById(id string) (*SVpcPeeringConnection, error) {
	vpcPC, err := self.region.GetVpcPeeringConnectionById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetVpcPeeringConnectionById(%s)", id)
	}
	if vpcPC.Status.Code != nil && (*vpcPC.Status.Code == ec2.VpcPeeringConnectionStateReasonCodeDeleted ||
		*vpcPC.Status.Code == ec2.VpcPeeringConnectionStateReasonCodeDeleting) {
		return nil, cloudprovider.ErrNotFound
	}
	svpcPC := SVpcPeeringConnection{}
	svpcPC.vpc = self
	svpcPC.vpcPC = vpcPC
	return &svpcPC, nil
}

func (self *SVpc) getRequesterVpcPeeringConnections() ([]*SVpcPeeringConnection, error) {
	vpcPCs, err := self.region.DescribeRequesterVpcPeeringConnections(self.VpcId)
	if err != nil {
		return nil, errors.Wrap(err, "self.region.DescribeRequesterVpcPeeringConnections()")
	}
	ivpcPCs := []*SVpcPeeringConnection{}
	for i := range vpcPCs {
		if vpcPCs[i].Status.Code != nil && (*vpcPCs[i].Status.Code == ec2.VpcPeeringConnectionStateReasonCodeDeleted ||
			*vpcPCs[i].Status.Code == ec2.VpcPeeringConnectionStateReasonCodeDeleting) {
			continue
		}
		svpcPC := SVpcPeeringConnection{}
		svpcPC.vpc = self
		svpcPC.vpcPC = vpcPCs[i]
		ivpcPCs = append(ivpcPCs, &svpcPC)
	}
	return ivpcPCs, nil
}

func (self *SVpc) getAccepterVpcPeeringConnections() ([]*SVpcPeeringConnection, error) {
	vpcPCs, err := self.region.DescribeAccepterVpcPeeringConnections(self.VpcId)
	if err != nil {
		return nil, errors.Wrap(err, "self.region.DescribeAccepterVpcPeeringConnections()")
	}
	ivpcPCs := []*SVpcPeeringConnection{}
	for i := range vpcPCs {
		if vpcPCs[i].Status.Code != nil && (*vpcPCs[i].Status.Code == ec2.VpcPeeringConnectionStateReasonCodeDeleted ||
			*vpcPCs[i].Status.Code == ec2.VpcPeeringConnectionStateReasonCodeDeleting) {
			continue
		}
		svpcPC := SVpcPeeringConnection{}
		svpcPC.vpc = self
		svpcPC.vpcPC = vpcPCs[i]
		ivpcPCs = append(ivpcPCs, &svpcPC)
	}
	return ivpcPCs, nil
}

func (self *SVpc) createSVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeeringConnection, error) {
	svpcPC := SVpcPeeringConnection{}
	vpcPC, err := self.region.CreateVpcPeeringConnection(self.VpcId, opts)
	if err != nil {
		return nil, errors.Wrapf(err, " self.region.CreateVpcPeeringConnection(%s,%s)", self.VpcId, jsonutils.Marshal(opts).String())
	}
	svpcPC.vpc = self
	svpcPC.vpcPC = vpcPC
	err = cloudprovider.WaitMultiStatus(&svpcPC, []string{api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT,
		api.VPC_PEERING_CONNECTION_STATUS_ACTIVE,
		api.VPC_PEERING_CONNECTION_STATUS_DELETING}, 5*time.Second, 60*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if svpcPC.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_DELETING {
		return nil, errors.Wrapf(cloudprovider.ErrInvalidStatus, "vpcpeeringconnection:%s  invalidate status", jsonutils.Marshal(svpcPC.vpcPC).String())
	}
	return &svpcPC, nil
}

func (self *SVpc) acceptSVpcPeeringConnection(id string) error {
	svpcPC, err := self.getSVpcPeeringConnectionById(id)
	if err != nil {
		return errors.Wrapf(err, "self.getSVpcPeeringConnectionById(%s)", id)
	}
	//	其他region 创建的连接请求,有短暂的provisioning状态
	err = cloudprovider.WaitMultiStatus(svpcPC, []string{api.VPC_PEERING_CONNECTION_STATUS_ACTIVE,
		api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT,
		api.VPC_PEERING_CONNECTION_STATUS_DELETING}, 5*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if svpcPC.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_DELETING {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, "vpcpeeringconnection:%s  invalidate status", jsonutils.Marshal(svpcPC.vpcPC).String())
	}

	if svpcPC.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT {
		_, err := self.region.AcceptVpcPeeringConnection(id)
		if err != nil {
			return errors.Wrapf(err, "self.region.AcceptVpcPeeringConnection(%s)", id)
		}
	}
	err = cloudprovider.WaitMultiStatus(svpcPC, []string{api.VPC_PEERING_CONNECTION_STATUS_ACTIVE,
		api.VPC_PEERING_CONNECTION_STATUS_DELETING}, 5*time.Second, 60*time.Second)
	if err != nil {
		return errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if svpcPC.GetStatus() == api.VPC_PEERING_CONNECTION_STATUS_DELETING {
		return errors.Wrapf(cloudprovider.ErrInvalidStatus, "vpcpeeringconnection:%s  invalidate status", jsonutils.Marshal(svpcPC.vpcPC).String())
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
	ec2Client, err := self.region.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	input := ec2.AttachInternetGatewayInput{}
	input.SetInternetGatewayId(igwId)
	input.SetVpcId(self.GetId())

	_, err = ec2Client.AttachInternetGateway(&input)
	if err != nil {
		return errors.Wrap(err, "AttachInternetGateway")
	}

	err = self.AddDefaultInternetGatewayRoute(igwId)
	if err != nil {
		return errors.Wrap(err, "AddDefaultInternetGatewayRoute")
	}

	return nil
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
	rt, err := self.region.GetRouteTables(self.GetId(), true)
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
			err = self.DetachInternetGateway(igws[i].GetId())
			if err != nil {
				return errors.Wrap(err, "DetachInternetGateway")
			}
		}
	}

	return nil
}

func (self *SVpc) DetachInternetGateway(igwId string) error {
	ec2Client, err := self.region.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	input := ec2.DetachInternetGatewayInput{}
	input.SetInternetGatewayId(igwId)
	input.SetVpcId(self.GetId())

	_, err = ec2Client.DetachInternetGateway(&input)
	if err != nil {
		return errors.Wrap(err, "DetachInternetGateway")
	}

	return nil
}

func (self *SVpc) DeleteInternetGateway(igwId string) error {
	ec2Client, err := self.region.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	input := ec2.DeleteInternetGatewayInput{}
	input.SetInternetGatewayId(igwId)

	_, err = ec2Client.DeleteInternetGateway(&input)
	if err != nil {
		return errors.Wrap(err, "DeleteInternetGateway")
	}

	return nil
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

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	if len(vpcId) == 0 {
		return nil, fmt.Errorf("GetVpc vpc id should not be empty.")
	}

	vpcs, err := self.GetVpcs([]string{vpcId})
	if err != nil {
		return nil, errors.Wrap(err, "GetVpcs")
	}
	if len(vpcs) != 1 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "getVpc")
	}
	vpcs[0].region = self
	return &vpcs[0], nil
}

func (self *SRegion) revokeSecurityGroup(secgroupId, instanceId string, keep bool) error {
	// todo : keep ? 直接使用assignSecurityGroup 即可？
	return nil
}

func (self *SRegion) assignSecurityGroup(secgroupId, instanceId string) error {
	return self.assignSecurityGroups([]*string{&secgroupId}, instanceId)
}

func (self *SRegion) assignSecurityGroups(secgroupIds []*string, instanceId string) error {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return errors.Wrap(err, "GetInstance")
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	for _, eth := range instance.NetworkInterfaces {
		params := &ec2.ModifyNetworkInterfaceAttributeInput{}
		params.SetNetworkInterfaceId(eth.NetworkInterfaceId)
		params.SetGroups(secgroupIds)

		_, err := ec2Client.ModifyNetworkInterfaceAttribute(params)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *SRegion) DeleteSecurityGroup(secGrpId string) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	params := &ec2.DeleteSecurityGroupInput{}
	params.SetGroupId(secGrpId)

	_, err = ec2Client.DeleteSecurityGroup(params)
	return errors.Wrap(err, "DeleteSecurityGroup")
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	params := &ec2.DeleteVpcInput{}
	params.SetVpcId(vpcId)

	_, err = ec2Client.DeleteVpc(params)
	return errors.Wrap(err, "DeleteVpc")
}

func (self *SRegion) GetVpcs(vpcId []string) ([]SVpc, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	params := &ec2.DescribeVpcsInput{}
	if len(vpcId) > 0 {
		params.SetVpcIds(ConvertedList(vpcId))
	}

	ret, err := ec2Client.DescribeVpcs(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, err
	}

	vpcs := []SVpc{}
	for _, item := range ret.Vpcs {
		if err := FillZero(item); err != nil {
			return nil, err
		}
		cidrBlockAssociationSet := []string{}
		for i := range item.CidrBlockAssociationSet {
			cidr := item.CidrBlockAssociationSet[i]
			if cidr.CidrBlockState.State != nil && *cidr.CidrBlockState.State == "associated" {
				cidrBlockAssociationSet = append(cidrBlockAssociationSet, *cidr.CidrBlock)
			}
		}

		tagspec := TagSpec{ResourceType: "vpc"}
		tagspec.LoadingEc2Tags(item.Tags)

		vpc := SVpc{
			region:                  self,
			RegionId:                self.RegionId,
			VpcId:                   *item.VpcId,
			VpcName:                 tagspec.GetNameTag(),
			CidrBlock:               *item.CidrBlock,
			CidrBlockAssociationSet: cidrBlockAssociationSet,
			IsDefault:               *item.IsDefault,
			Status:                  *item.State,
			InstanceTenancy:         *item.InstanceTenancy,
		}
		jsonutils.Update(&vpc.AwsTags.TagSet, item.Tags)
		vpcs = append(vpcs, vpc)
	}

	return vpcs, nil
}

func (self *SRegion) GetInternetGateways(vpcId string) ([]SInternetGateway, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	input := ec2.DescribeInternetGatewaysInput{}
	filters := make([]*ec2.Filter, 0)
	if len(vpcId) > 0 {
		filters = AppendSingleValueFilter(filters, "attachment.vpc-id", vpcId)
	}

	if len(filters) > 0 {
		input.SetFilters(filters)
	}
	output, err := ec2Client.DescribeInternetGateways(&input)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeInternetGateways")
	}

	igws := make([]SInternetGateway, len(output.InternetGateways))
	err = unmarshalAwsOutput(output, "InternetGateways", &igws)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput")
	}

	for i := range igws {
		igws[i].region = self
	}

	return igws, nil
}
