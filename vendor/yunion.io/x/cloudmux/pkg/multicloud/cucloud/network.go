package cucloud

import (
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
)

type SNetwork struct {
	multicloud.SResourceBase
	multicloud.STagBase

	wire *SWire

	VpcName               string
	CloudRegionId         string
	TimeConsume           float64
	NetworkName           string
	SubNetworkDNS         string
	RequestId             string
	VpcId                 string
	SubNetworkCidr        string
	SubNetworkIPRange     string
	AzId                  string
	ZoneId                string
	NetworkID             string
	SecurityZoneName      string
	StartTime             string
	ZoneName              string
	SubNetworkId          string
	NetworkType           string
	NetworkUUID           string
	CloudRegionName       string
	StatusEn              string
	IsDefaultSubnet       string
	LoadBalanceNum        int
	VirtualMachineNum     int
	SubNetworkUUID        string
	NetCardNum            int
	RelationId            string
	UserId                string
	Ipv4Id                string
	SubNetworkWhetherDhcp string
	SubNetworkIpVersion   string
	Ipv6Id                string
	CloudRegionCode       string
	CloudRegionNature     string
	AccountId             string
	CreateTime            string
	SubNetworkGatewayIp   string
	SubNetworkName        string
	EndTime               string
	VlanPlanId            string
	ZoneCode              string
}

func (self *SNetwork) GetId() string {
	return self.SubNetworkId
}

func (self *SNetwork) GetName() string {
	return self.SubNetworkName
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubNetworkId
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (self *SNetwork) Refresh() error {
	return nil
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIp6Start() string {
	return ""
}

func (self *SNetwork) GetIp6End() string {
	return ""
}

func (self *SNetwork) GetIp6Mask() uint8 {
	return 0
}

func (self *SNetwork) GetGateway6() string {
	return ""
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.SubNetworkCidr)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.SubNetworkCidr)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.SubNetworkCidr)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	return self.SubNetworkGatewayIp
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (region *SRegion) GetNetworks(vpcId, zoneCode string) ([]SNetwork, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	if len(vpcId) > 0 {
		params.Set("vpcId", vpcId)
	}
	if len(zoneCode) > 0 {
		params.Set("zoneCode", zoneCode)
	}
	resp, err := region.list("/instance/v1/product/subnets", params)
	if err != nil {
		return nil, err
	}
	ret := []SNetwork{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	result := []SNetwork{}
	for i := range ret {
		if len(zoneCode) == 0 || ret[i].ZoneCode == zoneCode {
			result = append(result, ret[i])
		}
	}
	return result, nil
}
