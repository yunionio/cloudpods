package openstack

import (
	"fmt"
	"net/url"
	"time"
)

type DnsAssignment struct {
	Hostname  string
	IpAddress string
	Fqdn      string
}

type ExtraDhcpOpt struct {
	OptValue  string
	IpVersion int
	OptName   string
}

type FixedIPs struct {
	IpAddress string
	SubnetID  string
}

type SPort struct {
	AdminStateUp            bool
	AllowedAddressPairs     []string
	CreatedAt               time.Time
	DataPlaneStatus         string
	Description             string
	DeviceID                string
	DeviceOwner             string
	DnsAssignment           DnsAssignment
	DnsDomain               string
	DnsName                 string
	ExtraDhcpOpts           []ExtraDhcpOpt
	FixedIps                []FixedIPs
	ID                      string
	IpAllocation            string
	MacAddress              string
	Name                    string
	NetworkID               string
	ProjectID               string
	RevisionNumber          int
	SecurityGroups          []string
	Status                  string
	Tags                    []string
	TenantID                string
	UpdatedAt               time.Time
	QosPolicyID             string
	PortSecurityEnabled     bool
	UplinkStatusPropagation bool
}

func (region *SRegion) GetPorts(macAddress string) ([]SPort, error) {
	base := fmt.Sprintf("/v2.0/ports")
	params := url.Values{}
	if len(macAddress) > 0 {
		params.Set("mac_address", macAddress)
	}
	url := fmt.Sprintf("%s?%s", base, params.Encode())
	_, resp, err := region.Get("network", url, "", nil)
	if err != nil {
		return nil, err
	}
	ports := []SPort{}
	return ports, resp.Unmarshal(&ports, "ports")
}
