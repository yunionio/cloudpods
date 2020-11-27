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

package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance

	UUID           string   `json:"uuid"`
	VMInstanceUUID string   `json:"vmInstanceUuid"`
	L3NetworkUUID  string   `json:"l3NetworkUuid"`
	IP             string   `json:"ip"`
	Mac            string   `json:"mac"`
	HypervisorType string   `json:"hypervisorType"`
	IPVersion      int      `json:"ipVersion"`
	UsedIps        []string `json:"usedIps"`
	InternalName   string   `json:"internalName"`
	DeviceID       int      `json:"deviceId"`
	ZStackTime

	cloudprovider.DummyICloudNic
}

func (nic *SInstanceNic) GetId() string {
	return ""
}

func (nic *SInstanceNic) GetIP() string {
	return nic.IP
}

func (nic *SInstanceNic) GetMAC() string {
	return nic.Mac
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (nic *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	networks, err := nic.instance.host.zone.region.GetNetworks(nic.instance.host.zone.UUID, "", nic.L3NetworkUUID, "")
	if err != nil {
		log.Errorf("failed to found networks for nic %v error: %v", jsonutils.Marshal(nic).String(), err)
		return nil
	}
	for i := 0; i < len(networks); i++ {
		if networks[i].Contains(nic.IP) {
			l3Network, err := nic.instance.host.zone.region.GetL3Network(networks[i].L3NetworkUUID)
			if err != nil {
				log.Errorf("failed to found l3Network for network %v error: %v", jsonutils.Marshal(networks[i]).String(), err)
				return nil
			}
			wire, err := nic.instance.host.zone.region.GetWire(l3Network.L2NetworkUUID)
			if err != nil {
				log.Errorf("failed to found wire for l3Network %v error: %v", jsonutils.Marshal(l3Network).String(), err)
				return nil
			}
			networks[i].wire = wire
			return &networks[i]
		}
	}
	return nil
}
