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

package ctyun

type ServerSku struct {
	GpuVendor     string  `json:"gpuVendor"`
	CPUInfo       string  `json:"cpuInfo"`
	BaseBandwidth float64 `json:"baseBandwidth"`
	FlavorName    string  `json:"flavorName"`
	VideoMemSize  int     `json:"videoMemSize"`
	FlavorType    string  `json:"flavorType"`
	FlavorSeries  string  `json:"flavorSeries"`
	FlavorRAM     int     `json:"flavorRAM"`
	NicMultiQueue int     `json:"nicMultiQueue"`
	Pps           string  `json:"pps"`
	FlavorCPU     int     `json:"flavorCPU"`
	Bandwidth     int     `json:"bandwidth"`
	GpuType       string  `json:"gpuType"`
	FlavorId      string  `json:"flavorID"`
	GpuCount      int     `json:"gpuCount"`
}

func (self *SRegion) GetServerSkus(zoneId string) ([]ServerSku, error) {
	params := map[string]interface{}{}
	if zoneId != "default" {
		params["azName"] = zoneId
	}
	resp, err := self.post(SERVICE_ECS, "/v4/ecs/flavor/list", params)
	if err != nil {
		return nil, err
	}
	ret := struct {
		ReturnObj struct {
			FlavorList []ServerSku
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret.ReturnObj.FlavorList, nil
}
