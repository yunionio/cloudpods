package huawei

import (
	"strconv"
	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
)

/*
Subnets
*/

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090590.html
type SNetwork struct {
	wire *SWire

	AvailabilityZone string   `json:"availability_zone"`
	CIDR             string   `json:"cidr"`
	DHCPEnable       bool     `json:"dhcp_enable"`
	DNSList          []string `json:"dnsList"`
	GatewayIP        string   `json:"gateway_ip"`
	ID               string   `json:"id"`
	Ipv6Enable       bool     `json:"ipv6_enable"`
	Name             string   `json:"name"`
	NeutronNetworkID string   `json:"neutron_network_id"`
	NeutronSubnetID  string   `json:"neutron_subnet_id"`
	PrimaryDNS       string   `json:"primary_dns"`
	SecondaryDNS     string   `json:"secondary_dns"`
	Status           string   `json:"status"`
	VpcID            string   `json:"vpc_id"`
}

func (self *SNetwork) GetId() string {
	return self.ID
}

func (self *SNetwork) GetName() string {
	if len(self.Name) == 0 {
		return self.ID
	}

	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return self.ID
}

func (self *SNetwork) GetStatus() string {
	return strings.ToLower(self.Status)
}

func (self *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", self.GetId())
	new, err := self.wire.zone.region.getNetwork(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetServerType() string {
	return models.SERVER_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) Delete() error {
	// todo: implement me
	return nil
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) getNetwork(networkId string) (*SNetwork, error) {
	network := SNetwork{}
	err := DoGet(self.ecsClient.Subnets.Get, networkId, nil, &network)
	return &network, err
}

func (self *SRegion) GetNetwroks(vpcId string, limit int, marker string) ([]SNetwork, int, error) {
	querys := map[string]string{}
	if len(vpcId) > 0 {
		querys["vpc_id"] = vpcId
	}

	if len(marker) > 0 {
		querys["marker"] = marker
	}

	querys["limit"] = strconv.Itoa(limit)
	networks := make([]SNetwork, 0)
	err := DoList(self.ecsClient.Subnets.List, querys, &networks)
	return networks, len(networks), err
}
