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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SIsolatedDevice struct {
	api.IsolateDeviceDetails
}

func (d *SIsolatedDevice) GetName() string {
	return d.Name
}

func (d *SIsolatedDevice) GetGlobalId() string {
	return d.Id
}

func (d *SIsolatedDevice) GetModel() string {
	return d.Model
}

func (d *SIsolatedDevice) GetAddr() string {
	return d.Addr
}

func (d *SIsolatedDevice) GetDevType() string {
	return d.DevType
}

func (d *SIsolatedDevice) GetNumaNode() int8 {
	return int8(d.NumaNode)
}

func (d *SIsolatedDevice) GetVendorDeviceId() string {
	return d.VendorDeviceId
}

func (region *SRegion) GetIsolatedDevices(hostId string, serverId string) ([]SIsolatedDevice, error) {
	params := map[string]interface{}{}
	if len(hostId) > 0 {
		params["host_id"] = hostId
	}
	if len(serverId) > 0 {
		params["guest_id"] = serverId
	}
	ret := []SIsolatedDevice{}
	err := region.list(&modules.IsolatedDevices, params, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetIsolatedDevice(id string) (*SIsolatedDevice, error) {
	ret := &SIsolatedDevice{}
	err := region.cli.get(&modules.IsolatedDevices, id, nil, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
