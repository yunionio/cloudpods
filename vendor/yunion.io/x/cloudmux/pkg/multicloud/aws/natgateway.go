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

package aws

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type NatGatewayAddress struct {
	AllocationId       string `xml:"allocationId"`
	NetworkInterfaceId string `xml:"networkInterfaceId"`
	PrivateIp          string `xml:"privateIp"`
	PublicIp           string `xml:"publicIp"`
}

type ProvisionedBandwidth struct {
	ProvisionTime time.Time `xml:"provisionTime"`
	Provisioned   string    `xml:"provisioned"`
	RequestTime   time.Time `xml:"requestTime"`
	Requested     string    `xml:"requested"`
	Status        string    `xml:"status"`
}

type SNatGateway struct {
	multicloud.SNatGatewayBase
	AwsTags

	region *SRegion

	ConnectivityType     string               `xml:"connectivityType"`
	CreateTime           time.Time            `xml:"createTime"`
	DeleteTime           time.Time            `xml:"deleteTime"`
	FailureCode          string               `xml:"failureCode"`
	FailureMessage       string               `xml:"failureMessage"`
	NatGatewayAddresses  []NatGatewayAddress  `xml:"natGatewayAddressSet>item"`
	NatGatewayId         string               `xml:"natGatewayId"`
	ProvisionedBandwidth ProvisionedBandwidth `xml:"provisionedBandwidth"`
	// pending | failed | available | deleting | deleted
	State    string `xml:"state"`
	SubnetId string `xml:"subnetId"`
	VpcId    string `xml:"vpcId"`
}

func (self *SNatGateway) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.NatGatewayId
}

func (self *SNatGateway) GetId() string {
	return self.NatGatewayId
}

func (self *SNatGateway) GetGlobalId() string {
	return self.NatGatewayId
}

func (self *SNatGateway) GetStatus() string {
	switch self.State {
	case "pending":
		return api.NAT_STATUS_ALLOCATE
	case "failed":
		return api.NAT_STATUS_CREATE_FAILED
	case "available":
		return api.NAT_STAUTS_AVAILABLE
	case "deleting", "deleted":
		return api.NAT_STATUS_DELETING
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (self *SNatGateway) GetNetworkType() string {
	if self.ConnectivityType == "public" {
		return api.NAT_NETWORK_TYPE_INTERNET
	}
	return api.NAT_NETWORK_TYPE_INTRANET
}

func (self *SNatGateway) GetNatSpec() string {
	return ""
}

func (self *SNatGateway) Refresh() error {
	nat, err := self.region.GetNatGateway(self.NatGatewayId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, nat)
}

func (self *SNatGateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.region.GetEips("", "", self.NatGatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetEIPs")
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = self.region
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (self *SNatGateway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
	return []cloudprovider.ICloudNatDEntry{}, nil
}

func (self *SNatGateway) GetINatSTable() ([]cloudprovider.ICloudNatSEntry, error) {
	return []cloudprovider.ICloudNatSEntry{}, nil
}

func (self *SNatGateway) GetINatDEntryById(id string) (cloudprovider.ICloudNatDEntry, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SNatGateway) GetINatSEntryById(id string) (cloudprovider.ICloudNatSEntry, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SNatGateway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SNatGateway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SNatGateway) GetINetworkId() string {
	return self.SubnetId
}

func (self *SNatGateway) GetBandwidthMb() int {
	return 0
}

func (self *SNatGateway) GetIpAddr() string {
	ipAddrs := []string{}
	for _, addr := range self.NatGatewayAddresses {
		if len(addr.PrivateIp) > 0 {
			ipAddrs = append(ipAddrs, addr.PrivateIp)
		}
	}
	return strings.Join(ipAddrs, ",")
}

func (self *SNatGateway) Delete() error {
	return self.region.DeleteNatgateway(self.NatGatewayId)
}

func (self *SRegion) DeleteNatgateway(id string) error {
	params := map[string]string{
		"NatGatewayId": id,
	}
	return self.ec2Request("DeleteNatGateway", params, nil)
}

func (self *SRegion) GetNatGateways(ids []string, vpcId, subnetId string) ([]SNatGateway, error) {
	params := map[string]string{}
	for i, id := range ids {
		params[fmt.Sprintf("NatGatewayId.%d", i+1)] = id
	}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcId
		idx++
	}
	if len(subnetId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "subnet-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = subnetId
		idx++
	}
	params[fmt.Sprintf("Filter.%d.Name", idx)] = "state"
	for i, state := range []string{
		"pending",
		"failed",
		"available",
		"deleting",
	} {
		params[fmt.Sprintf("Filter.%d.Value.%d", idx, i+1)] = state
	}
	idx++
	ret := []SNatGateway{}
	for {
		result := struct {
			Nats      []SNatGateway `xml:"natGatewaySet>item"`
			NextToken string        `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeNatGateways", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeNatGateways")
		}
		ret = append(ret, result.Nats...)
		if len(result.NextToken) == 0 || len(result.Nats) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetNatGateway(id string) (*SNatGateway, error) {
	nats, err := self.GetNatGateways([]string{id}, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetNatGateways")
	}
	for i := range nats {
		if nats[i].GetGlobalId() == id {
			nats[i].region = self
			return &nats[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats, err := self.region.GetNatGateways(nil, self.VpcId, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetINatGateways")
	}
	ret := []cloudprovider.ICloudNatGateway{}
	for i := range nats {
		nats[i].region = self.region
		ret = append(ret, &nats[i])
	}
	return ret, nil
}

func (self *SRegion) DescribeTags(resType, instanceId string) (map[string]string, error) {
	params := map[string]string{
		"Filter.1.Name":    "resource-id",
		"Filter.1.Value.1": instanceId,
		"Filter.2.Name":    "resource-type",
		"Filter.2.Value.1": resType,
	}
	ret := struct {
		NextToken string
		AwsTags
	}{}
	err := self.ec2Request("DescribeTags", params, &ret)
	if err != nil {
		return nil, err
	}
	return ret.GetTags()
}

func (self *SRegion) DeleteTags(instanceId string, tags map[string]string) error {
	params := map[string]string{
		"ResourceId.1": instanceId,
	}
	idx := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tag.%d.Key", idx)] = k
		params[fmt.Sprintf("Tag.%d.Value", idx)] = v
		idx++
	}
	ret := struct {
	}{}
	return self.ec2Request("DeleteTags", params, &ret)
}

func (self *SRegion) CreateTags(instanceId string, tags map[string]string) error {
	params := map[string]string{
		"ResourceId.1": instanceId,
	}
	idx := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tag.%d.Key", idx)] = k
		params[fmt.Sprintf("Tag.%d.Value", idx)] = v
		idx++
	}
	ret := struct {
	}{}
	return self.ec2Request("CreateTags", params, &ret)
}

func (self *SRegion) setTags(resType, resId string, tags map[string]string, replace bool) error {
	oldTags, err := self.DescribeTags(resType, resId)
	if err != nil {
		return errors.Wrapf(err, "DescribeTags")
	}
	added, removed := map[string]string{}, map[string]string{}
	for k, v := range tags {
		oldValue, ok := oldTags[k]
		if !ok {
			added[k] = v
		} else if oldValue != v {
			removed[k] = oldValue
			added[k] = v
		}
	}
	if replace {
		for k, v := range oldTags {
			newValue, ok := tags[k]
			if !ok {
				removed[k] = v
			} else if v != newValue {
				added[k] = newValue
				removed[k] = v
			}
		}
	}
	if len(removed) > 0 {
		err = self.DeleteTags(resId, removed)
		if err != nil {
			return errors.Wrapf(err, "DeleteTags %s", removed)
		}
	}
	if len(added) > 0 {
		return self.CreateTags(resId, added)
	}
	return nil
}

func (self *SNatGateway) SetTags(tags map[string]string, replace bool) error {
	return self.region.setTags("natgateway", self.NatGatewayId, tags, replace)
}
