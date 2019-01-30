package qcloud

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SNetwork struct {
	wire *SWire

	CidrBlock               string
	Zone                    string
	SubnetId                string
	VpcId                   string
	SubnetName              string
	AvailableIpAddressCount int
	CreatedTime             time.Time
	EnableBroadcast         bool
	IsDefault               bool
	RouteTableId            string
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	if len(self.SubnetName) > 0 {
		return self.SubnetName
	}
	return self.SubnetId
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetStatus() string {
	return models.NETWORK_STATUS_AVAILABLE
}

func (self *SNetwork) Delete() error {
	return self.wire.zone.region.DeleteNetwork(self.SubnetId)
}

func (self *SRegion) DeleteNetwork(networkId string) error {
	params := make(map[string]string)
	params["SubnetId"] = networkId

	_, err := self.vpcRequest("DeleteSubnet", params)
	return err
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	return pref.MaskLen
}

func (self *SNetwork) GetIsPublic() bool {
	// return self.IsDefault
	return true
}

func (self *SNetwork) GetServerType() string {
	return models.NETWORK_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", self.SubnetId)
	new, err := self.wire.zone.region.GetNetwork(self.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SRegion) CreateNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	params := make(map[string]string)
	params["Zone"] = zoneId
	params["VpcId"] = vpcId
	params["CidrBlock"] = cidr
	params["SubnetName"] = name
	body, err := self.vpcRequest("CreateSubnet", params)
	if err != nil {
		return "", err
	}
	return body.GetString("Subnet", "SubnetId")
}
