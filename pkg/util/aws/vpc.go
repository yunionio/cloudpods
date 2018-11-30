package aws

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type SUserCIDRs struct {
	UserCidr []string
}

type SVpc struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	RegionId  string
	VpcId     string
	VpcName   string
	CidrBlock string
	IsDefault bool
	Status    string
	Tags      map[string]string // 名称、描述等
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
	return self.CidrBlock
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
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) GetManagerId() string {
	return self.region.client.providerId
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
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	secgrpId := ""
	// 名称为default的安全组与aws默认安全组名冲突
	if strings.ToLower(name) == "default" {
		name = fmt.Sprintf("%s-%s", vpcId, name)
	}

	if strings.ToLower(secgroupId) == "default" {
		secgroupId = fmt.Sprintf("%s-%s", vpcId, secgroupId)
	}

	if secgroup, err := self.getSecurityGroupByTag(vpcId, secgroupId); err != nil {
		if len(desc) == 0 {
			desc = fmt.Sprintf("security group %s for vpc %s", name, vpcId)
		}

		if secgrpId, err = self.createSecurityGroup(vpcId, name, secgroupId, desc); err != nil {
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
	secgroups, _, err := self.region.GetSecurityGroups(self.VpcId, "", 0, 0)
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
	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcs[0].region = self
	return &vpcs[0], nil
}

func (self *SRegion) revokeSecurityGroup(secgroupId, instanceId string, keep bool) error {
	// todo : keep ? 直接使用assignSecurityGroup 即可？
	return nil
}

func (self *SRegion) assignSecurityGroup(secgroupId, instanceId string) error {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return err
	}

	for _, eth := range instance.NetworkInterfaces.NetworkInterface {
		params := &ec2.ModifyNetworkInterfaceAttributeInput{}
		params.SetNetworkInterfaceId(eth.NetworkInterfaceId)
		params.SetGroups([]*string{&secgroupId})

		_, err := self.ec2Client.ModifyNetworkInterfaceAttribute(params)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *SRegion) DeleteSecurityGroup(vpcId, secGrpId string) error {
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
	if err != nil {
		if strings.Contains(err.Error(), "InvalidVpcID.NotFound") {
			return nil, 0, cloudprovider.ErrNotFound
		}
		return nil, 0, err
	}

	vpcs := []SVpc{}
	for _, item := range ret.Vpcs {
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}

		vpcs = append(vpcs, SVpc{
			region: self,
			// secgroups: nil,
			RegionId:  self.RegionId,
			VpcId:     *item.VpcId,
			VpcName:   *item.VpcId,
			CidrBlock: *item.CidrBlock,
			IsDefault: *item.IsDefault,
			Status:    *item.State,
			// Tags:      *item.Tags,
		})
	}

	return vpcs, len(vpcs), nil
}
