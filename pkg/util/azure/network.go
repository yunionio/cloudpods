package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
)

type SNetwork struct {
	wire *SWire

	AvailableIpAddressCount int
	ID                      string
	Name                    string
	Properties              SubnetPropertiesFormat

	// Status string
}

func (self *SNetwork) GetId() string {
	return fmt.Sprintf("%s/%s/%s", self.wire.zone.region.GetGlobalId(), self.wire.zone.region.SubscriptionID, self.Name)
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return self.GetId()
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetStatus() string {
	if strings.ToLower(self.Properties.ProvisioningState) == "succeeded" {
		return "avaliable"
	}
	return "disabled"
}

func (self *SNetwork) Delete() error {
	return nil
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	return pref.MaskLen
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIsPublic() bool {
	// return self.IsDefault
	return true
}

func (self *SNetwork) GetServerType() string {
	return models.SERVER_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	// log.Debugf("vsiwtch refresh %s", self.VSwitchId)
	// new, err := self.wire.zone.region.getVSwitch(self.VSwitchId)
	// if err != nil {
	// 	return err
	// }
	// return jsonutils.Update(self, new)
	return nil
}
