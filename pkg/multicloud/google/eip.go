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

package google

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SAddress struct {
	region *SRegion

	Id                string
	CreationTimestamp time.Time
	Name              string
	Description       string
	Address           string
	Status            string
	Region            string
	Users             []string
	SelfLink          string
	NetworkTier       string
	AddressType       string
	Kind              string
}

func (region *SRegion) GetEips(address string, maxResults int, pageToken string) ([]SAddress, error) {
	eips := []SAddress{}
	params := map[string]string{}
	if len(address) > 0 {
		params["filter"] = fmt.Sprintf(`address="%s"`, address)
	}
	resource := fmt.Sprintf("regions/%s/addresses", region.Name)
	return eips, region.List(resource, params, maxResults, pageToken, &eips)
}

func (region *SRegion) GetEip(id string) (*SAddress, error) {
	eip := &SAddress{region: region}
	return eip, region.Get(id, eip)
}

func (addr *SAddress) GetId() string {
	return addr.SelfLink
}

func (addr *SAddress) GetName() string {
	return addr.Name
}

func (addr *SAddress) GetGlobalId() string {
	return getGlobalId(addr.SelfLink)
}

func (addr *SAddress) GetStatus() string {
	switch addr.Status {
	case "RESERVING":
		return api.EIP_STATUS_ASSOCIATE
	case "RESERVED":
		return api.EIP_STATUS_READY
	case "IN_USE":
		return api.EIP_STATUS_READY
	default:
		log.Errorf("Unknown eip status: %s", addr.Status)
		return api.EIP_STATUS_UNKNOWN
	}
}

func (addr *SAddress) GetProjectId() string {
	return addr.region.GetProjectId()
}

func (addr *SAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (addr *SAddress) IsEmulated() bool {
	if addr.Id == addr.SelfLink {
		return true
	}
	return false
}

func (addr *SAddress) GetCreatedAt() time.Time {
	return addr.CreationTimestamp
}

func (addr *SAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (addr *SAddress) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (addr *SAddress) Refresh() error {
	if addr.IsEmulated() {
		return nil
	}
	_addr, err := addr.region.GetEip(addr.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(addr, _addr)
}

func (addr *SAddress) GetIpAddr() string {
	return addr.Address
}

func (addr *SAddress) GetMode() string {
	if addr.IsEmulated() {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (addr *SAddress) GetINetworkId() string {
	return ""
}

func (addr *SAddress) GetAssociationType() string {
	return api.EIP_ASSOCIATE_TYPE_SERVER
}

func (addr *SAddress) GetAssociationExternalId() string {
	if len(addr.Users) > 0 {
		return getGlobalId(addr.Users[0])
	}
	return ""
}

func (addr *SAddress) GetBandwidth() int {
	return 0
}

func (addr *SAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (addr *SAddress) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (addr *SAddress) Associate(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (addr *SAddress) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (addr *SAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}
