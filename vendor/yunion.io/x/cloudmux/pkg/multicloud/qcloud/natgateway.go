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

package qcloud

import (
	"fmt"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNatGateway struct {
	multicloud.SNatGatewayBase
	QcloudTags
	vpc *SVpc

	State          string
	VpcId          string
	Zone           string
	NatGatewayName string
	NatGatewayId   string

	Bandwidth     string    `json:"bandwidth"`
	CreateTime    time.Time `json:"createTime"`
	EipCount      string    `json:"eipCount"`
	MaxConcurrent float32   `json:"maxConcurrent"`

	PublicIpAddressSet []struct {
		AddressId string
	}
	SourceIpTranslationNatRuleSet          []SSTable
	DestinationIpPortTranslationNatRuleSet []SDTable
}

func (nat *SNatGateway) GetName() string {
	if len(nat.NatGatewayName) > 0 {
		return nat.NatGatewayName
	}
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetId() string {
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetGlobalId() string {
	return nat.NatGatewayId
}

func (self *SNatGateway) GetINetworkId() string {
	return ""
}

func (self *SNatGateway) GetNetworkType() string {
	return api.NAT_NETWORK_TYPE_INTERNET
}

func (nat *SNatGateway) GetStatus() string {
	switch nat.State {
	case "PENDING":
		return api.NAT_STATUS_ALLOCATE
	case "AVAILABLE":
		return api.NAT_STAUTS_AVAILABLE
	case "UPDATING":
		return api.NAT_STATUS_DEPLOYING
	case "DELETING":
		return api.NAT_STATUS_DELETING
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (nat *SNatGateway) GetNatSpec() string {
	switch int(nat.MaxConcurrent) {
	case 100 * 10000:
		return api.QCLOUD_NAT_SPEC_SMALL
	case 300 * 10000:
		return api.QCLOUD_NAT_SPEC_MIDDLE
	case 1000 * 10000:
		return api.QCLOUD_NAT_SPEC_LARGE
	}
	return ""
}

func (nat *SNatGateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips := []SEipAddress{}
	for {
		part, total, err := nat.vpc.region.GetEips("", nat.GetId(), len(eips), 50)
		if err != nil {
			return nil, err
		}
		eips = append(eips, part...)
		if len(eips) >= total {
			break
		}
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i++ {
		eips[i].region = nat.vpc.region
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (nat *SNatGateway) GetINatSTable() ([]cloudprovider.ICloudNatSEntry, error) {
	ret := []cloudprovider.ICloudNatSEntry{}
	for i := 0; i < len(nat.SourceIpTranslationNatRuleSet); i++ {
		nat.SourceIpTranslationNatRuleSet[i].nat = nat
		ret = append(ret, &nat.SourceIpTranslationNatRuleSet[i])
	}
	return ret, nil
}

func (nat *SNatGateway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
	ret := []cloudprovider.ICloudNatDEntry{}
	for i := 0; i < len(nat.DestinationIpPortTranslationNatRuleSet); i++ {
		nat.DestinationIpPortTranslationNatRuleSet[i].nat = nat
		ret = append(ret, &nat.DestinationIpPortTranslationNatRuleSet[i])
	}
	return ret, nil
}

func (nat *SNatGateway) GetINatDEntryById(id string) (cloudprovider.ICloudNatDEntry, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (nat *SNatGateway) GetINatSEntryById(id string) (cloudprovider.ICloudNatSEntry, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (nat *SNatGateway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (nat *SNatGateway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetNatGateways(vpcId string, offset int, limit int) ([]SNatGateway, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	if len(vpcId) > 0 {
		params["Filters.0.Name"] = "vpc-id"
		params["Filters.0.Values.0"] = vpcId
	}
	body, err := region.vpcRequest("DescribeNatGateways", params)
	if err != nil {
		return nil, 0, err
	}
	nats := []SNatGateway{}
	err = body.Unmarshal(&nats, "NatGatewaySet")
	if err != nil {
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return nats, int(total), nil
}
