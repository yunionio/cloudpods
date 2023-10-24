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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	EIP_STATUS_INUSE     = "InUse"
	EIP_STATUS_AVAILABLE = "Available"
)

type SEipAddress struct {
	region *SRegion
	multicloud.SEipBase
	AwsTags

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
	new, err := self.region.GetEip(self.AllocationId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
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
	if len(self.InstanceId) > 0 {
		return api.EIP_ASSOCIATE_TYPE_SERVER
	}
	if len(self.NetworkInterfaceId) > 0 {
		net, err := self.region.GetNetworkInterface(self.NetworkInterfaceId)
		if err != nil {
			return ""
		}
		switch net.InterfaceType {
		case "nat_gateway":
			return api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
		default:
			return net.InterfaceType
		}
	}
	return ""
}

func (self *SEipAddress) GetAssociationExternalId() string {
	if len(self.InstanceId) > 0 {
		return self.InstanceId
	}
	if len(self.NetworkInterfaceId) > 0 {
		net, err := self.region.GetNetworkInterface(self.NetworkInterfaceId)
		if err != nil {
			return ""
		}
		switch net.InterfaceType {
		case "nat_gateway":
			nats, err := self.region.GetNatGateways(nil, net.VpcId, net.SubnetId)
			if err != nil {
				return ""
			}
			for i := range nats {
				for _, addr := range nats[i].NatGatewayAddresses {
					if addr.PublicIp == self.PublicIp {
						return nats[i].GetGlobalId()
					}
				}
			}
			return ""
		}
	}
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
	for i := 0; i < 3; i++ {
		err := self.region.AssociateEip(self.AllocationId, conf.InstanceId)
		if err == nil {
			return nil
		}
		if e, ok := err.(*sAwsError); ok && e.Errors.Code == "InvalidAllocationID.NotFound" {
			time.Sleep(time.Second * 30)
			continue
		}
		return errors.Wrapf(err, "AssociateEip")
	}
	return nil
}

func (self *SEipAddress) Dissociate() error {
	return self.region.DissociateEip(self.AssociationId)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.AllocationId, bw)
}

func (self *SRegion) GetEips(id, ip, associateId string) ([]SEipAddress, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["AllocationId.1"] = id
	}
	if len(ip) > 0 {
		params["PublicIp.1"] = ip
	}
	idx := 1
	if len(associateId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "association-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = associateId
		idx++
	}
	result := struct {
		AddressesSet []SEipAddress `xml:"addressesSet>item"`
	}{}
	for i := 0; i < 3; i++ {
		err := self.ec2Request("DescribeAddresses", params, &result)
		if err == nil {
			return result.AddressesSet, nil
		}
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			time.Sleep(time.Second * 10)
			continue
		}
		return nil, errors.Wrapf(err, "DescribeAddresses")
	}
	return result.AddressesSet, nil
}

func (self *SRegion) GetEip(id string) (*SEipAddress, error) {
	eips, err := self.GetEips(id, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetEips")
	}
	for i := range eips {
		if eips[i].GetGlobalId() == id {
			eips[i].region = self
			return &eips[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetEipByIpAddress(eipAddress string) (*SEipAddress, error) {
	eips, err := self.GetEips("", eipAddress, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetEips")
	}
	for i := range eips {
		if eips[i].GetIpAddr() == eipAddress {
			eips[i].region = self
			return &eips[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, eipAddress)
}

func (self *SRegion) AllocateEIP(opts *cloudprovider.SEip) (*SEipAddress, error) {
	params := map[string]string{
		"Domain": "vpc",
	}
	if len(opts.Name) > 0 {
		params["TagSpecification.1.ResourceType"] = "elastic-ip"
		params["TagSpecification.1.Tag.1.Key"] = "Name"
		params["TagSpecification.1.Tag.1.Value"] = opts.Name
	}
	ret := SEipAddress{region: self}
	err := self.ec2Request("AllocateAddress", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "AllocateAddress")
	}
	return &ret, nil
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	eip, err := self.AllocateEIP(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "AllocateEIP")
	}
	return eip, nil
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

func (self *SRegion) DissociateEip(insId string) error {
	params := map[string]string{
		"AssociationId": insId,
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

func (self *SEipAddress) SetTags(tags map[string]string, replace bool) error {
	return self.region.setTags("elastic-ip", self.AllocationId, tags, replace)
}
