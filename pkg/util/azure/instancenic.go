package azure

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/netutils"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
)

type PublicIPAddressPropertiesFormat struct {
	PublicIPAddressVersion string
	IPAddress              string
}

type PublicIPAddress struct {
	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat
}

type InterfaceIPConfigurationPropertiesFormat struct {
	PrivateIPAddress        string
	PrivateIPAddressVersion string
	Subnet                  Subnet
	Primary                 bool
	PublicIPAddress         PublicIPAddress
}

type InterfaceIPConfiguration struct {
	Properties InterfaceIPConfigurationPropertiesFormat
	Name       string
	ID         string
}

type InterfacePropertiesFormat struct {
	IPConfigurations []InterfaceIPConfiguration
	MacAddress       string
	Primary          bool
}

type SInstanceNic struct {
	instance   *SInstance
	ID         string
	Name       string
	Type       string
	Location   string
	Properties InterfacePropertiesFormat
}

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

func (self *SInstanceNic) GetIP() string {
	return self.Properties.IPConfigurations[0].Properties.PrivateIPAddress
}

func (self *SInstanceNic) GetMAC() string {
	mac := self.Properties.MacAddress
	if len(mac) == 0 {
		ip, _ := netutils.NewIPV4Addr(self.GetIP())
		return ip.ToMac("00:16:")
	}
	return strings.Replace(strings.ToLower(mac), "-", ":", -1)
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	if wires, err := self.instance.host.GetIWires(); err != nil {
		log.Errorf("GetINetwork error: %v", err)
		return nil
	} else {
		for i := 0; i < len(wires); i++ {
			wire := wires[i].(*SWire)
			if net := wire.getNetworkById(self.Properties.IPConfigurations[0].Properties.Subnet.ID); net != nil {
				return net
			}
		}
	}
	return nil
}
