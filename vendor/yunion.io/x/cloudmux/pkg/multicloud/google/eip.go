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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAddress struct {
	region *SRegion
	SResourceBase
	multicloud.SEipBase
	GoogleTags
	instanceId string

	CreationTimestamp time.Time
	Description       string
	Address           string
	Status            string
	Region            string
	Users             []string
	NetworkTier       string
	AddressType       string
	Kind              string
}

func (region *SRegion) GetEips(address string, maxResults int, pageToken string) ([]SAddress, error) {
	eips := []SAddress{}
	params := map[string]string{}
	filters := []string{"addressType=EXTERNAL"}
	if len(address) > 0 {
		filters = append(filters, fmt.Sprintf(`address="%s"`, address))
	}
	params["filter"] = strings.Join(filters, " ADN ")
	resource := fmt.Sprintf("regions/%s/addresses", region.Name)
	return eips, region.List(resource, params, maxResults, pageToken, &eips)
}

func (region *SRegion) GetEip(id string) (*SAddress, error) {
	eip := &SAddress{region: region}
	return eip, region.Get("addresses", id, eip)
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

func (addr *SAddress) IsEmulated() bool {
	return len(addr.instanceId) > 0
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

func (self *SAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	addr, err := self.region.GetEip(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, addr)
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
	if len(addr.GetAssociationExternalId()) > 0 {
		return api.EIP_ASSOCIATE_TYPE_SERVER
	}
	return ""
}

func (addr *SAddress) GetAssociationExternalId() string {
	if len(addr.instanceId) > 0 {
		return addr.instanceId
	}
	if len(addr.Users) > 0 {
		res := &SResourceBase{}
		err := addr.region.GetBySelfId(addr.Users[0], res)
		if err != nil {
			return ""
		}
		return res.GetGlobalId()
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
	return addr.region.Delete(addr.SelfLink)
}

func (addr *SAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	return addr.region.AssociateInstanceEip(conf.InstanceId, addr.Address)
}

func (addr *SAddress) Dissociate() error {
	if len(addr.Users) > 0 {
		return addr.region.DissociateInstanceEip(addr.Users[0], addr.Address)
	}
	return nil
}

func (addr *SAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) CreateEip(name string, desc string) (*SAddress, error) {
	body := map[string]string{
		"name":        name,
		"description": desc,
	}
	resource := fmt.Sprintf("regions/%s/addresses", region.Name)
	addr := &SAddress{region: region}
	err := region.Insert(resource, jsonutils.Marshal(body), addr)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (region *SRegion) AssociateInstanceEip(instanceId string, eip string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return errors.Wrap(err, "region.GetInstance")
	}
	for _, networkInterface := range instance.NetworkInterfaces {
		body := map[string]interface{}{
			"type":  "ONE_TO_ONE_NAT",
			"name":  "External NAT",
			"natIP": eip,
		}
		params := map[string]string{"networkInterface": networkInterface.Name}
		return region.Do(instance.SelfLink, "addAccessConfig", params, jsonutils.Marshal(body))
	}
	return fmt.Errorf("no valid networkinterface to associate")
}

func (self *SRegion) DissociateInstanceEip(instanceId string, eip string) error {
	instance := SInstance{}
	err := self.GetBySelfId(instanceId, &instance)
	if err != nil {
		return errors.Wrap(err, "region.GetInstance")
	}
	for _, networkInterface := range instance.NetworkInterfaces {
		for _, accessConfig := range networkInterface.AccessConfigs {
			if accessConfig.NatIP == eip {
				body := map[string]string{}
				params := map[string]string{
					"networkInterface": networkInterface.Name,
					"accessConfig":     accessConfig.Name,
				}
				return self.Do(instance.SelfLink, "deleteAccessConfig", params, jsonutils.Marshal(body))
			}
		}
	}
	return nil
}
