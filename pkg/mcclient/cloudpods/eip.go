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

package cloudpods

import (
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SEip struct {
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	CloudpodsTags
	region *SRegion

	api.ElasticipDetails
}

func (self *SEip) GetName() string {
	return self.Name
}

func (self *SEip) GetId() string {
	return self.Id
}

func (self *SEip) GetGlobalId() string {
	return self.Id
}

func (self *SEip) GetProjectId() string {
	return self.TenantId
}

func (self *SEip) GetStatus() string {
	return self.Status
}

func (self *SEip) Refresh() error {
	eip, err := self.region.GetEip(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEip) GetBillingType() string {
	return self.BillingType
}

func (self *SEip) GetIpAddr() string {
	return self.IpAddr
}

func (self *SEip) GetINetworkId() string {
	return self.NetworkId
}

func (self *SEip) GetAssociationType() string {
	return self.AssociateType
}

func (self *SEip) GetAssociationExternalId() string {
	return self.AssociateId
}

func (self *SEip) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SEip) GetInternetChargeType() string {
	return self.ChargeType
}

func (self *SEip) Delete() error {
	return self.region.cli.delete(&modules.Elasticips, self.Id)
}

func (self *SEip) IsAutoRenew() bool {
	return self.AutoRenew
}

func (self *SEip) Associate(opts *cloudprovider.AssociateConfig) error {
	input := api.ElasticipAssociateInput{}
	input.InstanceType = opts.AssociateType
	input.InstanceId = opts.InstanceId
	switch opts.AssociateType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
	default:
		return cloudprovider.ErrNotImplemented
	}
	_, err := self.region.perform(&modules.Elasticips, self.Id, "associate", input)
	return err
}

func (self *SEip) Dissociate() error {
	_, err := self.region.perform(&modules.Elasticips, self.Id, "dissociate", nil)
	return err
}

func (self *SEip) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SEip) GetExpiredAt() time.Time {
	return self.ExpiredAt
}

func (self *SEip) GetMode() string {
	return self.Mode
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.GetEips("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = self
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	input := api.SElasticipCreateInput{}
	input.Name = opts.Name
	input.CloudregionId = self.Id
	input.ChargeType = opts.ChargeType
	input.BandwidthMb = opts.BandwidthMbps
	input.ProjectId = opts.ProjectId
	input.NetworkId = opts.NetworkExternalId
	input.BgpType = opts.BGPType
	input.IpAddr = opts.Ip
	eip := &SEip{region: self}
	return eip, self.create(&modules.Elasticips, input, eip)
}

func (self *SRegion) GetEips(associateId string) ([]SEip, error) {
	eips := []SEip{}
	params := map[string]interface{}{}
	if len(associateId) > 0 {
		params["associate_id"] = associateId
	}
	return eips, self.list(&modules.Elasticips, params, &eips)
}

func (self *SRegion) GetEip(id string) (*SEip, error) {
	eip := &SEip{region: self}
	return eip, self.cli.get(&modules.Elasticips, id, nil, eip)
}
