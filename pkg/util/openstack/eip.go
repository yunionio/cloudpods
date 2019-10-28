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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SPortDetail struct {
	Status       string `json:"status"`
	Name         string `json:"name"`
	AdminStateUp bool   `json:"admin_state_up"`
	NetworkId    string `json:"network_id"`
	DeviceOwner  string `json:"device_owner"`
	MacAddress   string `json:"mac_address"`
	DeviceId     string `json:"device_id"`
}

type SEipAddress struct {
	region *SRegion

	RouterId          string      `json:"router_id"`
	Status            string      `json:"status"`
	Description       string      `json:"description"`
	Tags              []string    `json:"tags"`
	TenantId          string      `json:"tenant_id"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	FloatingNetworkId string      `json:"floating_network_id"`
	PortDetails       SPortDetail `json:"port_details"`
	FixedIPAddress    string      `json:"fixed_ip_address"`
	FloatingIPAddress string      `json:"floating_ip_address"`
	RevisionNumber    int         `json:"revision_number"`
	ProjectId         string      `json:"project_id"`
	PortId            string      `json:"port_id"`
	ID                string      `json:"id"`
	QosPolicyId       string      `json:"qos_policy_id"`
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	_, resp, err := region.Get("network", "/v2.0/floatingips/"+eipId, "", nil)
	if err != nil {
		return nil, err
	}
	eip := &SEipAddress{region: region}
	return eip, resp.Unmarshal(eip, "floatingip")
}

func (region *SRegion) GetEipByIp(ip string) (*SEipAddress, error) {
	params := url.Values{}
	if len(ip) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	params.Add("floating_ip_address", ip)
	_, resp, err := region.List("network", "/v2.0/floatingips?"+params.Encode(), "", nil)
	if err != nil {
		return nil, err
	}
	eips := []SEipAddress{}
	err = resp.Unmarshal(&eips, "floatingips")
	if err != nil {
		return nil, err
	}
	if len(eips) == 1 {
		eips[0].region = region
		return &eips[0], nil
	}
	if len(eips) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetEips() ([]SEipAddress, error) {
	url := "/v2.0/floatingips"
	eips := []SEipAddress{}
	for len(url) > 0 {
		_, resp, err := region.List("network", url, "", nil)
		if err != nil {
			return nil, err
		}
		_eips := []SEipAddress{}
		err = resp.Unmarshal(&_eips, "floatingips")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_eips, "floatingips")`)
		}
		eips = append(eips, _eips...)
		url = ""
		if resp.Contains("floatingips_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "floatingips_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "floatingips_links")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}
	return eips, nil
}

func (eip *SEipAddress) GetId() string {
	return eip.ID
}

func (eip *SEipAddress) GetName() string {
	return eip.FloatingIPAddress
}

func (eip *SEipAddress) GetGlobalId() string {
	return eip.ID
}

func (eip *SEipAddress) GetStatus() string {
	switch eip.Status {
	case "ACTIVE":
		return api.EIP_STATUS_READY
	case "DOWN": //实际是未绑定在机器上
		return api.EIP_STATUS_READY
	case "ERROR":
		return api.EIP_STATUS_UNKNOWN
	default:
		log.Errorf("Unknown eip %s status %s", eip.ID, eip.Status)
		return api.EIP_STATUS_UNKNOWN
	}
}

func (eip *SEipAddress) Refresh() error {
	new, err := eip.region.GetEip(eip.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(eip, new)
}

func (eip *SEipAddress) IsEmulated() bool {
	return false
}

func (eip *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (eip *SEipAddress) GetIpAddr() string {
	return eip.FloatingIPAddress
}

func (eip *SEipAddress) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (eip *SEipAddress) GetAssociationType() string {
	return "server"
}

func (eip *SEipAddress) GetAssociationExternalId() string {
	return eip.PortDetails.DeviceId
}

func (eip *SEipAddress) GetBillingType() string {
	return ""
}

func (eip *SEipAddress) GetCreatedAt() time.Time {
	return eip.CreatedAt
}

func (eip *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (eip *SEipAddress) Delete() error {
	return eip.region.DeleteEip(eip.ID)
}

func (eip *SEipAddress) GetBandwidth() int {
	return 0
}

func (eip *SEipAddress) GetINetworkId() string {
	networks, err := eip.region.GetNetworks(eip.FloatingNetworkId)
	if err != nil {
		log.Errorf("failed to find vpc id for eip %s(%s), error: %v", eip.FloatingIPAddress, eip.FloatingNetworkId, err)
		return ""
	}
	for _, network := range networks {
		if network.Contains(eip.FloatingIPAddress) {
			return network.ID
		}
	}
	log.Errorf("failed to find eip %s(%s) networkId", eip.FloatingIPAddress, eip.FloatingNetworkId)
	return ""
}

func (eip *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (eip *SEipAddress) Associate(instanceId string) error {
	return eip.region.AssociateEip(instanceId, eip.ID)
}

func (eip *SEipAddress) Dissociate() error {
	return eip.region.DisassociateEip(eip.ID)
}

func (eip *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (eip *SEipAddress) GetProjectId() string {
	return eip.ProjectId
}

func (region *SRegion) CreateEip(eip *cloudprovider.SEip) (*SEipAddress, error) {
	network, err := region.GetNetwork(eip.NetworkExternalId)
	if err != nil {
		log.Errorf("failed to get subnet %s", eip.NetworkExternalId)
		return nil, err
	}
	parmas := map[string]map[string]string{
		"floatingip": {
			"floating_network_id": network.NetworkID,
			"subnet_id":           network.ID,
		},
	}
	if len(eip.IP) > 0 {
		parmas["floatingip"]["floating_ip_address"] = eip.IP
	}
	_, resp, err := region.Post("network", "/v2.0/floatingips", "", jsonutils.Marshal(parmas))
	if err != nil {
		return nil, err
	}
	ieip := &SEipAddress{region: region}
	return ieip, resp.Unmarshal(ieip, "floatingip")
}

func (region *SRegion) AssociateEip(instanceId, eipId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	for networkName, address := range instance.Addresses {
		for i := 0; i < len(address); i++ {
			if instance.Addresses[networkName][i].Type == "fixed" {
				ports, err := region.GetPorts(instance.Addresses[networkName][i].MacAddr)
				if err != nil {
					return err
				}

				if len(ports) == 1 {
					params := map[string]map[string]string{
						"floatingip": {
							"port_id": ports[0].ID,
						},
					}
					_, _, err = region.Update("network", "/v2.0/floatingips/"+eipId, "", jsonutils.Marshal(params))
					return err
				}

				if len(ports) == 0 {
					log.Errorf("failed to found port for instance nic %s(%s)", instance.Addresses[networkName][i].Addr, instance.Addresses[networkName][i].MacAddr)
					return cloudprovider.ErrNotFound
				}
				return cloudprovider.ErrDuplicateId
			}
		}
	}
	return fmt.Errorf("failed to found instnace %s nics for binding eip", instanceId)
}

func (region *SRegion) DisassociateEip(eipId string) error {
	params, _ := jsonutils.Parse([]byte(`{
		"floatingip": {
			"port_id": null,
		},
	}`))
	_, _, err := region.Update("network", "/v2.0/floatingips/"+eipId, "", params)
	return err
}

func (region *SRegion) DeleteEip(eipId string) error {
	_, err := region.Delete("network", "/v2.0/floatingips/"+eipId, "")
	return err
}
