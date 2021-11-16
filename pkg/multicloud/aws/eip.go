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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SEipAddress struct {
	region *SRegion
	multicloud.SEipBase
	multicloud.AwsTags

	AllocationId            string `xml:"allocationId"`
	AssociationId           string `xml:"associationId"`
	CarrierIp               string `xml:"carrierIp"`
	CustomerOwnedIp         string `xml:"customerOwnedIp"`
	CustomerOwnedIpv4Pool   string `xml:"customerOwnedIpv4Pool"`
	Domain                  string `xml:"domain"`
	InstanceId              string `xml:"instanceId"`
	NetworkBorderGroup      string `xml:"networkBorderGroup"`
	NetworkInterfaceId      string `xml:"networkInterfaceId"`
	NetworkInterfaceOwnerId string `xml:"networkInterfaceOwnerId"`
	PrivateIpAddress        string `xml:"privateIpAddress"`
	PublicIp                string `xml:"publicIp"`
	PublicIpv4Pool          string `xml:"publicIpv4Pool"`
}

func (self *SEipAddress) GetId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.AllocationId
}

func (self *SEipAddress) GetGlobalId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetStatus() string {
	return api.EIP_STATUS_READY
}

func (self *SEipAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	eip, err := self.region.GetEip(self.AllocationId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEipAddress) IsEmulated() bool {
	if self.AllocationId == self.InstanceId {
		return true
	}
	return false
}

func (self *SEipAddress) GetIpAddr() string {
	return self.PublicIp
}

func (self *SEipAddress) GetMode() string {
	if self.InstanceId == self.AllocationId {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetAssociationType() string {
	return api.EIP_ASSOCIATE_TYPE_SERVER
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return self.InstanceId
}

func (self *SEipAddress) GetBandwidth() int {
	return 0
}

func (self *SEipAddress) GetINetworkId() string {
	return ""
}

func (self *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.AllocationId)
}

func (self *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	return self.region.AssociateEip(self.AllocationId, conf.InstanceId)
}

func (self *SEipAddress) Dissociate() error {
	return self.region.DissociateEip(self.AllocationId, self.AssociationId)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.AllocationId, bw)
}

func (self *SRegion) GetEips(instanceId, eipId string, eipAddress string) ([]SEipAddress, error) {
	params := map[string]string{}
	if len(eipId) > 0 {
		params["AllocationId.1"] = eipId
	}

	if len(instanceId) > 0 {
		params["Filter.1.instance-id"] = instanceId
	}

	if len(eipAddress) > 0 {
		params["PublicIp.1"] = eipAddress
	}

	ret := struct {
		Eips []SEipAddress `xml:"addressesSet>item"`
	}{}

	return ret.Eips, self.ec2Request("DescribeAddresses", params, &ret)
}

func (self *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eips, err := self.GetEips("", eipId, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetEips")
	}
	for i := range eips {
		if eips[i].GetId() == eipId {
			return &eips[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, eipId)
}

func (self *SRegion) GetEipByIpAddress(eipAddress string) (*SEipAddress, error) {
	eips, err := self.GetEips("", "", eipAddress)
	if err != nil {
		return nil, errors.Wrap(err, "GetEips")
	}
	for i := range eips {
		if eips[i].GetIpAddr() == eipAddress {
			return &eips[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, eipAddress)
}

func (self *SRegion) AllocateEIP(name, domainType string) (*SEipAddress, error) {
	params := map[string]string{
		"TagSpecification.1.ResourceType": "elastic-ip",
		"TagSpecification.1.Tags.1.Key":   "Name",
		"TagSpecification.1.Tags.1.Value": name,
	}
	if len(domainType) > 0 {
		params["Domain"] = domainType
	}
	ret := SEipAddress{region: self}
	return &ret, self.ec2Request("AllocateAddress", params, &ret)
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return self.AllocateEIP(opts.Name, "vpc")
}

func (self *SRegion) DeallocateEIP(eipId string) error {
	params := map[string]string{
		"AllocationId": eipId,
	}
	return self.ec2Request("ReleaseAddress", params, nil)
}

func (self *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := map[string]string{
		"AllocationId": eipId,
		"InstanceId":   instanceId,
	}
	return self.ec2Request("AssociateAddress", params, nil)
}

func (self *SRegion) DissociateEip(eipId string, associateId string) error {
	params := map[string]string{
		"AssociationId": associateId,
	}
	return self.ec2Request("DisassociateAddress", params, nil)
}

func (self *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SEipAddress) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (self *SEipAddress) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) GetProjectId() string {
	return ""
}
