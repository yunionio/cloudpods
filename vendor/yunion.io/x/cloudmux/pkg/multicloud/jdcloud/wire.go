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
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	JdcloudTags

	vpc       *SVpc
	inetworks []cloudprovider.ICloudNetwork
}

func (w *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", w.vpc.GetId(), w.vpc.region.GetId())
}

func (w *SWire) GetName() string {
	return w.GetId()
}

func (w *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", w.vpc.GetGlobalId(), w.vpc.region.GetGlobalId())
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
	return nil
}

func (w *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if w.inetworks == nil {
		nets, err := w.networks()
		if err != nil {
			return nil, err
		}
		w.inetworks = make([]cloudprovider.ICloudNetwork, 0, len(nets))
		for i := range nets {
			nets[i].wire = w
			w.inetworks = append(w.inetworks, &nets[i])
		}
	}
	return w.inetworks, nil
}

func (w *SWire) networks() ([]SNetwork, error) {
	nets := make([]SNetwork, 0)
	n := 1
	for {
		parts, total, err := w.vpc.region.GetNetworks(w.vpc.VpcId, n, 100)
		if err != nil {
			return nil, err
		}
		nets = append(nets, parts...)
		if len(nets) >= total {
			break
		}
		n++
	}
	return nets, nil
}

func (w *SWire) GetBandwidth() int {
	return 10000
}

func (w *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := w.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(networks); i += 1 {
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (w *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (w *SWire) CreateNetworks() (*SNetwork, error) {
	return nil, cloudprovider.ErrNotImplemented
}
