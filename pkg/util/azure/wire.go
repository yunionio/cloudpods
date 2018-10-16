package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SWire struct {
	zone      *SZone
	vpc       *SVpc
	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s/%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId, self.vpc.GetName())
}

func (self *SWire) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SWire) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.vpc.GetName())
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetStatus() string {
	return "available"
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) addNetwork(network *SNetwork) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetId() == network.ID {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, network)
	}
}

func (self *SRegion) createNetwork(vpc *SVpc, subnetName string, cidr string, desc string) (*SNetwork, error) {
	addressSpace := network.AddressSpace{AddressPrefixes: &vpc.Properties.AddressSpace.AddressPrefixes}
	subnets := []network.Subnet{}
	for i := 0; i < len(vpc.Properties.Subnets); i++ {
		subnet := vpc.Properties.Subnets[i]
		subnetPropertiesFormat := network.SubnetPropertiesFormat{AddressPrefix: &subnet.Properties.AddressPrefix}
		subNet := network.Subnet{Name: &subnet.Name, SubnetPropertiesFormat: &subnetPropertiesFormat}
		subnets = append(subnets, subNet)
	}
	subnetPropertiesFormat := network.SubnetPropertiesFormat{AddressPrefix: &cidr}
	subNet := network.Subnet{Name: &subnetName, SubnetPropertiesFormat: &subnetPropertiesFormat}
	subnets = append(subnets, subNet)

	properties := network.VirtualNetworkPropertiesFormat{AddressSpace: &addressSpace, Subnets: &subnets}
	params := network.VirtualNetwork{VirtualNetworkPropertiesFormat: &properties, Location: &vpc.Location}

	networkClient := network.NewVirtualNetworksClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	networkClient.Authorizer = self.client.authorizer
	_, resourceGroup, vpcName := pareResourceGroupWithName(vpc.ID, VPC_RESOURCE)
	result := SNetwork{}
	self.CreateResourceGroup(resourceGroup)
	if resp, err := networkClient.CreateOrUpdate(context.Background(), resourceGroup, vpcName, params); err != nil {
		return nil, err
	} else if err := resp.WaitForCompletion(context.Background(), networkClient.Client); err != nil {
		return nil, err
	} else if net, err := resp.Result(networkClient); err != nil {
		return nil, err
	} else {
		for _, subnet := range *net.Subnets {
			_, _, _subnetName := pareResourceGroupWithName(*subnet.ID, VPC_RESOURCE)
			if _subnetName == subnetName {
				if err := jsonutils.Update(&result, subnet); err != nil {
					return nil, err
				} else {
					return &result, nil
				}
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	if network, err := self.zone.region.createNetwork(self.vpc, name, cidr, desc); err != nil {
		return nil, err
	} else {
		network.wire = self
		return network, nil
	}
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(networks); i += 1 {
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if err := self.vpc.fetchNetworks(); err != nil {
		return nil, err
	}
	return self.inetworks, nil
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SWire) getNetworkById(networkId string) *SNetwork {
	if networks, err := self.GetINetworks(); err != nil {
		log.Errorf("getNetworkById error: %v", err)
		return nil
	} else {
		log.Debugf("search for networks %d", len(networks))
		for i := 0; i < len(networks); i++ {
			network := networks[i].(*SNetwork)
			if networkId == network.ID {
				return network
			}
		}
	}
	return nil
}
