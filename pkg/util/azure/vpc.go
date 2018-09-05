package azure

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
)

type AddressSpace struct {
	AddressPrefixes []string
}

type SubnetPropertiesFormat struct {
	AddressPrefix     string
	ProvisioningState string
}

type Subnet struct {
	Properties SubnetPropertiesFormat
	Name       string
	ID         string
}

type VirtualNetworkPropertiesFormat struct {
	AddressSpace      AddressSpace
	Subnets           []Subnet
	ProvisioningState string
}

type SVpc struct {
	region *SRegion

	iwires    []cloudprovider.ICloudWire
	secgroups []cloudprovider.ICloudSecurityGroup

	IsDefault bool

	ID         string
	Name       string
	Type       string
	Location   string
	Tags       map[string]string
	Properties VirtualNetworkPropertiesFormat
}

func (self *SVpc) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVpc) GetId() string {
	return self.ID
}

func (self *SVpc) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SVpc) GetGlobalId() string {
	resourceGroup, vpcName := PareResourceGroupWithName(self.ID, VPC_RESOURCE)
	return fmt.Sprintf("resourceGroups/%s/providers/vpc/%s", resourceGroup, vpcName)
}

func (self *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	return self.Properties.AddressSpace.AddressPrefixes[0]
}

func (self *SVpc) Delete() error {
	vpcClient := network.NewVirtualNetworksClientWithBaseURI(self.region.client.baseUrl, self.region.client.subscriptionId)
	vpcClient.Authorizer = self.region.client.authorizer
	resourceGroup, vpcName := PareResourceGroupWithName(self.ID, VPC_RESOURCE)
	if result, err := vpcClient.Delete(context.Background(), resourceGroup, vpcName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), vpcClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SVpc) fetchSecurityGroups() error {
	self.secgroups = make([]cloudprovider.ICloudSecurityGroup, 0)
	networkClient := network.NewSecurityGroupsClientWithBaseURI(self.region.client.baseUrl, self.region.SubscriptionID)
	networkClient.Authorizer = self.region.client.authorizer
	if secgrpList, err := networkClient.ListAll(context.Background()); err != nil {
		return err
	} else {
		for _, secgrp := range secgrpList.Values() {
			securityGroup := SSecurityGroup{vpc: self}
			if *secgrp.Location == self.Location {
				if err := jsonutils.Update(&securityGroup, secgrp); err != nil {
					return err
				}
				self.secgroups = append(self.secgroups, &securityGroup)
			}
		}
	}
	return nil
}

func (self *SVpc) getWire() *SWire {
	if self.iwires == nil {
		self.fetchWires()
	}
	return self.iwires[0].(*SWire)
}

func (self *SVpc) fetchNetworks() error {
	self.Refresh()
	for i := 0; i < len(self.Properties.Subnets); i++ {
		_network := self.Properties.Subnets[i]
		wire := self.getWire()
		network := SNetwork{wire: wire, Name: _network.Name, ID: _network.ID}
		if err := jsonutils.Update(&network, _network); err != nil {
			return err
		}
		wire.addNetwork(&network)
	}
	return nil
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if self.secgroups == nil {
		err := self.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return self.secgroups, nil
}

func (self *SVpc) fetchWires() error {
	networks := make([]cloudprovider.ICloudNetwork, len(self.Properties.Subnets))
	wire := SWire{zone: self.region.izones[0].(*SZone), vpc: self, inetworks: networks}
	for i, _network := range self.Properties.Subnets {
		network := SNetwork{wire: &wire}
		if err := jsonutils.Update(&network, _network); err != nil {
			return err
		}
		networks[i] = &network
	}
	self.iwires = []cloudprovider.ICloudWire{&wire}
	return nil
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		if err := self.fetchNetworks(); err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i++ {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		if err := self.fetchWires(); err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) GetStatus() string {
	if strings.ToLower(self.Properties.ProvisioningState) == "succeeded" {
		return "available"
	}
	return "disabled"
}

func (self *SVpc) Refresh() error {
	resourceGroup, vpcName := PareResourceGroupWithName(self.ID, VPC_RESOURCE)
	vpcClient := network.NewVirtualNetworksClientWithBaseURI(self.region.client.baseUrl, self.region.SubscriptionID)
	vpcClient.Authorizer = self.region.client.authorizer
	if result, err := vpcClient.Get(context.Background(), resourceGroup, vpcName, ""); err != nil {
		return cloudprovider.ErrNotFound
	} else if err := jsonutils.Update(self, result); err != nil {
		return err
	}
	return nil
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) GetNetworks() []Subnet {
	return self.Properties.Subnets
}
