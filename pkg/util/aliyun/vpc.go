package aliyun

import (
	"time"

	"github.com/yunionio/onecloud/pkg/cloudprovider"
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

	secgroups []SSecurityGroup

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

func (self *SVpc) GetId() string {
	return self.VpcId
}

func (self *SVpc) GetName() string {
	return self.VpcId
}

func (self *SVpc) GetGlobalId() string {
	return self.VpcId
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetStatus() string {
	return self.Status
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
	switches, _, err := self.region.GetVSwitches(self.VSwitchIds.VSwitchId, 0, len(self.VSwitchIds.VSwitchId))
	if err != nil {
		return err
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
	self.secgroups = secgroups
	return nil
}

func (self *SVpc) GetSecurityGroups() ([]SSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}
