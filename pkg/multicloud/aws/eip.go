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

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	EIP_STATUS_INUSE     = "InUse"
	EIP_STATUS_AVAILABLE = "Available"
)

type SEipAddress struct {
	region *SRegion

	AllocationId            string
	Bandwidth               int
	Tags                    TagSpec
	Status                  string
	InstanceId              string
	AssociationId           string
	Domain                  string
	NetworkInterfaceId      string
	NetworkInterfaceOwnerId string
	PrivateIpAddress        string
	IpAddress               string
	Name                    string
}

func (self *SEipAddress) GetId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetName() string {
	if len(self.Name) == 0 {
		return self.IpAddress
	}

	return self.Name
}

func (self *SEipAddress) GetGlobalId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetStatus() string {
	switch self.Status {
	// todo: EIP_STATUS_INUSE 对应READY？
	case EIP_STATUS_AVAILABLE, EIP_STATUS_INUSE:
		return api.EIP_STATUS_READY
	default:
		return api.EIP_STATUS_UNKNOWN
	}
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

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetIpAddr() string {
	return self.IpAddress
}

func (self *SEipAddress) GetMode() string {
	if self.InstanceId == self.AllocationId {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	} else {
		return api.EIP_MODE_STANDALONE_EIP
	}
}

func (self *SEipAddress) GetAssociationType() string {
	// todo : ?
	return api.EIP_ASSOCIATE_TYPE_SERVER
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return self.InstanceId
}

func (self *SEipAddress) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SEipAddress) GetINetworkId() string {
	return ""
}

func (self *SEipAddress) GetInternetChargeType() string {
	// todo : implement me
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.AllocationId)
}

func (self *SEipAddress) Associate(instanceId string) error {
	err := self.region.AssociateEip(self.AllocationId, instanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 5*time.Second, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.AllocationId, self.InstanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.AllocationId, bw)
}

func (self *SRegion) GetEips(eipId string, eipAddress string, offset int, limit int) ([]SEipAddress, int, error) {
	params := ec2.DescribeAddressesInput{}
	if len(eipId) > 0 {
		params.SetAllocationIds([]*string{&eipId})
	}

	if len(eipAddress) > 0 {
		params.SetPublicIps([]*string{&eipAddress})
	}

	res, err := self.ec2Client.DescribeAddresses(&params)
	err = parseNotFoundError(err)
	if err != nil {
		log.Errorf("DescribeEipAddresses fail %s", err)
		return nil, 0, err
	}

	eips := make([]SEipAddress, 0)
	for _, ip := range res.Addresses {
		if err := FillZero(ip); err != nil {
			return nil, 0, err
		}

		tagspec := TagSpec{ResourceType: "eip"}
		tagspec.LoadingEc2Tags(ip.Tags)

		var status string
		if len(*ip.AssociationId) > 0 {
			status = EIP_STATUS_INUSE
		} else {
			status = EIP_STATUS_AVAILABLE
		}

		eips = append(eips, SEipAddress{
			region:                  self,
			AllocationId:            *ip.AllocationId,
			Tags:                    tagspec,
			Status:                  status,
			InstanceId:              *ip.InstanceId,
			AssociationId:           *ip.AssociationId,
			Domain:                  *ip.Domain,
			NetworkInterfaceId:      *ip.NetworkInterfaceId,
			NetworkInterfaceOwnerId: *ip.NetworkInterfaceOwnerId,
			PrivateIpAddress:        *ip.PrivateIpAddress,
			IpAddress:               *ip.PublicIp,
			Name:                    tagspec.GetNameTag(),
		})
	}
	return eips, len(eips), nil
}

func (self *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	// 这里必须强制要求eipId大于零。避免用户账号正好只有一个eip的情况，返回错误的eip。
	if len(eipId) == 0 {
		return nil, fmt.Errorf("GetEip eipId should not be emtpy.")
	}

	eips, total, err := self.GetEips(eipId, "", 0, 0)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, ErrorNotFound()
	}
	return &eips[0], nil
}

func (self *SRegion) GetEipByIpAddress(eipAddress string) (*SEipAddress, error) {
	eips, total, err := self.GetEips("", eipAddress, 0, 0)
	if err != nil {
		return nil, err
	}

	if total != 1 {
		return nil, ErrorNotFound()
	}
	return &eips[0], nil
}

func (self *SRegion) AllocateEIP(domainType string) (*SEipAddress, error) {
	params := &ec2.AllocateAddressInput{}
	params.SetDomain(domainType)
	eip, err := self.ec2Client.AllocateAddress(params)
	if err != nil {
		log.Errorf("AllocateEipAddress fail %s", err)
		return nil, err
	}

	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetEip(*eip.AllocationId)
}

func (self *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	// todo: aws 不支持指定bwMbps, chargeType ？
	log.Debugf("CreateEip: aws not support specific params name/bwMbps/chargeType.")
	ieip, err := self.AllocateEIP("vpc")
	if err == nil && len(eip.Name) > 0 {
		eipId := ieip.GetId()
		k := "Name"
		nameTag := &ec2.Tag{Key: &k, Value: &eip.Name}
		params := &ec2.CreateTagsInput{}
		params.SetResources([]*string{&eipId})
		params.SetTags([]*ec2.Tag{nameTag})

		// name 创建成功与否不影响eip的正常使用
		if _, e := self.ec2Client.CreateTags(params); e != nil {
			log.Infof("CreateEIP create name tag failed: %s", e)
		}
	}

	return ieip, err
}

func (self *SRegion) DeallocateEIP(eipId string) error {
	params := &ec2.ReleaseAddressInput{}
	params.SetAllocationId(eipId)
	_, err := self.ec2Client.ReleaseAddress(params)
	return err
}

func (self *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := &ec2.AssociateAddressInput{}
	params.SetAllocationId(eipId)
	params.SetInstanceId(instanceId)
	_, err := self.ec2Client.AssociateAddress(params)
	return err
}

func (self *SRegion) DissociateEip(eipId string, instanceId string) error {
	eip, err := self.GetEip(eipId)
	if err != nil {
		return err
	}

	if len(eip.AssociationId) == 0 {
		// 已经是解绑状态
		return nil
	}

	if eip.InstanceId != instanceId {
		return fmt.Errorf("eip %s associate with another instance %s", eipId, eip.InstanceId)
	}

	params := &ec2.DisassociateAddressInput{}
	params.SetAssociationId(eip.AssociationId)
	_, err = self.ec2Client.DisassociateAddress(params)
	return err
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
