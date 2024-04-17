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

package oracle

import (
	"net/url"
	"time"

	"yunion.io/x/jsonutils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEipAddress struct {
	region *SRegion
	multicloud.SEipBase
	SOracleTag

	Id                 string
	AssignedEntityType string
	AvailabilityDomain string
	DisplayName        string
	IpAddress          string
	LifecycleState     string
	Lifetime           string
	Scope              string
	TimeCreated        time.Time
}

func (self *SEipAddress) GetId() string {
	return self.Id
}

func (self *SEipAddress) GetName() string {
	return self.DisplayName
}

func (self *SEipAddress) GetGlobalId() string {
	return self.Id
}

func (self *SEipAddress) GetStatus() string {
	// ASSIGNED, ASSIGNING, AVAILABLE, PROVISIONING, TERMINATED, TERMINATING, UNASSIGNED, UNASSIGNING
	switch self.LifecycleState {
	case "ASSIGNED", "AVAILABLE", "UNASSIGNED":
		return api.EIP_STATUS_READY
	case "ASSIGNING":
		return api.EIP_STATUS_ASSOCIATE
	case "PROVISIONING":
		return api.EIP_STATUS_ALLOCATE
	case "TERMINATED", "TERMINATING":
		return api.EIP_STATUS_DEALLOCATE
	case "UNASSIGNING":
		return api.EIP_STATUS_DISSOCIATE
	}
	return api.EIP_STATUS_READY
}

func (self *SEipAddress) Refresh() error {
	eip, err := self.region.GetEip(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEipAddress) IsEmulated() bool {
	return self.Lifetime == "EPHEMERAL"
}

func (self *SEipAddress) GetIpAddr() string {
	return self.IpAddress
}

func (self *SEipAddress) GetMode() string {
	if self.Lifetime == "EPHEMERAL" {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetAssociationType() string {
	return api.EIP_ASSOCIATE_TYPE_SERVER
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return ""
}

func (self *SEipAddress) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) GetBandwidth() int {
	return 0
}

func (self *SEipAddress) GetINetworkId() string {
	return ""
}

func (self *SEipAddress) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SEipAddress) GetCreatedAt() time.Time {
	return self.TimeCreated
}

func (self *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SEipAddress) GetProjectId() string {
	return ""
}

func (self *SRegion) GetEip(id string) (*SEipAddress, error) {
	resp, err := self.get(SERVICE_IAAS, "publicIps", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SEipAddress{region: self}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetEips(lifetime string) ([]SEipAddress, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("scope", "AVAILABILITY_DOMAIN")
	if len(lifetime) > 0 {
		query.Set("lifetime", lifetime)
	}
	ret := []SEipAddress{}
	for _, zone := range zones {
		query.Set("availabilityDomain", zone.Name)
		resp, err := self.list(SERVICE_IAAS, "publicIps", query)
		if err != nil {
			return nil, err
		}
		part := []SEipAddress{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part...)
	}
	return ret, nil
}
