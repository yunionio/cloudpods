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

package ksyun

import (
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	Instance *SInstance
	Id       string
	IpAddr   string
	MacAddr  string

	Classic bool

	cloudprovider.DummyICloudNic
}

func (nic *SInstanceNic) GetId() string {
	return nic.Id
}

func (nic *SInstanceNic) GetIP() string {
	return nic.IpAddr
}

func (nic *SInstanceNic) GetMAC() string {
	if len(nic.MacAddr) > 0 {
		return nic.MacAddr
	}
	ip, _ := netutils.NewIPV4Addr(nic.GetIP())
	return ip.ToMac("00:16:")
}

func (nic *SInstanceNic) InClassicNetwork() bool {
	return nic.Classic
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) GetINetworkId() string {
	return nic.Instance.SubnetID
}

func (nic *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}

func (nic *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}
