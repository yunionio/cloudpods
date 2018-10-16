package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
)

type SNetwork struct {
	wire *SWire

	AvailableIpAddressCount int
	ID                      string
	Name                    string
	Properties              SubnetPropertiesFormat

	// Status string
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetId() string {
	return self.ID
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetStatus() string {
	if strings.ToLower(self.Properties.ProvisioningState) == "succeeded" {
		return "available"
	}
	return "disabled"
}

func (self *SNetwork) Delete() error {
	vpc := self.wire.vpc
	addressSpace := network.AddressSpace{AddressPrefixes: &vpc.Properties.AddressSpace.AddressPrefixes}
	subnets := []network.Subnet{}
	for i := 0; i < len(vpc.Properties.Subnets); i++ {
		subnet := vpc.Properties.Subnets[i]
		if subnet.Name == self.Name {
			continue
		}
		subnetPropertiesFormat := network.SubnetPropertiesFormat{AddressPrefix: &subnet.Properties.AddressPrefix}
		subNet := network.Subnet{Name: &subnet.Name, SubnetPropertiesFormat: &subnetPropertiesFormat}
		subnets = append(subnets, subNet)
	}

	properties := network.VirtualNetworkPropertiesFormat{AddressSpace: &addressSpace, Subnets: &subnets}
	params := network.VirtualNetwork{VirtualNetworkPropertiesFormat: &properties, Location: &vpc.Location}

	region := self.wire.vpc.region
	networkClient := network.NewVirtualNetworksClientWithBaseURI(region.client.baseUrl, region.SubscriptionID)
	networkClient.Authorizer = region.client.authorizer
	_, resourceGroup, vpcName := pareResourceGroupWithName(vpc.ID, VPC_RESOURCE)
	region.CreateResourceGroup(resourceGroup)
	if result, err := networkClient.CreateOrUpdate(context.Background(), resourceGroup, vpcName, params); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), networkClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	return pref.MaskLen
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetServerType() string {
	return models.SERVER_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	if new, err := self.wire.zone.region.GetNetworkDetail(self.ID); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
	return nil
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}
