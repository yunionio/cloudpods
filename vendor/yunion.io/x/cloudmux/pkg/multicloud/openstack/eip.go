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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	multicloud.SEipBase
	OpenStackTags

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
	Id                string      `json:"id"`
	QosPolicyId       string      `json:"qos_policy_id"`
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	resource := fmt.Sprintf("/v2.0/floatingips/%s", eipId)
	resp, err := region.vpcGet(resource)
	if err != nil {
		return nil, err
	}
	eip := &SEipAddress{region: region}
	err = resp.Unmarshal(eip, "floatingip")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return eip, nil
}

func (region *SRegion) GetEipByIp(ip string) (*SEipAddress, error) {
	eips, err := region.GetEips(ip)
	if err != nil {
		return nil, errors.Wrap(err, "GetEips")
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

func (region *SRegion) GetEips(ip string) ([]SEipAddress, error) {
	resource := "/v2.0/floatingips"
	eips := []SEipAddress{}
	query := url.Values{}
	if len(ip) > 0 {
		query.Set("floating_ip_address", ip)
	}
	for {
		part := struct {
			Floatingips      []SEipAddress
			FloatingipsLinks SNextLinks
		}{}
		resp, err := region.vpcList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "vpcList")
		}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		eips = append(eips, part.Floatingips...)
		marker := part.FloatingipsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return eips, nil
}

func (eip *SEipAddress) GetId() string {
	return eip.Id
}

func (eip *SEipAddress) GetName() string {
	return eip.FloatingIPAddress
}

func (eip *SEipAddress) GetGlobalId() string {
	return eip.Id
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
		log.Errorf("Unknown eip %s status %s", eip.Id, eip.Status)
		return api.EIP_STATUS_UNKNOWN
	}
}

func (eip *SEipAddress) Refresh() error {
	_eip, err := eip.region.GetEip(eip.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(eip, _eip)
}

func (eip *SEipAddress) IsEmulated() bool {
	return false
}

func (eip *SEipAddress) GetIpAddr() string {
	return eip.FloatingIPAddress
}

func (eip *SEipAddress) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (eip *SEipAddress) GetAssociationType() string {
	if len(eip.GetAssociationExternalId()) > 0 {
		return api.EIP_ASSOCIATE_TYPE_SERVER
	}
	return ""
}

func (eip *SEipAddress) GetAssociationExternalId() string {
	if len(eip.PortDetails.DeviceId) > 0 {
		return eip.PortDetails.DeviceId
	}
	if len(eip.PortId) > 0 {
		port, err := eip.region.GetPort(eip.PortId)
		if err != nil {
			log.Errorf("failed to get eip port %s info", eip.PortId)
			return ""
		}
		return port.DeviceID
	}
	return ""
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
	return eip.region.DeleteEip(eip.Id)
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
		for _, pool := range network.AllocationPools {
			if pool.Contains(eip.FloatingIPAddress) {
				network.AllocationPools = []AllocationPool{pool}
				return network.GetGlobalId()
			}
		}
	}
	log.Errorf("failed to find eip %s(%s) networkId", eip.FloatingIPAddress, eip.FloatingNetworkId)
	return ""
}

func (eip *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (eip *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	return eip.region.AssociateEip(conf.InstanceId, eip.Id)
}

func (eip *SEipAddress) Dissociate() error {
	return eip.region.DisassociateEip(eip.Id)
}

func (eip *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (eip *SEipAddress) GetProjectId() string {
	return eip.ProjectId
}

func (region *SRegion) CreateEip(vpcId, networkId, ip string, projectId string) (*SEipAddress, error) {
	_, networkId = getNetworkId(networkId)
	params := map[string]map[string]string{
		"floatingip": map[string]string{
			"floating_network_id": vpcId,
			"subnet_id":           networkId,
		},
	}
	if len(projectId) > 0 {
		params["floatingip"]["tenant_id"] = projectId
	}
	if len(ip) > 0 {
		params["floatingip"]["floating_ip_address"] = ip
	}
	resource := "/v2.0/floatingips"
	resp, err := region.vpcPost(resource, params)
	if err != nil {
		return nil, errors.Wrap(err, "vpcPost")
	}
	eip := &SEipAddress{region: region}
	err = resp.Unmarshal(eip, "floatingip")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return eip, nil
}

func (region *SRegion) AssociateEip(instanceId, eipId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	for networkName, address := range instance.Addresses {
		for i := 0; i < len(address); i++ {
			if instance.Addresses[networkName][i].Type == "fixed" {
				ports, err := region.GetPorts(instance.Addresses[networkName][i].MacAddr, "")
				if err != nil {
					return err
				}

				if len(ports) == 1 {
					params := map[string]map[string]string{
						"floatingip": {
							"port_id": ports[0].ID,
						},
					}
					resource := "/v2.0/floatingips/" + eipId
					_, err = region.vpcUpdate(resource, params)
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

func (region *SRegion) AssociateEipWithPortId(portid, eipId string) error {
	params := map[string]map[string]string{
		"floatingip": {
			"port_id": portid,
		},
	}
	_, err := region.vpcUpdate("/v2.0/floatingips/"+eipId, jsonutils.Marshal(params))
	return err
}

func (region *SRegion) DisassociateEip(eipId string) error {
	params, _ := jsonutils.Parse([]byte(`{
		"floatingip": {
			"port_id": null,
		},
	}`))
	_, err := region.vpcUpdate("/v2.0/floatingips/"+eipId, params)
	return err
}

func (region *SRegion) DeleteEip(eipId string) error {
	resource := "/v2.0/floatingips/" + eipId
	_, err := region.vpcDelete(resource)
	return err
}
