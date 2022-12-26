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

package hcs

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucketAllocationPools struct {
	End   string `json:"end"`
	Start string `json:"start"`
}

type SExternalNetwork struct {
	multicloud.SResourceBase
	HcsTags
	wire *SExternalWire

	BucketAllocationPools []SBucketAllocationPools `json:"allocation_pools"`
	Cidr                  string                   `json:"cidr"`
	Description           string                   `json:"description"`
	DnsNameservers        []string                 `json:"dns_nameservers"`
	EnableDhcp            bool                     `json:"enable_dhcp"`
	EnableInternet        bool                     `json:"enable_internet"`
	EnableMulticast       string                   `json:"enable_multicast"`
	External              string                   `json:"external"`
	GatewayIp             string                   `json:"gateway_ip"`
	HostRoutes            []string                 `json:"host_routes"`
	Id                    string                   `json:"id"`
	IpVersion             int                      `json:"ip_version"`
	Ipv6AddressMode       string                   `json:"ipv6_address_mode"`
	Ipv6RaMode            string                   `json:"ipv6_ra_mode"`
	McastStatus           string                   `json:"mcast_status"`
	Name                  string                   `json:"name"`
	NetworkId             string                   `json:"network_id"`
	Routed                string                   `json:"routed"`
	SegmentId             string                   `json:"segment_id"`
	Tags                  []string                 `json:"tags"`
	TenantId              string                   `json:"tenant_id"`
}

func (self *SExternalNetwork) GetId() string {
	return self.Id
}

func (self *SExternalNetwork) GetName() string {
	if len(self.Name) == 0 {
		return self.Id
	}
	return self.Name
}

func (self *SExternalNetwork) GetGlobalId() string {
	return self.Id
}

func (self *SExternalNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (self *SExternalNetwork) Refresh() error {
	ret, err := self.wire.region.GetExternalNetwork(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SExternalNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SExternalNetwork) GetIpStart() string {
	for _, pool := range self.BucketAllocationPools {
		return pool.Start
	}
	return ""
}

func (self *SExternalNetwork) GetIpEnd() string {
	for _, pool := range self.BucketAllocationPools {
		return pool.End
	}
	return ""
}

func (self *SExternalNetwork) GetIpMask() int8 {
	return 8
}

func (self *SExternalNetwork) GetGateway() string {
	return self.GetIpStart()
}

func (self *SExternalNetwork) GetServerType() string {
	return api.NETWORK_TYPE_EIP
}

func (self *SExternalNetwork) GetIsPublic() bool {
	return true
}

func (self *SExternalNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SExternalNetwork) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SExternalNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) GetExternalNetwork(id string) (*SExternalNetwork, error) {
	ret := &SExternalNetwork{}
	res := fmt.Sprintf("subnets/%s", id)
	return ret, self.get("vpc", "v2.0", res, ret)
}

func (self *SRegion) GetExternalNetworks(netId string) ([]SExternalNetwork, error) {
	ret := []SExternalNetwork{}
	params := url.Values{}
	params.Set("network_id", netId)
	return ret, self.list("vpc", "v2.0", "subnets", params, &ret)
}

func (self *SExternalNetwork) GetProjectId() string {
	return ""
}
