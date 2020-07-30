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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type AllocationPool struct {
	Start string
	End   string
}

type SNextLinks []SNextLink

func (links SNextLinks) GetNextMark() string {
	for _, link := range links {
		if link.Rel == "next" && len(link.Href) > 0 {
			href, err := url.Parse(link.Href)
			if err != nil {
				log.Errorf("failed parse next link %s error: %v", link.Href, err)
				continue
			}
			marker := href.Query().Get("marker")
			if len(marker) > 0 {
				return marker
			}
		}
	}
	return ""
}

type SNextLink struct {
	Href string
	Rel  string
}

type SNetwork struct {
	wire *SWire

	Name            string
	EnableDhcp      bool
	NetworkId       string
	SegmentId       string
	ProjectId       string
	TenantId        string
	DnsNameservers  []string
	AllocationPools []AllocationPool
	HostRoutes      []string
	IpVersion       int
	GatewayIP       string
	CIDR            string
	Id              string
	CreatedAt       time.Time
	Description     string
	Ipv6AddressMode string
	Ipv6RaMode      string
	RevisionNumber  int
	ServiceTypes    []string
	SubnetpoolId    string
	Tags            []string
	UpdatedAt       time.Time
}

func (network *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (network *SNetwork) GetId() string {
	return network.Id
}

func (network *SNetwork) GetName() string {
	if len(network.Name) > 0 {
		return network.Name
	}
	return network.Id
}

func (network *SNetwork) GetGlobalId() string {
	return network.Id
}

func (network *SNetwork) IsEmulated() bool {
	return false
}

func (network *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (network *SNetwork) Delete() error {
	return network.wire.zone.region.DeleteNetwork(network.Id)
}

func (region *SRegion) DeleteNetwork(networkId string) error {
	resource := fmt.Sprintf("/v2.0/subnets/%s", networkId)
	_, err := region.vpcDelete(resource)
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
	pref, _ := netutils.NewIPV4Prefix(network.CIDR)
	return pref.MaskLen
}

func (network *SNetwork) GetIsPublic() bool {
	return network.wire.vpc.Shared
}

func (network *SNetwork) GetPublicScope() rbacutils.TRbacScope {
	if network.wire.vpc.Shared {
		return rbacutils.ScopeSystem
	}
	return rbacutils.ScopeNone
}

func (network *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (region *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	resource := fmt.Sprintf("/v2.0/subnets/%s", networkId)
	resp, err := region.vpcGet(resource)
	if err != nil {
		return nil, err
	}
	network := &SNetwork{}
	err = resp.Unmarshal(network, "subnet")
	if err != nil {
		return nil, err
	}
	return network, nil
}

func (region *SRegion) GetNetworks(vpcId string) ([]SNetwork, error) {
	resource := "/v2.0/subnets"
	networks := []SNetwork{}
	query := url.Values{}
	if len(vpcId) > 0 {
		query.Set("network_id", vpcId)
	}
	for {
		resp, err := region.vpcList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "vpcList")
		}

		part := struct {
			Subnets      []SNetwork
			SubnetsLinks SNextLinks
		}{}

		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}

		networks = append(networks, part.Subnets...)
		marker := part.SubnetsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return networks, nil
}

func (network *SNetwork) Refresh() error {
	_network, err := network.wire.zone.region.GetNetwork(network.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(network, _network)
}

func (region *SRegion) CreateNetwork(vpcId string, projectId, name string, cidr string, desc string) (*SNetwork, error) {
	params := map[string]map[string]interface{}{
		"subnet": {
			"name":        name,
			"network_id":  vpcId,
			"cidr":        cidr,
			"description": desc,
			"ip_version":  4,
		},
	}
	if len(projectId) > 0 {
		params["subnet"]["project_id"] = projectId
	}
	resp, err := region.vpcPost("/v2.0/subnets", params)
	if err != nil {
		return nil, errors.Wrap(err, "vpcPost")
	}
	network := &SNetwork{}
	err = resp.Unmarshal(network, "subnet")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return network, nil
}

func (network *SNetwork) GetProjectId() string {
	return network.TenantId
}
