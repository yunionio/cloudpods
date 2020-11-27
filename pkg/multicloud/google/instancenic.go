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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	return ip.ToMac("00:16:")
}

func (nic *SNetworkInterface) GetDriver() string {
	return "virtio"
}

func (nic *SNetworkInterface) InClassicNetwork() bool {
	return false
}

func (nic *SNetworkInterface) GetINetwork() cloudprovider.ICloudNetwork {
	network, err := nic.instance.host.zone.region.GetNetwork(nic.Subnetwork)
	if err != nil {
		log.Errorf("failed to found network(%s) for nic error: %v", nic.Subnetwork, err)
		return nil
	}
	wire := nic.instance.host.GetWire()
	network.wire = wire
	return network
}
