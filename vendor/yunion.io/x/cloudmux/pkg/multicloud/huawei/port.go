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

package huawei

import (
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SFixedIP struct {
	IpAddress string
	SubnetID  string
	NetworkId string
}

func (fixip *SFixedIP) GetGlobalId() string {
	return fixip.IpAddress
}

func (fixip *SFixedIP) GetIP() string {
	return fixip.IpAddress
}

func (fixip *SFixedIP) GetINetworkId() string {
	return fixip.NetworkId
}

func (fixip *SFixedIP) IsPrimary() bool {
	return true
}

type Port struct {
	multicloud.SNetworkInterfaceBase
	HuaweiTags
	region          *SRegion
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	AdminStateUp    string `json:"admin_state_up"`
	DNSName         string `json:"dns_name"`
	MACAddress      string `json:"mac_address"`
	NetworkID       string `json:"network_id"`
	TenantID        string `json:"tenant_id"`
	DeviceID        string `json:"device_id"`
	DeviceOwner     string `json:"device_owner"`
	BindingVnicType string `json:"binding:vnic_type"`
	FixedIps        []SFixedIP
}

func (port *Port) GetName() string {
	if len(port.Name) > 0 {
		return port.Name
	}
	return port.ID
}

func (port *Port) GetId() string {
	return port.ID
}

func (port *Port) GetGlobalId() string {
	return port.ID
}

func (port *Port) GetMacAddress() string {
	return port.MACAddress
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0133195888.html
func (port *Port) GetAssociateType() string {
	switch port.DeviceOwner {
	case "compute:nova":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER
	case "network:router_gateway", "network:router_interface", "network:router_interface_distributed":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_RESERVED
	case "network:dhcp":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_DHCP
	case "neutron:LOADBALANCERV2":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_LOADBALANCER
	case "neutron:VIP_PORT":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_VIP
	default:
		if strings.HasPrefix(port.DeviceOwner, "compute:") {
			return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER
		}
	}
	return port.DeviceOwner
}

func (port *Port) GetAssociateId() string {
	return port.DeviceID
}

func (port *Port) GetStatus() string {
	switch port.Status {
	case "ACTIVE", "DOWN":
		return api.NETWORK_INTERFACE_STATUS_AVAILABLE
	case "BUILD":
		return api.NETWORK_INTERFACE_STATUS_CREATING
	}
	return port.Status
}

func (port *Port) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	address := []cloudprovider.ICloudInterfaceAddress{}
	for i := 0; i < len(port.FixedIps); i++ {
		port.FixedIps[i].NetworkId = port.NetworkID
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
		if len(ports[i].DeviceID) == 0 || !utils.IsInStringArray(ports[i].DeviceOwner, []string{"compute:CCI", "compute:nova", "neutron:LOADBALANCERV2"}) {
			ports[i].region = region
			ret = append(ret, &ports[i])
		}
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ShowPort
func (self *SRegion) GetPort(id string) (*Port, error) {
	resp, err := self.list(SERVICE_VPC, "ports/"+id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get port")
	}
	port := &Port{}
	err = resp.Unmarshal(port, "port")
	if err != nil {
		return nil, err
	}
	return port, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ListPorts
func (self *SRegion) GetPorts(instanceId string) ([]Port, error) {
	ret := []Port{}
	query := url.Values{}
	if len(instanceId) > 0 {
		query.Set("device_id", instanceId)
	}
	for {
		resp, err := self.list(SERVICE_VPC, "ports", query)
		if err != nil {
			return nil, err
		}
		part := []Port{}
		err = resp.Unmarshal(&part, "ports")
		if err != nil {
			return nil, err
		}
		ret = append(ret, part...)
		if len(part) == 0 {
			break
		}
		query.Set("marker", part[len(part)-1].ID)
	}
	return ret, nil
}
