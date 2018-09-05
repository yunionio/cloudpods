package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type PublicIPAddressSku struct {
	Name string
}

type PublicIPAddressPropertiesFormat struct {
	PublicIPAddressVersion   string
	IPAddress                string
	PublicIPAllocationMethod string
	ProvisioningState        string
}

type SEipAddress struct {
	region *SRegion

	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat
	Sku        PublicIPAddressSku
}

func (region *SRegion) CreateEIP(bwMbps int) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	networkClient := network.NewPublicIPAddressesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	networkClient.Authorizer = region.client.authorizer
	if _eips, err := networkClient.ListAll(context.Background()); err != nil {
		return nil, err
	} else {
		eips := make([]cloudprovider.ICloudEIP, len(_eips.Values()))
		for i := 0; i < len(eips); i++ {
			eip := SEipAddress{region: region}
			jsonutils.Update(&eip, _eips.Values()[i])
			eips[i] = &eip
		}
		return eips, nil
	}
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eip := SEipAddress{region: region}
	resourceGroup, eipName := PareResourceGroupWithName(eipId, EIP_RESOURCE)
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
	resourceGroup, instanceName := PareResourceGroupWithName(instanceId, INSTANCE_RESOURCE)
	if instance, err := region.GetInstance(resourceGroup, instanceName); err != nil {
		return err
	} else {
		nicId := instance.Properties.NetworkProfile.NetworkInterfaces[0].ID
		resourceGroup, nicName := PareResourceGroupWithName(nicId, NIC_RESOURCE)
		if nic, err := region.getNetworkInterface(resourceGroup, nicName); err != nil {
			return err
		} else {
			oldIPConf := nic.Properties.IPConfigurations[0]
			interfaceClinet := network.NewInterfacesClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
			interfaceClinet.Authorizer = region.client.authorizer
			InterfaceIPConfiguration := []network.InterfaceIPConfiguration{
				network.InterfaceIPConfiguration{
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
				InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
					IPConfigurations: &InterfaceIPConfiguration,
				},
			}
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
	eip := SEipAddress{region: region}
	resourceGroup, eipName := PareResourceGroupWithName(eipId, EIP_RESOURCE)
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

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) Dissociate() error {
	// if err := self.region.DissociateEip(self.AllocationId, self.InstanceId); err != nil {
	// 	return err
	// }
	// return nil
	return cloudprovider.ErrNotImplemented
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return self.ID
}

func (self *SEipAddress) GetAssociationType() string {
	return ""
	// switch self.InstanceType {
	// case EIP_INSTANCE_TYPE_ECS:
	// 	return "server"
	// default:
	// 	log.Fatalf("unsupported type: %s", self.InstanceType)
	// 	return "unsupported"
	// }
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
	return ""
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
	return ""
}

func (self *SEipAddress) GetName() string {
	return self.Name
}

func (self *SEipAddress) GetStatus() string {
	return ""
}

func (self *SEipAddress) IsEmulated() bool {
	return false
	// if self.AllocationId == self.InstanceId {
	// 	// fixed Public IP
	// 	return true
	// } else {
	// 	return false
	// }
}

func (self *SEipAddress) Refresh() error {
	return nil
}
