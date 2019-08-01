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

	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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

type SFixedIP struct {
	IpAddress string
	SubnetID  string
}

func (fixip *SFixedIP) GetGlobalId() string {
	return fixip.IpAddress
}

func (fixip *SFixedIP) GetIP() string {
	return fixip.IpAddress
}

func (fixip *SFixedIP) GetINetworkId() string {
	return fixip.SubnetID
}

func (fixip *SFixedIP) IsPrimary() bool {
	return true
}

type SPort struct {
	multicloud.SNetworkInterfaceBase
	region                  *SRegion
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
	FixedIps                []SFixedIP
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

func (port *SPort) GetName() string {
	if len(port.Name) > 0 {
		return port.Name
	}
	return port.ID
}

func (port *SPort) GetId() string {
	return port.ID
}

func (port *SPort) GetGlobalId() string {
	return port.ID
}

func (port *SPort) GetMacAddress() string {
	return port.MacAddress
}

func (port *SPort) GetAssociateType() string {
	switch port.DeviceOwner {
	case "compute:nova":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER
	case "network:router_gateway", "network:dhcp", "network:router_interface":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_RESERVED
	}
	return port.DeviceOwner
}

func (port *SPort) GetAssociateId() string {
	return port.DeviceID
}

func (port *SPort) GetStatus() string {
	switch port.Status {
	case "ACTIVE", "DOWN":
		return api.NETWORK_INTERFACE_STATUS_AVAILABLE
	case "BUILD":
		return api.NETWORK_INTERFACE_STATUS_CREATING
	}
	return port.Status
}

func (port *SPort) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	address := []cloudprovider.ICloudInterfaceAddress{}
	for i := 0; i < len(port.FixedIps); i++ {
		address = append(address, &port.FixedIps[i])
	}
	return address, nil
}

func (region *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	ports, err := region.GetPorts("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(ports); i++ {
		if len(ports[i].DeviceID) == 0 || ports[i].DeviceOwner != "compute:nova" {
			ports[i].region = region
			ret = append(ret, &ports[i])
		}
	}
	return ret, nil
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
