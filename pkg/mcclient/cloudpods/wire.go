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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SWire struct {
	CloudpodsTags
	multicloud.SResourceBase

	vpc *SVpc

	api.WireDetails
}

func (self *SWire) GetId() string {
	return self.Id
}

func (self *SWire) GetGlobalId() string {
	return self.Id
}

func (self *SWire) GetName() string {
	return self.Name
}

func (self *SWire) GetStatus() string {
	return self.Status
}

func (self *SWire) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SWire) IsEmulated() bool {
	return self.SWire.IsEmulated
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	if len(self.ZoneId) == 0 {
		return nil
	}
	zone, err := self.vpc.region.GetZone(self.ZoneId)
	if err != nil {
		return nil
	}
	return zone
}

func (self *SWire) Refresh() error {
	wire, err := self.vpc.region.GetWire(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, wire)
}

func (self *SVpc) GetIWireById(id string) (cloudprovider.ICloudWire, error) {
	wire, err := self.region.GetWire(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetWire(%s)", id)
	}
	wire.vpc = self
	return wire, nil
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := self.region.GetWires(self.Id, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetWires")
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		wires[i].vpc = self
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (self *SRegion) GetWires(vpcId, hostId string) ([]SWire, error) {
	wires := []SWire{}
	params := map[string]interface{}{"cloud_env": ""}
	if len(vpcId) > 0 {
		params["vpc_id"] = vpcId
	}
	if len(hostId) > 0 {
		params["host_id"] = hostId
	}
	return wires, self.list(&modules.Wires, params, &wires)
}

func (self *SRegion) GetWire(id string) (*SWire, error) {
	wire := &SWire{}
	return wire, self.cli.get(&modules.Wires, id, nil, wire)
}
