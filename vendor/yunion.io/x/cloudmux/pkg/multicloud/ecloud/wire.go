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
	vpc       *SVpc
	zone      *SZone
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
	// TODO? w.fetchNetworks()
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
	if w.inetworks == nil {
		err := w.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return w.inetworks, nil
}

func (w *SWire) GetBandwidth() int {
	return 10000
}

func (w *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	n, err := w.vpc.region.GetNetworkById(w.vpc.RouterId, w.zone.Region, netid)
	if err != nil {
		return nil, err
	}
	n.wire = w
	return n, nil
}

func (w *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, nil
}

func (w *SWire) CreateNetworks() (*SNetwork, error) {
	// data := jsonutils.NewDict()
	// req := jsonutils.NewDict()
	// req.Set("availabilityZoneHints", jsonutils.NewString("RegionOne"))
	// req.Set("networkName", jsonutils.NewString("zyone"))
	// req.Set("networkTypeEnum", jsonutils.NewString("VM"))
	// req.Set("region", jsonutils.NewString("RegionOne"))
	// req.Set("routerId", jsonutils.NewString(w.vpc.RouterId))
	// subnet := jsonutils.NewDict()
	// subnet.Set("cidr", jsonutils.NewString("192.168.46.0/24"))
	// subnet.Set("ipVersion", jsonutils.NewString("4"))
	// subnet.Set("subnetName", jsonutils.NewString("zyone"))
	// req.Set("subnets", jsonutils.NewArray(subnet))
	// data.Set("networkCreateReq", req)
	// fmt.Printf("data:\n%s", data.PrettyString())
	// request := NewConsoleRequest(w.vpc.region.ID, "/api/v2/netcenter/network", nil, req)
	// request.SetMethod("POST")
	// resp, err := w.vpc.region.client.request(context.Background(), request)
	// if err != nil {
	// 	return nil, err
	// }
	// fmt.Printf("resp:%s", resp)
	// return nil, nil
	return nil, cloudprovider.ErrNotImplemented
}

func (w *SWire) GetNetworkById(netId string) (*SNetwork, error) {
	n, err := w.vpc.region.GetNetworkById(w.vpc.RouterId, w.zone.Region, netId)
	if err != nil {
		return nil, err
	}
	n.wire = w
	return n, nil
}

func (w *SWire) fetchNetworks() error {
	networks, err := w.vpc.region.GetNetworks(w.vpc.RouterId, w.zone.Region)
	if err != nil {
		return err
	}
	inetworks := make([]cloudprovider.ICloudNetwork, len(networks))
	for i := range networks {
		networks[i].wire = w
		inetworks[i] = &networks[i]
	}
	w.inetworks = inetworks
	return nil
}
