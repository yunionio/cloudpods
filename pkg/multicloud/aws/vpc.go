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

	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SUserCIDRs struct {
	UserCidr []string
}

type SVpc struct {
	multicloud.SVpc

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
	Tags                    map[string]string // 名称、描述等
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

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
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
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
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

func (self *SVpc) Delete() error {
	// 删除vpc会同步删除关联的安全组
	return self.region.DeleteVpc(self.VpcId)
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i += 1 {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, ErrorNotFound()
}

func (self *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	secgrpId := ""
	// 名称为default的安全组与aws默认安全组名冲突
	if strings.ToLower(name) == "default" {
		name = randomString(fmt.Sprintf("%s-", vpcId), 9)
	}

	rules = SecurityRuleSetToAllowSet(rules)
	if secgroup, err := self.getSecurityGroupById(vpcId, secgroupId); err != nil {
		if len(desc) == 0 {
			desc = fmt.Sprintf("security group %s for vpc %s", name, vpcId)
		}

		if secgrpId, err = self.CreateSecurityGroup(vpcId, name, secgroupId, desc); err != nil {
			return "", err
		}

		//addRules
		for _, rule := range rules {
			if err := self.addSecurityGroupRule(secgrpId, &rule); err != nil {
				return "", err
			}
		}
	} else {
		//syncRules
		secgrpId = secgroup.SecurityGroupId
		log.Debugf("Sync Rules for %s", secgroup.GetName())
		if secgroup.GetName() != name {
			if err := self.modifySecurityGroup(secgrpId, name, ""); err != nil {
				log.Errorf("Change SecurityGroup name to %s failed: %v", name, err)
			}
		}
		self.syncSecgroupRules(secgrpId, rules)
	}
	return secgrpId, nil
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
	networks, _, err := self.region.GetNetwroks(nil, self.VpcId, 0, 0)
	if err != nil {
		return err
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
	secgroups, _, err := self.region.GetSecurityGroups(self.VpcId, "", "", 0, 0)
	if err != nil {
		return err
	}

	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		self.secgroups[i] = &secgroups[i]
	}
	return nil
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	if len(vpcId) == 0 {
		return nil, fmt.Errorf("GetVpc vpc id should not be empty.")
	}

	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, ErrorNotFound()
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
		return err
	}

	for _, eth := range instance.NetworkInterfaces.NetworkInterface {
		params := &ec2.ModifyNetworkInterfaceAttributeInput{}
		params.SetNetworkInterfaceId(eth.NetworkInterfaceId)
		params.SetGroups(secgroupIds)

		_, err := self.ec2Client.ModifyNetworkInterfaceAttribute(params)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *SRegion) DeleteSecurityGroup(secGrpId string) error {
	params := &ec2.DeleteSecurityGroupInput{}
	params.SetGroupId(secGrpId)

	_, err := self.ec2Client.DeleteSecurityGroup(params)
	return err
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := &ec2.DeleteVpcInput{}
	params.SetVpcId(vpcId)

	_, err := self.ec2Client.DeleteVpc(params)
	return err
}

func (self *SRegion) GetVpcs(vpcId []string, offset int, limit int) ([]SVpc, int, error) {
	params := &ec2.DescribeVpcsInput{}
	if len(vpcId) > 0 {
		params.SetVpcIds(ConvertedList(vpcId))
	}

	ret, err := self.ec2Client.DescribeVpcs(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, 0, err
	}

	vpcs := []SVpc{}
	for _, item := range ret.Vpcs {
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}
		cidrBlockAssociationSet := []string{}
		for i := range item.CidrBlockAssociationSet {
			cidr := item.CidrBlockAssociationSet[i]
			if *cidr.CidrBlockState.State == "associated" {
				cidrBlockAssociationSet = append(cidrBlockAssociationSet, *cidr.CidrBlock)
			}
		}

		tags := make(map[string]string, 0)
		for _, tag := range item.Tags {
			tags[*tag.Key] = *tag.Value
		}

		vpcs = append(vpcs, SVpc{
			region:                  self,
			RegionId:                self.RegionId,
			VpcId:                   *item.VpcId,
			VpcName:                 tags["Name"],
			CidrBlock:               *item.CidrBlock,
			CidrBlockAssociationSet: cidrBlockAssociationSet,
			IsDefault:               *item.IsDefault,
			Status:                  *item.State,
			InstanceTenancy:         *item.InstanceTenancy,
			Tags:                    tags,
		})
	}

	return vpcs, len(vpcs), nil
}
