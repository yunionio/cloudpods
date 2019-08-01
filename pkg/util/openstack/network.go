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
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type AllocationPool struct {
	Start string
	End   string
}

type SNextLink struct {
	Href string
	Rel  string
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
	return api.NETWORK_STATUS_AVAILABLE
}

func (network *SNetwork) Delete() error {
	return network.wire.zone.region.DeleteNetwork(network.ID)
}

func (region *SRegion) DeleteNetwork(networkId string) error {
	_, err := region.Delete("network", "/v2.0/subnets/"+networkId, "")
	return err
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
	return api.NETWORK_TYPE_GUEST
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
	url := "/v2.0/subnets"
	networks := []SNetwork{}
	for len(url) > 0 {
		_, resp, err := region.List("network", url, "", nil)
		if err != nil {
			return nil, err
		}
		_networks := []SNetwork{}
		err = resp.Unmarshal(&_networks, "subnets")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_networks, "subnets")`)
		}
		networks = append(networks, _networks...)
		url = ""
		if resp.Contains("subnets_links") {
			nextLinks := []SNextLink{}
			err = resp.Unmarshal(&nextLinks, "subnets_links")
			if err != nil {
				return nil, errors.Wrapf(err, "resp.Unmarshal(subnets_links)")
			}
			for _, next := range nextLinks {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
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

func (region *SRegion) CreateNetwork(vpcId string, name string, cidr string, desc string) (string, error) {
	params := map[string]map[string]interface{}{
		"subnet": {
			"name":        name,
			"network_id":  vpcId,
			"cidr":        cidr,
			"description": desc,
			"ip_version":  4,
		},
	}
	_, resp, err := region.Post("network", "/v2.0/subnets", "", jsonutils.Marshal(params))
	if err != nil {
		return "", err
	}
	return resp.GetString("subnet", "id")
}

func (network *SNetwork) GetProjectId() string {
	return network.TenantID
}
