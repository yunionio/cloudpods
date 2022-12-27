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

package remotefile

import (
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SEip struct {
	SResourceBase

	IpAddr      string
	RegionId    string
	AssociateId string
	Mode        string
	Bandwidth   int
}

func (self *SEip) GetIpAddr() string {
	return self.IpAddr
}

func (self *SEip) GetMode() string {
	if utils.IsInStringArray(self.Mode, []string{api.EIP_MODE_STANDALONE_EIP, api.EIP_MODE_INSTANCE_PUBLICIP}) {
		return self.Mode
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetINetworkId() string {
	return ""
}

func (self *SEip) GetAssociationType() string {
	return api.EIP_ASSOCIATE_TYPE_SERVER
}

func (self *SEip) GetAssociationExternalId() string {
	return self.AssociateId
}

func (self *SEip) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SEip) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEip) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SEip) Dissociate() error {
	return cloudprovider.ErrNotSupported
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}
