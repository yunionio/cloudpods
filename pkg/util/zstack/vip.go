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
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
)

type SVirtualIP struct {
	ZStackBasic
	IPRangeUUID        string   `json:"ipRangeUuid"`
	L3NetworkUUID      string   `json:"l3NetworkUuid"`
	IP                 string   `json:"ip"`
	State              string   `json:"state"`
	Gateway            string   `json:"gateway"`
	Netmask            string   `json:"netmask"`
	PrefixLen          int      `json:"prefixLen"`
	ServiceProvider    string   `json:"serviceProvider"`
	PeerL3NetworkUuids []string `json:"peerL3NetworkUuids"`
	UseFor             string   `json:"useFor"`
	UsedIPUUID         string   `json:"usedIpUuid"`
	ZStackTime
}

func (region *SRegion) GetVirtualIP(vipId string) (*SVirtualIP, error) {
	vip := &SVirtualIP{}
	return vip, region.client.getResource("vips", vipId, vip)
}

func (region *SRegion) GetNetworkId(vip *SVirtualIP) string {
	networks, err := region.GetNetworks("", "", vip.L3NetworkUUID, "")
	if err == nil {
		for _, network := range networks {
			if network.Contains(vip.IP) {
				return fmt.Sprintf("%s/%s", vip.L3NetworkUUID, network.UUID)
			}
		}
	}
	return ""
}

func (region *SRegion) GetVirtualIPs(vipId string) ([]SVirtualIP, error) {
	vips := []SVirtualIP{}
	params := url.Values{}
	if len(vipId) > 0 {
		params.Add("q", "uuid="+vipId)
	}
	return vips, region.client.listAll("vips", params, &vips)
}

func (region *SRegion) CreateVirtualIP(name, desc, ip string, l3Id string) (*SVirtualIP, error) {
	vip := SVirtualIP{}
	params := map[string]map[string]string{
		"params": {
			"name":          name,
			"description":   desc,
			"l3NetworkUuid": l3Id,
		},
	}
	if len(ip) > 0 {
		params["params"]["requiredIp"] = ip
	}
	resp, err := region.client.post("vips", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	return &vip, resp.Unmarshal(&vip, "inventory")
}

func (region *SRegion) DeleteVirtualIP(vipId string) error {
	return region.client.delete("vips", vipId, "")
}
