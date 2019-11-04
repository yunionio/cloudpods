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

package google

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SNetwork struct {
	wire *SWire

	Id                    string
	CreationTimestamp     time.Time
	Name                  string
	Network               string
	IpCidrRange           string
	Region                string
	GatewayAddress        string
	SelfLink              string
	Status                string
	AvailableCpuPlatforms []string
	PrivateIpGoogleAccess bool
	Fingerprint           string
	Purpose               string
	Kind                  string
}

func (region *SRegion) GetNetworks(network string, maxResults int, pageToken string) ([]SNetwork, error) {
	networks := []SNetwork{}
	params := map[string]string{}
	if len(network) > 0 {
		params["filter"] = fmt.Sprintf(`network="%s"`, network)
	}
	resource := fmt.Sprintf("regions/%s/subnetworks", region.Name)
	return networks, region.List(resource, params, maxResults, pageToken, &networks)
}

func (region *SRegion) GetNetwork(id string) (*SNetwork, error) {
	network := &SNetwork{}
	return network, region.Get(id, network)
}

func (network *SNetwork) GetId() string {
	return network.SelfLink
}

func (network *SNetwork) GetName() string {
	return network.Name
}

func (network *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (network *SNetwork) GetProjectId() string {
	return network.wire.vpc.region.GetProjectId()
}

func (network *SNetwork) Refresh() error {
	_network, err := network.wire.vpc.region.GetNetwork(network.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(network, _network)
}

func (network *SNetwork) IsEmulated() bool {
	return false
}

func (network *SNetwork) GetStatus() string {
	return api.NETWORK_INTERFACE_STATUS_AVAILABLE
}

func (network *SNetwork) GetGlobalId() string {
	return getGlobalId(network.SelfLink)
}

func (network *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (network *SNetwork) GetAllocTimeoutSeconds() int {
	return 300
}

func (network *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return network.wire
}

func (network *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(network.IpCidrRange)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (network *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(network.IpCidrRange)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (network *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(network.IpCidrRange)
	return pref.MaskLen
}

func (network *SNetwork) GetGateway() string {
	return network.GatewayAddress
}

func (network *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (network *SNetwork) GetIsPublic() bool {
	return true
}

func (network *SNetwork) GetPublicScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}
