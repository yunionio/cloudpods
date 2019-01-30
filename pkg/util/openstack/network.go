package openstack

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
)

type AllocationPool struct {
	Start string
	End   string
}

type SNetwork struct {
	wire *SWire

	Name            string
	EnableDhcp      bool
	NetworkID       string
	SegmentID       string
	ProjectID       string
	TenantID        string
	DnsNameservers  []string
	AllocationPools []AllocationPool
	HostRoutes      []string
	IpVersion       int
	GatewayIP       string
	CIDR            string
	ID              string
	CreatedAt       time.Time
	Description     string
	Ipv6AddressMode string
	Ipv6RaMode      string
	RevisionNumber  int
	ServiceTypes    []string
	SubnetpoolID    string
	Tags            []string
	UpdatedAt       time.Time
}

func (network *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (network *SNetwork) GetId() string {
	return network.ID
}

func (network *SNetwork) GetName() string {
	if len(network.Name) > 0 {
		return network.Name
	}
	return network.ID
}

func (network *SNetwork) GetGlobalId() string {
	return network.ID
}

func (network *SNetwork) IsEmulated() bool {
	return false
}

func (network *SNetwork) GetStatus() string {
	return models.NETWORK_STATUS_AVAILABLE
}

func (network *SNetwork) Delete() error {
	return network.wire.zone.region.DeleteNetwork(network.ID)
}

func (network *SRegion) DeleteNetwork(networkId string) error {
	return cloudprovider.ErrNotImplemented
}

func (network *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return network.wire
}

func (network *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (network *SNetwork) GetGateway() string {
	return network.GatewayIP
}

func (network *SNetwork) GetIpStart() string {
	if len(network.AllocationPools) >= 1 {
		return network.AllocationPools[0].Start
	}
	return ""
}

func (network *SNetwork) GetIpEnd() string {
	if len(network.AllocationPools) >= 1 {
		return network.AllocationPools[0].End
	}
	return ""
}

func (network *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(network.CIDR)
	return pref.MaskLen
}

func (network *SNetwork) GetIsPublic() bool {
	return true
}

func (network *SNetwork) GetServerType() string {
	return models.NETWORK_TYPE_GUEST
}

func (region *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	_, resp, err := region.Get("network", "/v2.0/subnets/"+networkId, "", nil)
	if err != nil {
		return nil, err
	}
	network := SNetwork{}
	return &network, resp.Unmarshal(&network, "subnet")
}

func (region *SRegion) GetNetworks(vpcId string) ([]SNetwork, error) {
	_, resp, err := region.Get("network", "/v2.0/subnets", "", nil)
	if err != nil {
		return nil, err
	}
	networks := []SNetwork{}
	if err := resp.Unmarshal(&networks, "subnets"); err != nil {
		return nil, err
	}
	result := []SNetwork{}
	for i := 0; i < len(networks); i++ {
		if len(vpcId) == 0 || vpcId == networks[i].NetworkID {
			result = append(result, networks[i])
		}
	}
	return result, nil
}

func (network *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", network.Name)
	new, err := network.wire.zone.region.GetNetwork(network.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(network, new)
}

func (network *SRegion) CreateNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
