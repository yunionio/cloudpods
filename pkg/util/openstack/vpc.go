package openstack

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	VPC_STATUS_ACTIVE = "ACTIVE"
	VPC_STATUS_DOWN   = "DOWN"
	VPC_STATUS_BUILD  = "BUILD"
	VPC_STATUS_ERROR  = "ERROR"
)

type SVpc struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	AdminStateUp          bool
	AvailabilityZoneHints []string
	AvailabilityZones     []string
	CreatedAt             time.Time
	DnsDomain             string
	ID                    string
	Ipv4AddressScope      string
	Ipv6AddressScope      string
	L2Adjacency           bool
	Mtu                   int
	Name                  string
	PortSecurityEnabled   bool
	ProjectID             string
	QosPolicyID           string
	RevisionNumber        int
	External              bool `json:"router:external"`
	Shared                bool
	Status                string
	Subnets               []string
	TenantID              string
	UpdatedAt             time.Time
	VlanTransparent       bool
	Fescription           string
	IsDefault             bool
}

func (vpc *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (vpc *SVpc) GetId() string {
	return vpc.ID
}

func (vpc *SVpc) GetName() string {
	if len(vpc.Name) > 0 {
		return vpc.Name
	}
	return vpc.ID
}

func (vpc *SVpc) GetGlobalId() string {
	return vpc.ID
}

func (vpc *SVpc) IsEmulated() bool {
	return false
}

func (vpc *SVpc) GetIsDefault() bool {
	return vpc.IsDefault
}

func (vpc *SVpc) GetCidrBlock() string {
	return ""
}

func (vpc *SVpc) GetStatus() string {
	switch vpc.Status {
	case VPC_STATUS_ACTIVE:
		return api.VPC_STATUS_AVAILABLE
	case VPC_STATUS_BUILD, VPC_STATUS_DOWN:
		return api.VPC_STATUS_PENDING
	case VPC_STATUS_ERROR:
		return api.VPC_STATUS_FAILED
	default:
		return api.VPC_STATUS_UNKNOWN
	}
}

func (vpc *SVpc) Delete() error {
	return vpc.region.DeleteVpc(vpc.ID)
}

func (region *SRegion) DeleteVpc(vpcId string) error {
	_, err := region.Delete("network", "/v2.0/networks/"+vpcId, "")
	return err
}

func (vpc *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := vpc.region.GetSecurityGroups()
	if err != nil {
		return nil, err
	}
	iSecgroups := []cloudprovider.ICloudSecurityGroup{}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = vpc
		iSecgroups = append(iSecgroups, &secgroups[i])
	}
	return iSecgroups, nil
}

func (vpc *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	return rts, nil
}

func (vpc *SVpc) fetchWires() error {
	if len(vpc.region.izones) == 0 {
		if err := vpc.region.fetchZones(); err != nil {
			return err
		}
	}
	wire := SWire{zone: vpc.region.izones[0].(*SZone), vpc: vpc}
	vpc.iwires = []cloudprovider.ICloudWire{&wire}
	return nil
}

func (vpc *SVpc) getWire() *SWire {
	if vpc.iwires == nil {
		vpc.fetchWires()
	}
	return vpc.iwires[0].(*SWire)
}

func (vpc *SVpc) fetchNetworks() error {
	networks, err := vpc.region.GetNetworks(vpc.ID)
	if err != nil {
		return err
	}
	for i := 0; i < len(networks); i++ {
		wire := vpc.getWire()
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
}

func (vpc *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if vpc.iwires == nil {
		err := vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(vpc.iwires); i++ {
		if vpc.iwires[i].GetGlobalId() == wireId {
			return vpc.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (vpc *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if vpc.iwires == nil {
		err := vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return vpc.iwires, nil
}

func (vpc *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.region
}

func (region *SRegion) GetVpc(vpcId string) (*SVpc, error) {
	_, resp, err := region.Get("network", "/v2.0/networks/"+vpcId, "", nil)
	if err != nil {
		return nil, err
	}
	vpc := SVpc{}
	return &vpc, resp.Unmarshal(&vpc, "network")
}

func (region *SRegion) GetVpcs() ([]SVpc, error) {
	url := "/v2.0/networks"
	vpcs := []SVpc{}
	for len(url) > 0 {
		_, resp, err := region.List("network", url, "", nil)
		if err != nil {
			return nil, err
		}
		_vpcs := []SVpc{}
		err = resp.Unmarshal(&_vpcs, "networks")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_vpcs, "networks")`)
		}
		vpcs = append(vpcs, _vpcs...)
		url = ""
		if resp.Contains("networks_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "networks_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "networks_links")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}
	return vpcs, nil
}

func (vpc *SVpc) Refresh() error {
	new, err := vpc.region.GetVpc(vpc.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(vpc, new)
}

func (vpc *SVpc) addWire(wire *SWire) {
	if vpc.iwires == nil {
		vpc.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	vpc.iwires = append(vpc.iwires, wire)
}
