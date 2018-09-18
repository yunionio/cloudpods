package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type TInternetChargeType string

const (
	InternetChargeByTraffic = TInternetChargeType("PayByTraffic")
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
	PublicIPAddressVersion   string
	IPAddress                string
	PublicIPAllocationMethod string
	ProvisioningState        string
	IPConfiguration          IPConfiguration
}

type SEipAddress struct {
	region *SRegion

	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat
	Sku        PublicIPAddressSku
}

func (region *SRegion) AllocateEIP(eipName string) (*SEipAddress, error) {
	eip := SEipAddress{region: region}
	_, resourceGroup, eipName := pareResourceGroupWithName(eipName, EIP_RESOURCE)
	networkClient := network.NewPublicIPAddressesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	networkClient.Authorizer = region.client.authorizer
	params := network.PublicIPAddress{
		Location: &region.Name,
		Name:     &eipName,
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Static,
			PublicIPAddressVersion:   network.IPv4,
		},
	}
	if result, err := networkClient.CreateOrUpdate(context.Background(), resourceGroup, eipName, params); err != nil {
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), networkClient.Client); err != nil {
		return nil, err
	} else if value, err := result.Result(networkClient); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&eip, value); err != nil {
		return nil, err
	}
	return &eip, nil
}

func (region *SRegion) CreateEIP(eipName string, bwMbps int, chargeType string) (cloudprovider.ICloudEIP, error) {
	return region.AllocateEIP(eipName)
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	if eips, err := region.GetEips(); err != nil {
		return nil, err
	} else {
		ieips := make([]cloudprovider.ICloudEIP, len(eips))
		for i := 0; i < len(eips); i++ {
			eips[i].region = region
			ieips[i] = &eips[i]
		}
		return ieips, nil
	}
}

func (region *SRegion) GetEips() ([]SEipAddress, error) {
	networkClient := network.NewPublicIPAddressesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	networkClient.Authorizer = region.client.authorizer
	if result, err := networkClient.ListAll(context.Background()); err != nil {
		return nil, err
	} else {
		eips := make([]SEipAddress, 0)
		for i := 0; i < len(result.Values()); i++ {
			eip := SEipAddress{region: region}
			if _eip := result.Values()[i]; *_eip.Location == region.Name {
				if err := jsonutils.Update(&eip, _eip); err != nil {
					return nil, err
				} else {
					eips = append(eips, eip)
				}
			}
		}
		return eips, nil
	}
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eip := SEipAddress{region: region}
	_, resourceGroup, eipName := pareResourceGroupWithName(eipId, EIP_RESOURCE)
	if len(eipName) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	networkClient := network.NewPublicIPAddressesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	networkClient.Authorizer = region.client.authorizer
	if result, err := networkClient.Get(context.Background(), resourceGroup, eipName, ""); err != nil {
		if result.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if err := jsonutils.Update(&eip, result); err != nil {
		return nil, err
	}
	return &eip, nil
}

func (self *SEipAddress) Associate(instanceId string) error {
	if err := self.region.AssociateEip(self.ID, instanceId); err != nil {
		return err
	}
	return nil
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	if instance, err := region.GetInstance(instanceId); err != nil {
		return err
	} else {
		nicId := instance.Properties.NetworkProfile.NetworkInterfaces[0].ID
		if nic, err := region.GetNetworkInterfaceDetail(nicId); err != nil {
			return err
		} else {
			oldIPConf := nic.Properties.IPConfigurations[0]
			interfaceClinet := network.NewInterfacesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
			interfaceClinet.Authorizer = region.client.authorizer
			InterfaceIPConfiguration := []network.InterfaceIPConfiguration{
				{
					Name: &nic.Name,
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Primary:                   &oldIPConf.Properties.Primary,
						PrivateIPAddress:          &oldIPConf.Properties.PrivateIPAddress,
						PrivateIPAllocationMethod: network.Static,
						PublicIPAddress:           &network.PublicIPAddress{ID: &eipId},
						Subnet:                    &network.Subnet{ID: &oldIPConf.Properties.Subnet.ID},
					},
				},
			}
			params := network.Interface{
				Location: &region.Name,
				InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
					IPConfigurations:     &InterfaceIPConfiguration,
					NetworkSecurityGroup: &network.SecurityGroup{},
				},
			}
			if len(nic.Properties.NetworkSecurityGroup.ID) > 0 {
				params.InterfacePropertiesFormat.NetworkSecurityGroup.ID = &nic.Properties.NetworkSecurityGroup.ID
			}
			_, resourceGroup, nicName := pareResourceGroupWithName(nic.ID, NIC_RESOURCE)
			if result, err := interfaceClinet.CreateOrUpdate(context.Background(), resourceGroup, nicName, params); err != nil {
				return err
			} else if err := result.WaitForCompletion(context.Background(), interfaceClinet.Client); err != nil {
				return err
			}
		}
	}
	return nil
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	if eip, err := region.GetEip(eipId); err != nil {
		return nil, err
	} else {
		eip.region = region
		return eip, nil
	}
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.ID)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	_, resourceGroup, eipName := pareResourceGroupWithName(eipId, EIP_RESOURCE)
	networkClient := network.NewPublicIPAddressesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	networkClient.Authorizer = region.client.authorizer
	if result, err := networkClient.Delete(context.Background(), resourceGroup, eipName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), networkClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SEipAddress) Dissociate() error {
	return self.region.DissociateEip(self.ID)
}

func (region *SRegion) DissociateEip(eipId string) error {
	if eip, err := region.GetEip(eipId); err != nil {
		return err
	} else if len(eip.Properties.IPConfiguration.ID) == 0 {
		log.Debugf("eip %s not associate any instance", eip.Name)
		return nil
	} else {
		if nic, err := region.GetNetworkInterfaceDetail(eip.Properties.IPConfiguration.ID); err != nil {
			return err
		} else {
			oldIPConf := nic.Properties.IPConfigurations[0]
			interfaceClinet := network.NewInterfacesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
			interfaceClinet.Authorizer = region.client.authorizer
			InterfaceIPConfiguration := []network.InterfaceIPConfiguration{
				{
					Name: &nic.Name,
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Primary:                   &oldIPConf.Properties.Primary,
						PrivateIPAddress:          &oldIPConf.Properties.PrivateIPAddress,
						PrivateIPAllocationMethod: network.Static,
						Subnet: &network.Subnet{ID: &oldIPConf.Properties.Subnet.ID},
					},
				},
			}
			params := network.Interface{
				Location: &region.Name,
				InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
					IPConfigurations:     &InterfaceIPConfiguration,
					NetworkSecurityGroup: &network.SecurityGroup{},
				},
			}
			if len(nic.Properties.NetworkSecurityGroup.ID) > 0 {
				params.InterfacePropertiesFormat.NetworkSecurityGroup.ID = &nic.Properties.NetworkSecurityGroup.ID
			}
			_, resourceGroup, nicName := pareResourceGroupWithName(nic.ID, NIC_RESOURCE)
			if result, err := interfaceClinet.CreateOrUpdate(context.Background(), resourceGroup, nicName, params); err != nil {
				return err
			} else if err := result.WaitForCompletion(context.Background(), interfaceClinet.Client); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SEipAddress) GetAssociationExternalId() string {
	if nic, err := self.region.GetNetworkInterfaceDetail(self.Properties.IPConfiguration.ID); err != nil {
		log.Errorf("Failt to find NetworkInterface for eip %s", self.Name)
	} else if len(nic.Properties.VirtualMachine.ID) > 0 {
		return nic.Properties.VirtualMachine.ID
	}
	return ""
}

func (self *SEipAddress) GetAssociationType() string {
	return "server"
}

func (self *SEipAddress) GetBandwidth() int {
	return 0
}

func (self *SEipAddress) GetGlobalId() string {
	return self.ID
}

func (self *SEipAddress) GetId() string {
	return self.ID
}

func (self *SEipAddress) GetInternetChargeType() string {
	return models.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) GetIpAddr() string {
	return self.Properties.IPAddress
}

func (self *SEipAddress) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetMode() string {
	if nic, err := self.region.GetNetworkInterfaceDetail(self.Properties.IPConfiguration.ID); err != nil {
		log.Errorf("Failt to find NetworkInterface for eip %s", self.Name)
	} else if len(nic.Properties.VirtualMachine.ID) > 0 {
		return models.EIP_MODE_INSTANCE_PUBLICIP
	}
	return models.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetName() string {
	return self.Name
}

func (self *SEipAddress) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Succeeded":
		return models.EIP_STATUS_READY
	default:
		log.Errorf("Unknown eip status: %s", self.Properties.ProvisioningState)
		return models.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) IsEmulated() bool {
	return false
}

func (self *SEipAddress) Refresh() error {
	if eip, err := self.region.GetEip(self.ID); err != nil {
		return err
	} else if err := jsonutils.Update(self, eip); err != nil {
		return err
	}
	return nil
}
