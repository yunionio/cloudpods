// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openstack

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/pkg/errors"
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

	ports := []SPort{}
	for len(url) > 0 {
		_, resp, err := region.List("network", url, "", nil)
		if err != nil {
			return nil, err
		}
		_ports := []SPort{}
		err = resp.Unmarshal(&_ports, "ports")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_ports, "ports")`)
		}
		ports = append(ports, _ports...)
		url = ""
		if resp.Contains("ports_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "ports_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "ports_links")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}

	return ports, nil
}
