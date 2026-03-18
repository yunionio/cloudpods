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
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	EcloudTags
	vpc  *SVpc
	zone *SZone
}

func (w *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", w.vpc.GetId(), w.zone.GetId())
}

func (w *SWire) GetName() string {
	return w.GetId()
}

func (w *SWire) GetGlobalId() string {
	return w.GetId()
}

func (w *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (w *SWire) Refresh() error {
	return nil
}

func (w *SWire) IsEmulated() bool {
	return true
}

func (w *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return w.vpc
}

func (w *SWire) GetIZone() cloudprovider.ICloudZone {
	return w.zone
}

func (w *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	networks, err := w.vpc.region.GetNetworks(w.vpc.Id, w.zone.ZoneCode)
	if err != nil {
		return nil, err
	}
	inetworks := make([]cloudprovider.ICloudNetwork, len(networks))
	for i := range networks {
		networks[i].wire = w
		inetworks[i] = &networks[i]
	}
	return inetworks, nil
}

func (w *SWire) GetBandwidth() int {
	return 10000
}

func (w *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	n, err := w.vpc.region.GetNetwork(netid)
	if err != nil {
		return nil, err
	}
	n.wire = w
	return n, nil
}

func (w *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	networkName := opts.Name
	if networkName == "" {
		networkName = "subnet-1"
	}
	// 规范：5-22 位、字母开头
	if len(networkName) < 5 {
		networkName = networkName + "xxxx"[:5-len(networkName)]
	}
	if len(networkName) > 22 {
		networkName = networkName[:22]
	}
	if c := networkName[0]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
		networkName = "n" + networkName
		if len(networkName) > 22 {
			networkName = networkName[:22]
		}
	}
	cidr := opts.Cidr
	if cidr == "" {
		cidr = "192.168.0.0/24"
	}
	regionPoolId := ""
	if w.zone != nil && w.zone.ZoneCode != "" {
		regionPoolId = regionIdToPoolId[w.zone.ZoneCode]
		if regionPoolId == "" {
			regionPoolId = w.zone.ZoneCode
		}
	}
	net, err := w.vpc.region.CreateNetwork(w.vpc.RouterId, regionPoolId, networkName, cidr)
	if err != nil {
		return nil, err
	}
	net.wire = w
	return net, nil
}
