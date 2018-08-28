package aliyun

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
)

// {"AvailableIpAddressCount":4091,"CidrBlock":"172.31.32.0/20","CreationTime":"2017-03-19T13:37:44Z","Description":"System created default virtual switch.","IsDefault":true,"Status":"Available","VSwitchId":"vsw-j6c3gig5ub4fmi2veyrus","VSwitchName":"","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q","ZoneId":"cn-hongkong-b"}

const (
	VSwitchPending   = "Pending"
	VSwitchAvailable = "Available"
)

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

func (self *SVSwitch) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVSwitch) GetId() string {
	return self.VSwitchId
}

func (self *SVSwitch) GetName() string {
	if len(self.VSwitchName) > 0 {
		return self.VSwitchName
	}
	return self.VSwitchId
}

func (self *SVSwitch) GetGlobalId() string {
	return self.VSwitchId
}

func (self *SVSwitch) IsEmulated() bool {
	return false
}

func (self *SVSwitch) GetStatus() string {
	return strings.ToLower(self.Status)
}

func (self *SVSwitch) Refresh() error {
	log.Debugf("vsiwtch refresh %s", self.VSwitchId)
	new, err := self.wire.zone.region.getVSwitch(self.VSwitchId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
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

func (self *SRegion) createVSwitch(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	params := make(map[string]string)
	params["ZoneId"] = zoneId
	params["VpcId"] = vpcId
	params["CidrBlock"] = cidr
	params["VSwitchName"] = name
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := self.ecsRequest("CreateVSwitch", params)
	if err != nil {
		return "", err
	}
	return body.GetString("VSwitchId")
}

func (self *SRegion) getVSwitch(vswitchId string) (*SVSwitch, error) {
	vswitches, total, err := self.GetVSwitches([]string{vswitchId}, "", 0, 1)
	log.Debugf("getVSwitch %d %d %s %s", len(vswitches), total, err, vswitchId)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &vswitches[0], nil
}

func (self *SRegion) deleteVSwitch(vswitchId string) error {
	params := make(map[string]string)
	params["VSwitchId"] = vswitchId

	_, err := self.ecsRequest("DeleteVSwitch", params)
	return err
}

func (self *SVSwitch) Delete() error {
	return self.wire.zone.region.deleteVSwitch(self.VSwitchId)
}
