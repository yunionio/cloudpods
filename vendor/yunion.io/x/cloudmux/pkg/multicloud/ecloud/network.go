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

package ecloud

import (
	"context"

	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	EcloudTags
	SZoneRegionBase

	wire *SWire

	Id       string
	Name     string
	Shared   bool
	Enabled  bool
	EcStatus string
	Subnets  []SSubnet
}

type SSubnet struct {
	Id         string
	Name       string
	NetworkId  string
	Region     string
	GatewayIp  string
	EnableDHCP bool
	Cidr       string
	IpVersion  string
}

func (n *SNetwork) GetId() string {
	return n.Id
}

func (n *SNetwork) GetName() string {
	return n.Name
}

func (n *SNetwork) GetGlobalId() string {
	return n.Id
}

func (n *SNetwork) GetStatus() string {
	switch n.EcStatus {
	case "ACTIVE":
		return api.NETWORK_STATUS_AVAILABLE
	case "DOWN", "BUILD", "ERROR":
		return api.NETWORK_STATUS_UNAVAILABLE
	case "PENDING_DELETE":
		return api.NETWORK_STATUS_DELETING
	case "PENDING_CREATE", "PENDING_UPDATE":
		return api.NETWORK_STATUS_PENDING
	default:
		return api.NETWORK_STATUS_UNKNOWN
	}
}

func (n *SNetwork) Refresh() error {
	return nil
	// nn, err := n.wire.vpc.region.GetNetworkById(n.wire.vpc.RouterId, n.wire.zone.Region, n.GetId())
	// if err != nil {
	// 	return err
	// }
	// return jsonutils.Update(n, nn)
}

func (n *SNetwork) IsEmulated() bool {
	return false
}

func (n *SNetwork) GetProjectId() string {
	return ""
}

func (n *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return n.wire
}

func (n *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(n.Cidr())
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (n *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(n.Cidr())
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (n *SNetwork) Cidr() string {
	return n.Subnets[0].Cidr
}

func (n *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(n.Cidr())
	return pref.MaskLen
}

func (n *SNetwork) GetGateway() string {
	return n.Subnets[0].GatewayIp
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120
}

func (r *SRegion) GetNetworks(routerId, zoneRegion string) ([]SNetwork, error) {
	queryParams := map[string]string{
		"networkTypeEnum": "VM",
	}
	if len(routerId) > 0 {
		queryParams["routerId"] = routerId
	}
	if len(zoneRegion) > 0 {
		queryParams["region"] = zoneRegion
	}
	request := NewConsoleRequest(r.ID, "/api/v2/netcenter/network/NetworkResps", queryParams, nil)
	networks := make([]SNetwork, 0)
	err := r.client.doList(context.Background(), request, &networks)
	if err != nil {
		return nil, err
	}
	return networks, nil
}

func (r *SRegion) GetNetworkById(routerId, zoneRegion, netId string) (*SNetwork, error) {
	queryParams := map[string]string{
		"rangeInNetworkIds": netId,
		"networkTypeEnum":   "VM",
	}
	if len(routerId) > 0 {
		queryParams["routerId"] = routerId
	}
	if len(zoneRegion) > 0 {
		queryParams["region"] = zoneRegion
	}
	request := NewConsoleRequest(r.ID, "/api/v2/netcenter/network/NetworkResps", queryParams, nil)
	networks := make([]SNetwork, 0, 1)
	err := r.client.doList(context.Background(), request, &networks)
	if err != nil {
		return nil, err
	}
	if len(networks) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &networks[0], nil
}
