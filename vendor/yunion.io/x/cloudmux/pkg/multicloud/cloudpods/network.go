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

package cloudpods

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/rbacutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SResourceBase
	CloudpodsTags
	wire *SWire

	api.NetworkDetails
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetId() string {
	return self.Id
}

func (self *SNetwork) GetGlobalId() string {
	return self.Id
}

func (self *SNetwork) GetStatus() string {
	return self.Status
}

func (self *SNetwork) Delete() error {
	return self.wire.vpc.region.cli.delete(&modules.Networks, self.Id)
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	return self.GuestIpStart
}

func (self *SNetwork) GetIpEnd() string {
	return self.GuestIpEnd
}

func (self *SNetwork) GetIpMask() int8 {
	return int8(self.GuestIpMask)
}

func (self *SNetwork) GetGateway() string {
	return self.GuestGateway
}

func (self *SNetwork) GetServerType() string {
	return self.ServerType
}

func (self *SNetwork) GetPublicScope() rbacutils.TRbacScope {
	return rbacutils.TRbacScope(self.PublicScope)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return self.AllocTimoutSeconds
}

func (self *SNetwork) GetProjectId() string {
	return self.TenantId
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := self.vpc.region.GetNetworks(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range networks {
		networks[i].wire = self
		ret = append(ret, &networks[i])
	}
	return ret, nil
}

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	net, err := self.vpc.region.GetNetwork(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetwork(%s)", id)
	}
	net.wire = self
	return net, nil
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	input := api.NetworkCreateInput{}
	input.Name = opts.Name
	input.Description = opts.Desc
	input.GuestIpPrefix = opts.Cidr
	input.WireId = self.Id
	input.ProjectId = opts.ProjectId
	network := &SNetwork{wire: self}
	return network, self.vpc.region.create(&modules.Networks, input, network)
}

func (self *SRegion) GetNetworks(wireId string) ([]SNetwork, error) {
	networks := []SNetwork{}
	params := map[string]interface{}{}
	if len(wireId) > 0 {
		params["wire_id"] = wireId
	}
	return networks, self.list(&modules.Networks, params, &networks)
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	network := &SNetwork{}
	return network, self.cli.get(&modules.Networks, id, nil, network)
}
