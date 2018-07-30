package aliyun

import (
	"time"

	"github.com/yunionio/pkg/util/netutils"

	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

// {"AvailableIpAddressCount":4091,"CidrBlock":"172.31.32.0/20","CreationTime":"2017-03-19T13:37:44Z","Description":"System created default virtual switch.","IsDefault":true,"Status":"Available","VSwitchId":"vsw-j6c3gig5ub4fmi2veyrus","VSwitchName":"","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q","ZoneId":"cn-hongkong-b"}

type SVSwitch struct {
	wire *SWire

	AvailableIpAddressCount int
	CidrBlock               string
	CreationTime            time.Time
	Description             string
	IsDefault               bool
	Status                  string
	VSwitchId               string
	VSwitchName             string
	VpcId                   string
	ZoneId                  string
}

func (self *SVSwitch) GetId() string {
	return self.VSwitchId
}

func (self *SVSwitch) GetName() string {
	return self.VSwitchId
}

func (self *SVSwitch) GetGlobalId() string {
	return self.VSwitchId
}

func (self *SVSwitch) GetStatus() string {
	return self.Status
}

func (self *SVSwitch) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SVSwitch) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SVSwitch) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SVSwitch) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	return pref.MaskLen
}

func (self *SVSwitch) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SVSwitch) GetServerType() string {
	return models.SERVER_TYPE_GUEST
}

func (self *SVSwitch) GetIsPublic() bool {
	// return self.IsDefault
	return true
}
