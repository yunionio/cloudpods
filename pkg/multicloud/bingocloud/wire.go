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

package bingocloud

import (
	"fmt"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SWire struct {
	multicloud.STagBase
	multicloud.SResourceBase

	cluster *SCluster
	vpc     *SVpc
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s/%s", self.vpc.GetGlobalId(), self.cluster.GetGlobalId())
}

func (self *SWire) GetGlobalId() string {
	return self.GetId()
}

func (self *SWire) GetName() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetName(), self.cluster.GetName())
}

func (self *SWire) GetBandwidth() int {
	return 1000
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.cluster
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wires, err := self.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range wires {
		if wires[i].GetGlobalId() == id {
			return wires[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	clusters, err := self.region.GetClusters()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range clusters {
		wire := &SWire{
			vpc:     self,
			cluster: &clusters[i],
		}
		ret = append(ret, wire)
	}
	return ret, nil
}

func (self *SWire) getNetworkById(networkId string) *SNetwork {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil
	}
	log.Debugf("search for networks %d", len(networks))
	for i := 0; i < len(networks); i += 1 {
		log.Debugf("search %s", networks[i].GetName())
		network := networks[i].(*SNetwork)
		if network.SubnetId == networkId {
			return network
		}
	}
	return nil
}
