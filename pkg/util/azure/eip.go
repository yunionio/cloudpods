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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	PublicIPAddressVersion   string           `json:"publicIPAddressVersion,omitempty"`
	IPAddress                string           `json:"ipAddress,omitempty"`
	PublicIPAllocationMethod string           `json:"publicIPAllocationMethod,omitempty"`
	ProvisioningState        string           `json:"provisioningState,omitempty"`
	IPConfiguration          *IPConfiguration `json:"ipConfiguration,omitempty"`
}

type SEipAddress struct {
	region *SRegion

	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat `json:"properties,omitempty"`
	Type       string
	Sku        *PublicIPAddressSku
}

func (region *SRegion) AllocateEIP(eipName string) (*SEipAddress, error) {
	eip := SEipAddress{
		region:   region,
		Location: region.Name,
		Name:     eipName,
		Properties: PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   "IPv4",
			PublicIPAllocationMethod: "Static",
		},
		Type: "Microsoft.Network/publicIPAddresses",
	}
	err := region.client.Create(jsonutils.Marshal(eip), &eip)
	if err != nil {
		return nil, err
	}
	return &eip, cloudprovider.WaitStatus(&eip, api.EIP_STATUS_READY, 10*time.Second, 300*time.Second)
}

func (region *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return region.AllocateEIP(eip.Name)
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eip := SEipAddress{region: region}
	return &eip, region.client.Get(eipId, []string{}, &eip)
}

func (self *SEipAddress) Associate(instanceId string) error {
	return self.region.AssociateEip(self.ID, instanceId)
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	if len(instance.Properties.NetworkProfile.NetworkInterfaces) > 0 {
		nic, err := region.GetNetworkInterfaceDetail(instance.Properties.NetworkProfile.NetworkInterfaces[0].ID)
		if err != nil {
			return err
		}
		log.Errorf("nic: %s", jsonutils.Marshal(nic).PrettyString())
		if len(nic.Properties.IPConfigurations) > 0 {
			nic.Properties.IPConfigurations[0].Properties.PublicIPAddress = &PublicIPAddress{ID: eipId}
			return region.client.Update(jsonutils.Marshal(nic), nil)
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
	startTime := time.Now()
	timeout := time.Minute * 3
	for {
		err := region.client.Delete(eipId)
		if err == nil {
			return nil
		}
		// {"error":{"code":"PublicIPAddressCannotBeDeleted","details":[],"message":"Public IP address /subscriptions/d4f0ec08-3e28-4ae5-bdf9-3dc7c5b0eeca/resourceGroups/Default/providers/Microsoft.Network/publicIPAddresses/eip-for-test-wwl can not be deleted since it is still allocated to resource /subscriptions/d4f0ec08-3e28-4ae5-bdf9-3dc7c5b0eeca/resourceGroups/Default/providers/Microsoft.Network/networkInterfaces/test-wwl-ipconfig."}}
		// 刚解绑eip后可能数据未刷新，需要再次尝试
		if strings.Contains(err.Error(), "it is still allocated to resource") {
			time.Sleep(time.Second * 5)
		} else {
			return err
		}
		if time.Now().Sub(startTime) > timeout {
			return err
		}
	}
}

func (self *SEipAddress) Dissociate() error {
	return self.region.DissociateEip(self.ID)
}

func (region *SRegion) DissociateEip(eipId string) error {
	eip, err := region.GetEip(eipId)
	if err != nil {
		return err
	}
	if eip.Properties.IPConfiguration == nil {
		log.Debugf("eip %s not associate any instance", eip.Name)
		return nil
	}
	interfaceId := eip.Properties.IPConfiguration.ID
	if strings.Index(interfaceId, "/ipConfigurations/") > 0 {
		interfaceId = strings.Split(interfaceId, "/ipConfigurations/")[0]
	}
	nic, err := region.GetNetworkInterfaceDetail(interfaceId)
	if err != nil {
		return err
	}
	for i := 0; i < len(nic.Properties.IPConfigurations); i++ {
		if nic.Properties.IPConfigurations[i].Properties.PublicIPAddress != nil && nic.Properties.IPConfigurations[i].Properties.PublicIPAddress.ID == eipId {
			nic.Properties.IPConfigurations[i].Properties.PublicIPAddress = nil
			break
		}
	}
	return region.client.Update(jsonutils.Marshal(nic), nil)
}

func (self *SEipAddress) GetAssociationExternalId() string {
	if self.Properties.IPConfiguration != nil {
		interfaceId := self.Properties.IPConfiguration.ID
		if strings.Index(interfaceId, "/ipConfigurations/") > 0 {
			interfaceId = strings.Split(interfaceId, "/ipConfigurations/")[0]
		}
		nic, err := self.region.GetNetworkInterfaceDetail(interfaceId)
		if err != nil {
			log.Errorf("Failt to find NetworkInterface for eip %s", self.Name)
			return ""
		}
		if nic.Properties.VirtualMachine != nil {
			return nic.Properties.VirtualMachine.ID
		}
	}
	return ""
}

func (self *SEipAddress) GetAssociationType() string {
	return "server"
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

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetMode() string {
	if self.IsEmulated() {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	if self.Properties.IPConfiguration != nil {
		nic, err := self.region.GetNetworkInterfaceDetail(self.Properties.IPConfiguration.ID)
		if err == nil && nic.Properties.VirtualMachine != nil && len(nic.Properties.VirtualMachine.ID) > 0 {
			return api.EIP_MODE_INSTANCE_PUBLICIP
		}
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
