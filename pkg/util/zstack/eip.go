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

package zstack

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SEipAddress struct {
	region *SRegion

	ZStackBasic
	VMNicUUID string `json:"vmNicUuid"`
	VipUUID   string `json:"vipUuid"`
	State     string `json:"state"`
	VipIP     string `json:"vipIp"`
	GuestIP   string `json:"guestIp"`
	ZStackTime
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eip := &SEipAddress{region: region}
	return eip, region.client.getResource("eips", eipId, eip)
}

func (region *SRegion) GetEips(eipId, instanceId string) ([]SEipAddress, error) {
	eips := []SEipAddress{}
	params := url.Values{}
	if len(eipId) > 0 {
		params.Add("q", "uuid="+eipId)
	}
	if len(instanceId) > 0 {
		params.Add("q", "vmNic.vmInstanceUuid="+instanceId)
	}
	err := region.client.listAll("eips", params, &eips)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(eips); i++ {
		eips[i].region = region
	}
	return eips, nil
}

func (eip *SEipAddress) GetId() string {
	return eip.UUID
}

func (eip *SEipAddress) GetName() string {
	return eip.Name
}

func (eip *SEipAddress) GetGlobalId() string {
	return eip.UUID
}

func (eip *SEipAddress) GetStatus() string {
	return api.EIP_STATUS_READY
}

func (eip *SEipAddress) Refresh() error {
	new, err := eip.region.GetEip(eip.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(eip, new)
}

func (eip *SEipAddress) IsEmulated() bool {
	return false
}

func (eip *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (eip *SEipAddress) GetIpAddr() string {
	return eip.VipIP
}

func (eip *SEipAddress) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (eip *SEipAddress) GetAssociationType() string {
	return "server"
}

func (eip *SEipAddress) GetAssociationExternalId() string {
	if len(eip.VMNicUUID) > 0 {
		instances, err := eip.region.GetInstances("", "", eip.VMNicUUID)
		if err == nil && len(instances) == 1 {
			return instances[0].UUID
		}
	}
	return ""
}

func (eip *SEipAddress) GetBillingType() string {
	return ""
}

func (eip *SEipAddress) GetCreatedAt() time.Time {
	return eip.CreateDate
}

func (eip *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (eip *SEipAddress) Delete() error {
	return eip.region.DeleteVirtualIP(eip.VipUUID)
}

func (eip *SEipAddress) GetBandwidth() int {
	return 0
}

func (eip *SEipAddress) GetINetworkId() string {
	vip, err := eip.region.GetVirtualIP(eip.VipUUID)
	if err == nil {
		return eip.region.GetNetworkId(vip)
	}
	return ""
}

func (eip *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (eip *SEipAddress) Associate(instanceId string) error {
	return eip.region.AssociateEip(instanceId, eip.UUID)
}

func (eip *SEipAddress) Dissociate() error {
	return eip.region.DisassociateEip(eip.UUID)
}

func (eip *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (eip *SEipAddress) GetProjectId() string {
	return ""
}

func (region *SRegion) CreateEip(name string, vipId string, desc string) (*SEipAddress, error) {
	params := map[string]map[string]string{
		"params": {
			"name":        name,
			"description": desc,
			"vipUuid":     vipId,
		},
	}
	eip := &SEipAddress{region: region}
	resp, err := region.client.post("eips", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	return eip, resp.Unmarshal(eip, "inventory")
}

func (region *SRegion) AssociateEip(instanceId, eipId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	if len(instance.VMNics) == 0 {
		return fmt.Errorf("instance %s does not have any nic", instance.Name)
	}
	resource := fmt.Sprintf("eips/%s/vm-instances/nics/%s", eipId, instance.VMNics[0].UUID)
	params := map[string]interface{}{
		"params": jsonutils.NewDict(),
	}
	_, err = region.client.post(resource, jsonutils.Marshal(params))
	return err
}

func (region *SRegion) DisassociateEip(eipId string) error {
	resource := fmt.Sprintf("eips/%s/vm-instances/nics", eipId)
	return region.client.delete(resource, "", "")
}
