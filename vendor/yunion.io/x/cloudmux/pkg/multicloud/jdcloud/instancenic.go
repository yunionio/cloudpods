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

package jdcloud

import (
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/models"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	cloudprovider.DummyICloudNic

	models.InstanceNetworkInterface
}

func (in *SInstanceNic) GetIP() string {
	return in.PrimaryIp.PrivateIpAddress
}

func (in *SInstanceNic) GetMAC() string {
	return in.MacAddress
}

func (in *SInstanceNic) GetId() string {
	return in.NetworkInterfaceId
}

func (in *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (in *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (in *SInstanceNic) GetINetworkId() string {
	return in.SubnetId
}
