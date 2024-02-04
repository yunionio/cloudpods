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

package huawei

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

/*
Subnets
*/

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090590.html
type SNetwork struct {
	multicloud.SNetworkBase
	HuaweiTags
	wire *SWire

	AvailabilityZone string   `json:"availability_zone"`
	CIDR             string   `json:"cidr"`
	CIDRV6           string   `json:"cidr_v6"`
	DHCPEnable       bool     `json:"dhcp_enable"`
	DNSList          []string `json:"dnsList"`
	GatewayIP        string   `json:"gateway_ip"`
	GatewayIPv6      string   `json:"gateway_ip_v6"`
	ID               string   `json:"id"`
	Ipv6Enable       bool     `json:"ipv6_enable"`
	Name             string   `json:"name"`
	NeutronNetworkID string   `json:"neutron_network_id"`
	NeutronSubnetID  string   `json:"neutron_subnet_id"`
	PrimaryDNS       string   `json:"primary_dns"`
	SecondaryDNS     string   `json:"secondary_dns"`
	Status           string   `json:"status"`
	VpcID            string   `json:"vpc_id"`
}

func (self *SNetwork) GetId() string {
	return self.ID
}

func (self *SNetwork) GetName() string {
	if len(self.Name) == 0 {
		return self.ID
	}

	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return self.ID
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090591.html
func (self *SNetwork) GetStatus() string {
	switch self.Status {
	case "ACTIVE", "UNKNOWN":
		return api.NETWORK_STATUS_AVAILABLE // ? todo: // UNKNOWN
	case "ERROR":
		return api.NETWORK_STATUS_UNKNOWN
	default:
		return api.NETWORK_STATUS_UNKNOWN
	}
}

func (self *SNetwork) Refresh() error {
	net, err := self.wire.vpc.region.GetNetwork(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, net)
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetTags() (map[string]string, error) {
	res := fmt.Sprintf("subnets/%s/tags", self.ID)
	resp, err := self.wire.vpc.region.list(SERVICE_VPC_V2_0, res, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "list tags")
	}
	ret := []struct {
		Key   string
		Value string
	}{}
	err = resp.Unmarshal(&ret, "tags")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	result := map[string]string{}
	for _, tag := range ret {
		result[tag.Key] = tag.Value
	}
	return result, nil

}

func (self *SNetwork) SetTags(tags map[string]string, replace bool) error {
	existedTags, err := self.GetTags()
	if err != nil {
		return errors.Wrapf(err, "GetTags")
	}
	return self.wire.vpc.region.SetNetworkTags(self.ID, existedTags, tags, replace)
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=DeleteSubnetTag
func (self *SRegion) DeleteNetworkTag(subnetId string, key string) error {
	res := fmt.Sprintf("subnets/%s/tags/%s", subnetId, key)
	_, err := self.delete(SERVICE_VPC_V2_0, res)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=CreateSubnetTag
func (self *SRegion) CreateNetworkTag(subnetId string, tags map[string]string) error {
	params := map[string]interface{}{
		"action": "create",
	}
	add := []map[string]string{}
	for k, v := range tags {
		add = append(add, map[string]string{"key": k, "value": v})
	}
	params["tags"] = add
	res := fmt.Sprintf("subnets/%s/tags/action", subnetId)
	_, err := self.post(SERVICE_VPC_V2_0, res, params)
	return err
}

func (self *SRegion) SetNetworkTags(netId string, existedTags map[string]string, tags map[string]string, replace bool) error {
	deleteTagsKey := []string{}
	for k := range existedTags {
		if replace {
			deleteTagsKey = append(deleteTagsKey, k)
		} else {
			if _, ok := tags[k]; ok {
				deleteTagsKey = append(deleteTagsKey, k)
			}
		}
	}
	if len(deleteTagsKey) > 0 {
		for _, k := range deleteTagsKey {
			err := self.DeleteNetworkTag(netId, k)
			if err != nil {
				return errors.Wrapf(err, "remove tags")
			}
		}
	}
	if len(tags) > 0 {
		err := self.CreateNetworkTag(netId, tags)
		if err != nil {
			return errors.Wrapf(err, "add tags")
		}
	}
	return nil
}

func (net *SNetwork) GetIp6Start() string {
	if len(net.CIDRV6) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.CIDRV6)
		if err != nil {
			return ""
		}
		return prefix.Address.NetAddr(prefix.MaskLen).StepUp().StepUp().String()
	}
	return ""
}

func (net *SNetwork) GetIp6End() string {
	if len(net.CIDRV6) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.CIDRV6)
		if err != nil {
			return ""
		}
		end := prefix.Address.NetAddr(prefix.MaskLen).BroadcastAddr(prefix.MaskLen)
		return end.StepDown().StepDown().StepDown().StepDown().StepDown().String()
	}
	return ""
}

func (net *SNetwork) GetIp6Mask() uint8 {
	if len(net.CIDRV6) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.CIDRV6)
		if err != nil {
			return 0
		}
		return prefix.MaskLen
	}
	return 0
}

func (net *SNetwork) GetGateway6() string {
	return net.GatewayIPv6
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return self.wire.vpc.region.deleteNetwork(self.VpcID, self.GetId())
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ShowSubnet
func (self *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	resp, err := self.list(SERVICE_VPC, "subnets/"+networkId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "show subnet")
	}
	network := &SNetwork{}
	err = resp.Unmarshal(&network, "subnet")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return network, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=ListSubnets
func (self *SRegion) GetNetworks(vpcId string) ([]SNetwork, error) {
	ret := []SNetwork{}
	query := url.Values{}
	if len(vpcId) > 0 {
		query.Set("vpc_id", vpcId)
	}
	for {
		resp, err := self.list(SERVICE_VPC, "subnets", query)
		if err != nil {
			return nil, err
		}
		part := []SNetwork{}
		err = resp.Unmarshal(&part, "subnets")
		if err != nil {
			return nil, err
		}
		ret = append(ret, part...)
		if len(part) == 0 {
			break
		}
		query.Set("marker", part[len(part)-1].ID)
	}
	return ret, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/VPC/doc?version=v2&api=DeleteSubnet
func (self *SRegion) deleteNetwork(vpcId string, networkId string) error {
	res := fmt.Sprintf("vpcs/%s/subnets/%s", vpcId, networkId)
	_, err := self.delete(SERVICE_VPC, res)
	return err
}

func (self *SNetwork) GetProjectId() string {
	return self.wire.vpc.EnterpriseProjectID
}
