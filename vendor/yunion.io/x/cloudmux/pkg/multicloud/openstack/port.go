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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	port *SPort

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
	network, err := fixip.port.region.GetNetwork(fixip.SubnetID)
	if err != nil {
		return ""
	}
	for _, pool := range network.AllocationPools {
		if pool.Contains(fixip.IpAddress) {
			network.AllocationPools = []AllocationPool{pool}
			return network.GetGlobalId()
		}
	}
	return ""
}

func (fixip *SFixedIP) IsPrimary() bool {
	return true
}

type SPort struct {
	multicloud.SNetworkInterfaceBase
	OpenStackTags
	region                  *SRegion
	AdminStateUp            bool
	AllowedAddressPairs     []string
	CreatedAt               time.Time
	DataPlaneStatus         string
	Description             string
	DeviceID                string
	DeviceOwner             string
	DnsAssignment           []DnsAssignment
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
		port.FixedIps[i].port = port
		address = append(address, &port.FixedIps[i])
	}
	return address, nil
}

func (region *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	ports, err := region.GetPorts("", "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(ports); i++ {
		if strings.HasPrefix(ports[i].DeviceOwner, "compute:") || ports[i].DeviceOwner == "network:floatingip" {
			continue
		}
		ports[i].region = region
		ret = append(ret, &ports[i])
	}
	return ret, nil
}

func (region *SRegion) GetPort(portId string) (*SPort, error) {
	resource := "/v2.0/ports/" + portId
	resp, err := region.vpcGet(resource)
	if err != nil {
		return nil, err
	}
	port := &SPort{}
	err = resp.Unmarshal(port, "port")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return port, nil
}

func (region *SRegion) GetPorts(macAddress, deviceId string) ([]SPort, error) {
	resource, ports := "/v2.0/ports", []SPort{}
	query := url.Values{}
	if len(macAddress) > 0 {
		query.Set("mac_address", macAddress)
	}
	if len(deviceId) > 0 {
		query.Set("device_id", deviceId)
	}
	for {
		resp, err := region.vpcList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "vpcList")
		}
		part := struct {
			Ports      []SPort
			PortsLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		ports = append(ports, part.Ports...)
		marker := part.PortsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return ports, nil
}
