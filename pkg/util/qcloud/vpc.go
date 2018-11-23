package qcloud

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/secrules"
)

type SVpc struct {
	region *SRegion

	iwires []cloudprovider.ICloudWire

	secgroups []cloudprovider.ICloudSecurityGroup

	CidrBlock       string
	CreatedTime     time.Time
	DhcpOptionsId   string
	DnsServerSet    []string
	DomainName      string
	EnableMulticast bool
	IsDefault       bool
	VpcId           string
	VpcName         string
}

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
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

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetStatus() string {
	return models.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.VpcId)
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)
	for {
		parts, total, err := self.region.GetSecurityGroups(self.VpcId, len(secgroups), 50)
		if err != nil {
			return nil, err
		}
		secgroups = append(secgroups, parts...)
		if len(secgroups) >= total {
			break
		}
	}
	isecgroups := make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		isecgroups[i] = &secgroups[i]
	}
	return isecgroups, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (self *SVpc) getWireByZoneId(zoneId string) *SWire {
	for i := 0; i <= len(self.iwires); i++ {
		wire := self.iwires[i].(*SWire)
		if wire.zone.Zone == zoneId {
			return wire
		}
	}
	return nil
}

func (self *SVpc) fetchNetworks() error {
	networks, total, err := self.region.GetNetworks(nil, self.VpcId, 0, 50)
	if err != nil {
		return err
	}
	if total > len(networks) {
		networks, _, err = self.region.GetNetworks(nil, self.VpcId, 0, total)
		if err != nil {
			return err
		}
	}
	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByZoneId(networks[i].Zone)
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
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

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) Refresh() error {
	new, err := self.region.getVpc(self.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) (string, error) {
	secgrpId := ""
	// if secgroup, err := self.region.getSecurityGroupByTag(self.VpcId, secgroupId); err != nil {
	// 	if secgrpId, err = self.region.createSecurityGroup(self.VpcId, name, ""); err != nil {
	// 		return "", err
	// 	} else if err := self.region.addTagToSecurityGroup(secgrpId, "id", secgroupId, 1); err != nil {
	// 		return "", err
	// 	}
	// 	//addRules
	// 	log.Debugf("Add Rules for %s", secgrpId)
	// 	for _, rule := range rules {
	// 		if err := self.region.addSecurityGroupRule(secgrpId, &rule); err != nil {
	// 			return "", err
	// 		}
	// 	}
	// } else {
	// 	//syncRules
	// 	secgrpId = secgroup.SecurityGroupId
	// 	log.Debugf("Sync Rules for %s", secgroup.GetName())
	// 	if secgroup.GetName() != name {
	// 		if err := self.region.modifySecurityGroup(secgrpId, name, ""); err != nil {
	// 			log.Errorf("Change SecurityGroup name to %s failed: %v", name, err)
	// 		}
	// 	}
	// 	self.region.syncSecgroupRules(secgrpId, rules)
	// }
	return secgrpId, nil
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}
