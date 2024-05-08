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
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type PublicIPAddressSku struct {
	Name string
}

type IPConfigurationPropertiesFormat struct {
	PrivateIPAddress string
}

type IPConfiguration struct {
	Name string
	ID   string
}

type PublicIPAddressPropertiesFormat struct {
	PublicIPAddressVersion   string          `json:"publicIPAddressVersion,omitempty"`
	IPAddress                string          `json:"ipAddress,omitempty"`
	PublicIPAllocationMethod string          `json:"publicIPAllocationMethod,omitempty"`
	ProvisioningState        string          `json:"provisioningState,omitempty"`
	IPConfiguration          IPConfiguration `json:"ipConfiguration,omitempty"`
}

type SEipAddress struct {
	region *SRegion
	multicloud.SEipBase
	AzureTags

	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat `json:"properties,omitempty"`
	Type       string
	Sku        *PublicIPAddressSku
}

func (self *SRegion) AllocateEIP(name, projectId string) (*SEipAddress, error) {
	params := map[string]interface{}{
		"Location": self.Name,
		"Name":     name,
		"Properties": map[string]string{
			"PublicIPAddressVersion":   "IPv4",
			"PublicIPAllocationMethod": "Static",
		},
		"Type": "Microsoft.Network/publicIPAddresses",
	}
	eip := &SEipAddress{region: self}
	err := self.create(projectId, jsonutils.Marshal(params), eip)
	if err != nil {
		return nil, err
	}
	return eip, cloudprovider.WaitStatus(eip, api.EIP_STATUS_READY, 10*time.Second, 300*time.Second)
}

func (region *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return region.AllocateEIP(eip.Name, eip.ProjectId)
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eip := SEipAddress{region: region}
	return &eip, region.get(eipId, url.Values{}, &eip)
}

func (self *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	return self.region.AssociateEip(self.ID, conf.InstanceId)
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	if len(instance.Properties.NetworkProfile.NetworkInterfaces) > 0 {
		nic, err := region.GetNetworkInterface(instance.Properties.NetworkProfile.NetworkInterfaces[0].ID)
		if err != nil {
			return err
		}
		if len(nic.Properties.IPConfigurations) > 0 {
			nic.Properties.IPConfigurations[0].Properties.PublicIPAddress = &PublicIPAddress{ID: eipId}
			return region.update(jsonutils.Marshal(nic), nil)
		}
		return fmt.Errorf("network interface with no IPConfigurations")
	}
	return fmt.Errorf("Instance with no interface")
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return region.GetEip(eipId)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SEipAddress) Delete() error {
	self.Dissociate() //避免eip挂载在弹性网卡之上,导致删除失败
	return self.region.DeallocateEIP(self.ID)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	return cloudprovider.Wait(time.Second*5, time.Minute*5, func() (bool, error) {
		err := region.del(eipId)
		if err == nil {
			return true, nil
		}
		// {"error":{"code":"PublicIPAddressCannotBeDeleted","details":[],"message":"Public IP address /subscriptions/d4f0ec08-3e28-4ae5-bdf9-3dc7c5b0eeca/resourceGroups/Default/providers/Microsoft.Network/publicIPAddresses/eip-for-test-wwl can not be deleted since it is still allocated to resource /subscriptions/d4f0ec08-3e28-4ae5-bdf9-3dc7c5b0eeca/resourceGroups/Default/providers/Microsoft.Network/networkInterfaces/test-wwl-ipconfig."}}
		// 刚解绑eip后可能数据未刷新，需要再次尝试
		if strings.Contains(err.Error(), "it is still allocated to resource") {
			return false, nil
		}
		return false, errors.Wrapf(err, "del(%s)", eipId)
	})
}

func (self *SEipAddress) Dissociate() error {
	return self.region.DissociateEip(self.ID)
}

func (region *SRegion) DissociateEip(eipId string) error {
	eip, err := region.GetEip(eipId)
	if err != nil {
		return errors.Wrapf(err, "GetEip(%s)", eipId)
	}
	interfaceId := eip.Properties.IPConfiguration.ID
	if strings.Index(interfaceId, "/ipConfigurations/") > 0 {
		interfaceId = strings.Split(interfaceId, "/ipConfigurations/")[0]
	}
	nic, err := region.GetNetworkInterface(interfaceId)
	if err != nil {
		return err
	}
	for i := 0; i < len(nic.Properties.IPConfigurations); i++ {
		if nic.Properties.IPConfigurations[i].Properties.PublicIPAddress != nil && nic.Properties.IPConfigurations[i].Properties.PublicIPAddress.ID == eipId {
			nic.Properties.IPConfigurations[i].Properties.PublicIPAddress = nil
			break
		}
	}
	return region.update(jsonutils.Marshal(nic), nil)
}

func (self *SEipAddress) GetAssociationExternalId() string {
	info := strings.Split(self.Properties.IPConfiguration.ID, "/")
	if len(info) > 2 {
		return strings.ToLower(strings.Join(info[:len(info)-2], "/"))
	}
	return ""
}

func (self *SEipAddress) GetAssociationType() string {
	if len(self.Properties.IPConfiguration.ID) == 0 {
		return ""
	}
	if info := strings.Split(self.Properties.IPConfiguration.ID, "/"); len(info) > 7 {
		resType := strings.ToLower(info[7])
		if utils.IsInStringArray(resType, []string{"networkinterfaces"}) {
			return api.EIP_ASSOCIATE_TYPE_SERVER
		}
		if utils.IsInStringArray(resType, []string{"loadbalancers", "applicationgateways"}) {
			return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
		}
		return resType
	}
	return ""
}

func (self *SEipAddress) GetBandwidth() int {
	return 0
}

func (self *SEipAddress) GetINetworkId() string {
	return ""
}

func (self *SEipAddress) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SEipAddress) GetId() string {
	return self.ID
}

func (self *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) GetIpAddr() string {
	return self.Properties.IPAddress
}

func (self *SEipAddress) GetMode() string {
	if self.IsEmulated() {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetName() string {
	return self.Name
}

func (self *SEipAddress) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Succeeded", "":
		return api.EIP_STATUS_READY
	case "Updating":
		return api.EIP_STATUS_ALLOCATE
	default:
		log.Errorf("Unknown eip status: %s", self.Properties.ProvisioningState)
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) IsEmulated() bool {
	if strings.ToLower(self.Properties.PublicIPAllocationMethod) == "Dynamic" || len(self.Properties.IPAddress) == 0 {
		return true
	}
	return false
}

func (self *SEipAddress) Refresh() error {
	eip, err := self.region.GetEip(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEipAddress) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SEipAddress) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) GetProjectId() string {
	return getResourceGroup(self.ID)
}
