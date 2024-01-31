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

package zstack

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	ZStackTags
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

type SHostRoute struct {
	ID            int
	L3NetworkUuid string `json:"l3NetworkUuid"`
	Prefix        string
	Nexthop       string
	ZStackTime
}

type SL3Network struct {
	ZStackBasic
	Type          string       `json:"type"`
	ZoneUUID      string       `json:"zoneUuid"`
	L2NetworkUUID string       `json:"l2NetworkUuid"`
	State         string       `json:"state"`
	System        bool         `json:"system"`
	Category      bool         `json:"category"`
	IPVersion     int          `json:"ipVersion"`
	DNS           []string     `json:"dns"`
	Networks      []SNetwork   `json:"ipRanges"`
	HostRoute     []SHostRoute `json:"hostRoute"`
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

func (region *SRegion) GetL3Network(l3Id string) (*SL3Network, error) {
	l3network := &SL3Network{}
	return l3network, region.client.getResource("l3-networks", l3Id, l3network)
}

func (region *SRegion) GetL3Networks(zoneId string, wireId string, l3Id string) ([]SL3Network, error) {
	l3Networks := []SL3Network{}
	params := url.Values{}
	if len(zoneId) > 0 {
		params.Add("q", "zone.uuid="+zoneId)
	}
	if len(wireId) > 0 {
		params.Add("q", "l2NetworkUuid="+wireId)
	}
	if len(l3Id) > 0 {
		params.Add("q", "uuid="+l3Id)
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

func (network *SNetwork) GetId() string {
	return network.UUID
}

func (network *SNetwork) GetName() string {
	return network.Name
}

func (network *SNetwork) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", network.L3NetworkUUID, network.UUID)
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
	network, err := region.GetNetwork("", "", "", networkId)
	if err != nil {
		return err
	}
	l3, err := region.GetL3Network(network.L3NetworkUUID)
	if err != nil {
		return err
	}
	if len(l3.Networks) == 1 {
		return region.client.delete("l3-networks", l3.UUID, "")
	}
	return region.client.delete("l3-networks/ip-ranges", networkId, "")
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

func (network *SNetwork) Contains(ipAddr string) bool {
	ip, err := netutils.NewIPV4Addr(ipAddr)
	if err != nil {
		return false
	}
	return network.GetIPRange().Contains(ip)
}

func (network *SNetwork) GetIpMask() int8 {
	return int8(network.PrefixLen)
}

func (network *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
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

func (region *SRegion) CreateNetwork(name string, cidr string, wireId string, desc string) (*SNetwork, error) {
	params := map[string]interface{}{
		"params": map[string]interface{}{
			"name":          name,
			"type":          "L3BasicNetwork",
			"l2NetworkUuid": wireId,
			"category":      "Private",
			"system":        false,
		},
	}
	l3 := &SL3Network{}
	resp, err := region.client.post("l3-networks", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(l3, "inventory")
	if err != nil {
		return nil, err
	}
	region.AttachServiceForl3Network(l3.UUID, []string{"Flat", "SecurityGroup"})
	params = map[string]interface{}{
		"params": map[string]interface{}{
			"name":        name,
			"networkCidr": cidr,
		},
	}
	resource := fmt.Sprintf("l3-networks/%s/ip-ranges/by-cidr", l3.UUID)
	resp, err = region.client.post(resource, jsonutils.Marshal(params))
	if err != nil {
		region.client.delete("l3-networks", l3.UUID, "")
		return nil, err
	}
	network := &SNetwork{}
	return network, resp.Unmarshal(network, "inventory")
}
