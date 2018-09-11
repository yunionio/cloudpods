package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"yunion.io/x/jsonutils"
)

func (self *SRegion) getNetworkInterface(resourceGroup string, nicName string) (*SInstanceNic, error) {
	nic := SInstanceNic{}
	networkClient := network.NewInterfacesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	networkClient.Authorizer = self.client.authorizer
	if _nic, err := networkClient.Get(context.Background(), resourceGroup, nicName, ""); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&nic, _nic); err != nil {
		return nil, err
	}
	return &nic, nil
}

func (self *SRegion) GetNetworkInterfaces() ([]SInstanceNic, error) {
	networkClinet := network.NewInterfacesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	networkClinet.Authorizer = self.client.authorizer
	nics := make([]SInstanceNic, 0)
	if _nics, err := networkClinet.ListAll(context.Background()); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&nics, _nics.Values()); err != nil {
		return nil, err
	}
	return nics, nil
}

func (self *SRegion) isNetworkInstanceNameAvaliable(nicName string) bool {
	networkClinet := network.NewInterfacesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	networkClinet.Authorizer = self.client.authorizer
	resourceGroup, nicName := PareResourceGroupWithName(nicName, NIC_RESOURCE)
	if result, err := networkClinet.Get(context.Background(), resourceGroup, nicName, ""); err != nil || result.Response.StatusCode == 404 {
		return true
	}
	return false
}

func (self *SRegion) CreateNetworkInterface(nicName string, ipAddr string, subnetId string) (*SInstanceNic, error) {
	networkClinet := network.NewInterfacesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	networkClinet.Authorizer = self.client.authorizer
	nic := SInstanceNic{}
	PrivateIPAllocationMethod := network.Static
	if len(ipAddr) == 0 {
		PrivateIPAllocationMethod = network.Dynamic
	}
	_nicName := nicName
	for i := 0; i < 10; i++ {
		nicName = fmt.Sprintf("%s-%d", _nicName, i)
		if self.isNetworkInstanceNameAvaliable(nicName) {
			break
		}
	}

	if nicName == fmt.Sprintf("%s-9", _nicName) {
		return nil, fmt.Errorf("Can not find avaliable nic name")
	}

	IPConfigurations := []network.InterfaceIPConfiguration{
		network.InterfaceIPConfiguration{
			Name: &nicName,
			InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
				PrivateIPAddress:          &ipAddr,
				PrivateIPAddressVersion:   network.IPv4,
				PrivateIPAllocationMethod: PrivateIPAllocationMethod,
				Subnet: &network.Subnet{ID: &subnetId},
			},
		},
	}
	params := network.Interface{
		Name:     &nicName,
		Location: &self.Name,
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &IPConfigurations,
		},
	}
	//log.Debugf("create params: %", jsonutils.Marshal(params).PrettyString())
	resourceGroup, nicName := PareResourceGroupWithName(nicName, NIC_RESOURCE)
	if result, err := networkClinet.CreateOrUpdate(context.Background(), resourceGroup, nicName, params); err != nil {
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), networkClinet.Client); err != nil {
		return nil, err
	} else if _nic, err := result.Result(networkClinet); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&nic, _nic); err != nil {
		return nil, err
	}
	return &nic, nil
}
