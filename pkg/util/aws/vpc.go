package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
)

type SUserCIDRs struct {
	UserCidr []string
}

type SVSwitchIds struct {
	VSwitchId []string
}

type SVpc struct {
	region *SRegion

	iwires []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	RegionId     string
	VpcId        string
	CidrBlock    string
	IsDefault    bool
	Status       string
	Tags         map[string]string  // 名称、描述等
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) GetId() string {
	panic("implement me")
}

func (self *SVpc) GetName() string {
	panic("implement me")
}

func (self *SVpc) GetGlobalId() string {
	panic("implement me")
}

func (self *SVpc) GetStatus() string {
	panic("implement me")
}

func (self *SVpc) Refresh() error {
	panic("implement me")
}

func (self *SVpc) IsEmulated() bool {
	panic("implement me")
}

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	panic("implement me")
}

func (self *SVpc) GetIsDefault() bool {
	panic("implement me")
}

func (self *SVpc) GetCidrBlock() string {
	panic("implement me")
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	panic("implement me")
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	panic("implement me")
}

func (self *SVpc) GetManagerId() string {
	panic("implement me")
}

func (self *SVpc) Delete() error {
	panic("implement me")
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	panic("implement me")
}

func (self *SVpc) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) (string, error) {
	panic("implement me")
}
