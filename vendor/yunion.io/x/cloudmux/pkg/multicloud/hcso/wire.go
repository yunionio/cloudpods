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

package hcso

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

// 华为云的子网有点特殊。子网在整个region可用。
type SWire struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
	region *SRegion
	vpc    *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.region.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.region.GetGlobalId())
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if self.inetworks == nil {
		err := self.vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.inetworks, nil
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
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

/*
华为云子网可用区，类似一个zone标签。即使指定了zone子网在整个region依然是可用。
通过华为web控制台创建子网需要指定可用区。这里是不指定的。
*/
func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	networkId, err := self.region.createNetwork(self.vpc.GetId(), opts.Name, opts.Cidr, opts.Desc)
	if err != nil {
		log.Errorf("createNetwork error %s", err)
		return nil, err
	}

	var network *SNetwork
	err = cloudprovider.WaitCreated(5*time.Second, 60*time.Second, func() bool {
		self.inetworks = nil
		network = self.getNetworkById(networkId)
		if network == nil {
			return false
		} else {
			return true
		}
	})

	if err != nil {
		log.Errorf("cannot find network after create????")
		return nil, err
	}

	network.wire = self
	return network, nil
}

func (self *SWire) addNetwork(network *SNetwork) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetId() == network.ID {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, network)
	}
}

func (self *SWire) getNetworkById(networkId string) *SNetwork {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil
	}
	log.Debugf("search for networks %d", len(networks))
	for i := 0; i < len(networks); i += 1 {
		log.Debugf("search %s", networks[i].GetName())
		network := networks[i]
		if network.GetId() == networkId {
			return network.(*SNetwork)
		}
	}
	return nil
}

func getDefaultGateWay(cidr string) (string, error) {
	pref, err := netutils.NewIPV4Prefix(cidr)
	if err != nil {
		return "", errors.Wrap(err, "getDefaultGateWay.NewIPV4Prefix")
	}
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String(), nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090590.html
// cidr 掩码长度不能大于28
func (self *SRegion) createNetwork(vpcId string, name string, cidr string, desc string) (string, error) {
	gateway, err := getDefaultGateWay(cidr)
	if err != nil {
		return "", err
	}

	params := jsonutils.NewDict()
	subnetObj := jsonutils.NewDict()
	subnetObj.Add(jsonutils.NewString(name), "name")
	subnetObj.Add(jsonutils.NewString(vpcId), "vpc_id")
	subnetObj.Add(jsonutils.NewString(cidr), "cidr")
	subnetObj.Add(jsonutils.NewString(gateway), "gateway_ip")
	// hard code for hcso
	// https://support.huaweicloud.com/dns_faq/dns_faq_002.html
	// https://support.huaweicloud.com/api-dns/dns_api_69001.html
	if self.client != nil && len(self.client.endpoints.DefaultSubnetDns) > 0 {
		dns := strings.Split(self.client.endpoints.DefaultSubnetDns, ",")
		if len(dns) > 0 && len(dns[0]) > 0 {
			subnetObj.Add(jsonutils.NewString(dns[0]), "primary_dns")
		}

		if len(dns) > 1 && len(dns[1]) > 0 {
			subnetObj.Add(jsonutils.NewString(dns[1]), "secondary_dns")
		}
	}
	params.Add(subnetObj, "subnet")

	subnet := SNetwork{}
	err = DoCreate(self.ecsClient.Subnets.Create, params, &subnet)
	return subnet.ID, err
}
