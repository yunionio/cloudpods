package azure

import (
	"strings"

	"yunion.io/x/jsonutils"
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
	AddressPrefix           string `json:"addressPrefix,omitempty"`
	// Status string
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetId() string {
	return self.ID
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetStatus() string {
	if strings.ToLower(self.Properties.ProvisioningState) == "succeeded" || len(self.AddressPrefix) > 0 {
		return "available"
	}
	return "disabled"
}

func (self *SNetwork) Delete() error {
	vpc := self.wire.vpc
	subnets := []SNetwork{}
	if vpc.Properties.Subnets != nil {
		for i := 0; i < len(*vpc.Properties.Subnets); i++ {
			if (*vpc.Properties.Subnets)[i].Name == self.Name {
				continue
			}
			subnets = append(subnets, (*vpc.Properties.Subnets)[i])
		}
		vpc.Properties.Subnets = &subnets
		_, err := self.wire.vpc.region.client.Update(jsonutils.Marshal(vpc))
		return err
	}
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
	return true
}

func (self *SNetwork) GetServerType() string {
	return models.SERVER_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	if new, err := self.wire.zone.region.GetNetworkDetail(self.ID); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
	return nil
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}
