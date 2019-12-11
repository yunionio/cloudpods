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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SVirtualIP struct {
	multicloud.SNetworkInterfaceBase
	region *SRegion
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

type SInterfaceIP struct {
	IP            string
	L3NetworkUUID string
	IPRangeUUID   string
}

func (ip *SInterfaceIP) GetIP() string {
	return ip.IP
}

func (ip *SInterfaceIP) GetINetworkId() string {
	return fmt.Sprintf("%s/%s", ip.L3NetworkUUID, ip.IPRangeUUID)
}

func (ip *SInterfaceIP) IsPrimary() bool {
	return true
}

func (ip *SInterfaceIP) GetGlobalId() string {
	return ip.IP
}

func (vip *SVirtualIP) GetName() string {
	if len(vip.Name) > 0 {
		return vip.Name
	}
	return vip.UUID
}

func (vip *SVirtualIP) GetId() string {
	return vip.UUID
}

func (vip *SVirtualIP) GetGlobalId() string {
	return vip.UUID
}

func (vip *SVirtualIP) GetMacAddress() string {
	ip, _ := netutils.NewIPV4Addr(vip.IP)
	return ip.ToMac("00:16:")
}

func (vip *SVirtualIP) GetAssociateType() string {
	switch vip.UseFor {
	case "LoadBalancer":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_LOADBALANCER
	case "Eip":
		return api.NETWORK_INTERFACE_ASSOCIATE_TYPE_RESERVED
	}
	return vip.UseFor
}

func (vip *SVirtualIP) GetAssociateId() string {
	return vip.UsedIPUUID
}

func (vip *SVirtualIP) GetStatus() string {
	if vip.State == "Enabled" {
		return api.NETWORK_INTERFACE_STATUS_AVAILABLE
	}
	return api.NETWORK_INTERFACE_STATUS_UNKNOWN
}

func (vip *SVirtualIP) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	ip := &SInterfaceIP{IP: vip.IP, IPRangeUUID: vip.IPRangeUUID, L3NetworkUUID: vip.L3NetworkUUID}
	return []cloudprovider.ICloudInterfaceAddress{ip}, nil
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

func (region *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	vips, err := region.GetVirtualIPs("")
	if err != nil {
		return nil, errors.Wrap(err, "region.GetVirtualIPs")
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := 0; i < len(vips); i++ {
		if vips[i].UseFor != "Eip" {
			vips[i].region = region
			ret = append(ret, &vips[i])
		}
	}
	return ret, nil
}
