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

package azure

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type ClassicEipProperties struct {
	IpAddress         string       `json:"ipAddress,omitempty"`
	Status            string       `json:"status,omitempty"`
	ProvisioningState string       `json:"provisioningState,omitempty"`
	InUse             bool         `json:"inUse,omitempty"`
	AttachedTo        *SubResource `joson:"attachedTo,omitempty"`
}

type SClassicEipAddress struct {
	region *SRegion

	ID         string
	instanceId string
	Name       string
	Location   string
	Properties ClassicEipProperties `json:"properties,omitempty"`
	Type       string
}

func (self *SClassicEipAddress) Associate(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicEipAddress) Delete() error {
	if !self.IsEmulated() {
		return self.region.DeallocateEIP(self.ID)
	}
	return nil
}

func (self *SClassicEipAddress) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicEipAddress) GetAssociationExternalId() string {
	// TODO
	return self.instanceId
}

func (self *SClassicEipAddress) GetAssociationType() string {
	return "server"
}

func (self *SClassicEipAddress) GetBandwidth() int {
	return 0
}

func (self *SClassicEipAddress) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicEipAddress) GetId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SClassicEipAddress) GetIpAddr() string {
	return self.Properties.IpAddress
}

func (self *SClassicEipAddress) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SClassicEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicEipAddress) GetMode() string {
	// TODO
	if self.instanceId == self.ID {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SClassicEipAddress) GetName() string {
	return self.Name
}

func (self *SClassicEipAddress) GetStatus() string {
	switch self.Properties.Status {
	case "Created", "":
		return api.EIP_STATUS_READY
	default:
		log.Errorf("Unknown eip status: %s", self.Properties.ProvisioningState)
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SClassicEipAddress) IsEmulated() bool {
	if self.ID == self.instanceId {
		return true
	}
	return false
}

func (region *SRegion) GetClassicEip(eipId string) (*SClassicEipAddress, error) {
	eip := SClassicEipAddress{region: region}
	return &eip, region.client.Get(eipId, []string{}, &eip)
}

func (self *SClassicEipAddress) Refresh() error {
	eip, err := self.region.GetClassicEip(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (region *SRegion) GetClassicEips() ([]SClassicEipAddress, error) {
	eips := []SClassicEipAddress{}
	err := region.client.ListAll("Microsoft.ClassicNetwork/reservedIps", &eips)
	if err != nil {
		return nil, err
	}
	result := []SClassicEipAddress{}
	for i := 0; i < len(eips); i++ {
		if eips[i].Location == region.Name {
			eips[i].region = region
			result = append(result, eips[i])
		}
	}
	return result, nil
}

func (self *SClassicEipAddress) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SClassicEipAddress) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SClassicEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SClassicEipAddress) GetProjectId() string {
	return getResourceGroup(self.ID)
}
