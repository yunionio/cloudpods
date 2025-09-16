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
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SNetworkInterface struct {
	instance *SInstance

	Network       string
	Subnetwork    string
	NetworkIP     string
	Name          string
	AccessConfigs []AccessConfig
	Fingerprint   string
	Kind          string

	cloudprovider.DummyICloudNic
}

func (nic *SNetworkInterface) GetId() string {
	return ""
}

func (nic *SNetworkInterface) GetIP() string {
	return nic.NetworkIP
}

func (nic *SNetworkInterface) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(nic.NetworkIP)
	return ip.ToMac("42:01:")
}

func (nic *SNetworkInterface) GetDriver() string {
	return "virtio"
}

func (nic *SNetworkInterface) InClassicNetwork() bool {
	return false
}

func (nic *SNetworkInterface) GetINetworkId() string {
	vpc := &SVpc{region: nic.instance.host.zone.region}
	err := nic.instance.host.zone.region.GetBySelfId(nic.Subnetwork, vpc)
	if err != nil {
		return ""
	}
	return vpc.Id
}
