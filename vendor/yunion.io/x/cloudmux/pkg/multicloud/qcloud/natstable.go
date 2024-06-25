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

package qcloud

import (
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSTable struct {
	multicloud.SResourceBase
	QcloudTags
	nat *SNatGateway

	CreatedTime       string
	Description       string
	NatGatewayId      string
	NatGatewaySnatId  string
	PrivateIpAddress  string
	PublicIpAddresses []string
	ResourceId        string
	ResourceType      string
	VpcId             string
}

func (stable *SSTable) GetName() string {
	return stable.NatGatewaySnatId
}

func (stable *SSTable) GetId() string {
	return stable.NatGatewaySnatId
}

func (stable *SSTable) GetGlobalId() string {
	return stable.NatGatewaySnatId
}

func (stable *SSTable) GetStatus() string {
	return api.NAT_STAUTS_AVAILABLE
}

func (stable *SSTable) GetIP() string {
	return strings.Join(stable.PublicIpAddresses, ",")
}

func (stable *SSTable) GetSourceCIDR() string {
	return stable.PrivateIpAddress
}

func (stable *SSTable) GetNetworkId() string {
	return stable.ResourceId
}

func (stable *SSTable) Delete() error {
	return cloudprovider.ErrNotImplemented
}
