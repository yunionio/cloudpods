package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SNetwork struct {
	wire *SWire

	ZStackBasic
	L3NetworkUUID string `json:"l3NetworkUuid"`
	StartIP       string `json:"startIp"`
	EndIP         string `json:"endIp"`
	Netmask       string `json:"netmask"`
	Gateway       string `json:"gateway"`
	NetworkCIDR   string `json:"networkCidr"`
	IPVersion     int    `json:"ipVersion"`
	PrefixLen     int    `json:"prefixLen"`
	ZStackTime
}

type SL3Network struct {
	ZStackBasic
	Type          string     `json:"type"`
	ZoneUUID      string     `json:"zoneUuid"`
	L2NetworkUUID string     `json:"l2NetworkUuid"`
	State         string     `json:"state"`
	System        bool       `json:"system"`
	Category      bool       `json:"category"`
	IPVersion     int        `json:"ipVersion"`
	DNS           []string   `json:"dns"`
	Networks      []SNetwork `json:"ipRanges"`
	ZStackTime
}

func (region *SRegion) GetNetwork(zoneId, wireId, l3Id, networkId string) (*SNetwork, error) {
	networks, err := region.GetNetworks(zoneId, wireId, l3Id, networkId)
	if err != nil {
		return nil, err
	}
	if len(networks) == 1 {
		if networks[0].UUID == networkId {
			return &networks[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(networks) == 0 || len(networkId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetL3Network(zoneId string, wireId string, l3Id string) (*SL3Network, error) {
	l3Networks, err := region.GetL3Networks(zoneId, wireId, l3Id)
	if err != nil {
		return nil, err
	}
	if len(l3Networks) == 1 {
		if l3Networks[0].UUID == l3Id {
			return &l3Networks[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(l3Networks) == 0 || len(l3Id) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetL3Networks(zoneId string, wireId string, l3Id string) ([]SL3Network, error) {
	l3Networks := []SL3Network{}
	params := []string{}
	if len(zoneId) > 0 {
		params = append(params, "q=zone.uuid="+zoneId)
	}
	if len(wireId) > 0 {
		params = append(params, "q=l2NetworkUuid="+wireId)
	}
	if len(l3Id) > 0 {
		params = append(params, "q=uuid="+l3Id)
	}
	return l3Networks, region.client.listAll("l3-networks", params, &l3Networks)
}

func (region *SRegion) GetNetworks(zoneId string, wireId string, l3Id string, networkId string) ([]SNetwork, error) {
	l3Networks, err := region.GetL3Networks(zoneId, wireId, l3Id)
	if err != nil {
		return nil, err
	}
	networks := []SNetwork{}
	for i := 0; i < len(l3Networks); i++ {
		for j := 0; j < len(l3Networks[i].Networks); j++ {
			if len(networkId) == 0 || l3Networks[i].Networks[j].UUID == networkId {
				networks = append(networks, l3Networks[i].Networks[j])
			}
		}
	}
	return networks, nil
}

func (network *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (network *SNetwork) GetId() string {
	return network.UUID
}

func (network *SNetwork) GetName() string {
	return network.Name
}

func (network *SNetwork) GetGlobalId() string {
	return network.UUID
}

func (network *SNetwork) IsEmulated() bool {
	return false
}

func (network *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (network *SNetwork) Delete() error {
	return network.wire.vpc.region.DeleteNetwork(network.UUID)
}

func (region *SRegion) DeleteNetwork(networkId string) error {
	return cloudprovider.ErrNotImplemented
}

func (network *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return network.wire
}

func (network *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (network *SNetwork) GetGateway() string {
	return network.Gateway
}

func (network *SNetwork) GetIpStart() string {
	return network.StartIP
}

func (network *SNetwork) GetIpEnd() string {
	return network.EndIP
}

func (network *SNetwork) GetIPRange() netutils.IPV4AddrRange {
	start, _ := netutils.NewIPV4Addr(network.GetIpStart())
	end, _ := netutils.NewIPV4Addr(network.GetIpEnd())
	return netutils.NewIPV4AddrRange(start, end)
}

func (network *SNetwork) GetIpMask() int8 {
	return int8(network.PrefixLen)
}

func (network *SNetwork) GetIsPublic() bool {
	// return network.IsDefault
	return true
}

func (network *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (network *SNetwork) Refresh() error {
	new, err := network.wire.vpc.region.GetNetwork("", network.wire.UUID, network.L3NetworkUUID, network.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(network, new)
}

func (network *SNetwork) GetProjectId() string {
	return ""
}
