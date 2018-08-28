package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SWire struct {
	zone      *SZone
	vpc       *SVpc
	name      string
	id        string
	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.region.GetId())
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId)
}

func (self *SWire) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.region.Name)
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

func (self *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	// vswitchId, err := self.zone.region.createVSwitch(self.zone.ZoneId, self.vpc.VpcId, name, cidr, desc)
	// if err != nil {
	// 	log.Errorf("createVSwitch error %s", err)
	// 	return nil, err
	// }
	// self.inetworks = nil
	// vswitch := self.getNetworkById(vswitchId)
	// if vswitch == nil {
	// 	log.Errorf("cannot find vswitch after create????")
	// 	return nil, cloudprovider.ErrNotFound
	// }
	// return vswitch, nil
	return nil, nil
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
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, len(self.vpc.Properties.Subnets))
		for i, _netwrok := range self.vpc.Properties.Subnets {
			network := SNetwork{wire: self}
			if err := jsonutils.Update(&network, _netwrok); err != nil {
				return nil, err
			}
			self.inetworks[i] = &network
		}
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
			if network.ID == networkId {
				return network
			}
		}
	}
	return nil
}
