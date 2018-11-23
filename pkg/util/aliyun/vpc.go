package aliyun

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
)

const (
	VpcAvailable = "Available"
	VpcPending   = "Pending"
)

// "CidrBlock":"172.31.0.0/16","CreationTime":"2017-03-19T13:37:40Z","Description":"System created default VPC.","IsDefault":true,"RegionId":"cn-hongkong","Status":"Available","UserCidrs":{"UserCidr":[]},"VRouterId":"vrt-j6c00qrol733dg36iq4qj","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q","VpcName":""

type SUserCIDRs struct {
	UserCidr []string
}

type SVSwitchIds struct {
	VSwitchId []string
}

type SVpc struct {
	region *SRegion

	iwires []cloudprovider.ICloudWire

	secgroups   []cloudprovider.ICloudSecurityGroup
	routeTables []cloudprovider.ICloudRouteTable

	CidrBlock    string
	CreationTime time.Time
	Description  string
	IsDefault    bool
	RegionId     string
	Status       string
	UserCidrs    SUserCIDRs
	VRouterId    string
	VSwitchIds   SVSwitchIds
	VpcId        string
	VpcName      string
}

func (self *SVpc) apiRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.region.getSdkClient()
	if err != nil {
		return nil, err
	}
	return _jsonRequest(client, "vpc.aliyuncs.com", ALIYUN_API_VERSION_VPC, action, params)
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

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) getWireByZoneId(zoneId string) *SWire {
	for i := 0; i <= len(self.iwires); i += 1 {
		wire := self.iwires[i].(*SWire)
		if wire.zone.ZoneId == zoneId {
			return wire
		}
	}
	return nil
}

func (self *SVpc) fetchVSwitches() error {
	switches, total, err := self.region.GetVSwitches(nil, self.VpcId, 0, 50)
	if err != nil {
		return err
	}
	if total > len(switches) {
		switches, _, err = self.region.GetVSwitches(nil, self.VpcId, 0, total)
		if err != nil {
			return err
		}
	}
	for i := 0; i < len(switches); i += 1 {
		wire := self.getWireByZoneId(switches[i].ZoneId)
		switches[i].wire = wire
		wire.addNetwork(&switches[i])
	}
	return nil
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchVSwitches()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchVSwitches()
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

func (self *SVpc) fetchSecurityGroups() error {
	secgroups := make([]SSecurityGroup, 0)
	for {
		parts, total, err := self.region.GetSecurityGroups(self.VpcId, len(secgroups), 50)
		if err != nil {
			return err
		}
		secgroups = append(secgroups, parts...)
		if len(secgroups) >= total {
			break
		}
	}
	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = self
		self.secgroups[i] = &secgroups[i]
	}
	return nil
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

func (self *SVpc) fetchRouteTables() error {
	routeTables := make([]*SRouteTable, 0)
	for {
		parts, total, err := self.RemoteGetRouteTableList(len(routeTables), 50)
		if err != nil {
			return err
		}
		routeTables = append(routeTables, parts...)
		if len(routeTables) >= total {
			break
		}
	}
	self.routeTables = make([]cloudprovider.ICloudRouteTable, len(routeTables))
	for i := 0; i < len(routeTables); i++ {
		routeTables[i].vpc = self
		self.routeTables[i] = routeTables[i]
	}
	return nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	if self.routeTables == nil {
		err := self.fetchRouteTables()
		if err != nil {
			return nil, err
		}
	}
	return self.routeTables, nil
}

func (self *SVpc) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SVpc) Delete() error {
	err := self.fetchSecurityGroups()
	if err != nil {
		log.Errorf("fetchSecurityGroup for VPC delete fail %s", err)
		return err
	}
	for i := 0; i < len(self.secgroups); i += 1 {
		secgroup := self.secgroups[i].(*SSecurityGroup)
		err := self.region.deleteSecurityGroup(secgroup.SecurityGroupId)
		if err != nil {
			log.Errorf("deleteSecurityGroup for VPC delete fail %s", err)
			return err
		}
	}
	return self.region.DeleteVpc(self.VpcId)
}

func (self *SVpc) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) (string, error) {
	secgrpId := ""
	if secgroup, err := self.region.getSecurityGroupByTag(self.VpcId, secgroupId); err != nil {
		if secgrpId, err = self.region.createSecurityGroup(self.VpcId, name, ""); err != nil {
			return "", err
		} else if err := self.region.addTagToSecurityGroup(secgrpId, "id", secgroupId, 1); err != nil {
			return "", err
		}
		//addRules
		log.Debugf("Add Rules for %s", secgrpId)
		for _, rule := range rules {
			if err := self.region.addSecurityGroupRule(secgrpId, &rule); err != nil {
				return "", err
			}
		}
	} else {
		//syncRules
		secgrpId = secgroup.SecurityGroupId
		log.Debugf("Sync Rules for %s", secgroup.GetName())
		if secgroup.GetName() != name {
			if err := self.region.modifySecurityGroup(secgrpId, name, ""); err != nil {
				log.Errorf("Change SecurityGroup name to %s failed: %v", name, err)
			}
		}
		self.region.syncSecgroupRules(secgrpId, rules)
	}
	return secgrpId, nil
}

func (self *SVpc) assignSecurityGroup(secgroupId string, instanceId string) error {
	return self.region.assignSecurityGroup(secgroupId, instanceId)
}

func (self *SVpc) revokeSecurityGroup(secgroupId string, instanceId string, keep bool) error {
	return self.region.revokeSecurityGroup(secgroupId, instanceId, keep)
}
