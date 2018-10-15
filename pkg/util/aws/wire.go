package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"fmt"
	"yunion.io/x/log"
)

type SWire struct {
	zone *SZone
	vpc  *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.zone.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.zone.GetGlobalId())
}

func (self *SWire) GetStatus() string {
	return "available"
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if self.inetworks == nil {
		// todo: vpc.fetchVSwitches()
		// err := self.vpc.fetchVSwitches()
		// if err != nil {
			return nil, nil
		// }
	}
	return self.inetworks, nil
}

func (self *SWire) GetBandwidth() int {
	panic("implement me")
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	panic("implement me")
}

func (self *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	panic("implement me")
}

func (self *SWire) getNetworkById(networkId string) *SNetwork {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil
	}
	log.Debugf("search for networks %d", len(networks))
	for i := 0; i < len(networks); i += 1 {
		log.Debugf("search %s", networks[i].GetName())
		network := networks[i]
		if network.GetId() == networkId {
			return network.(*SNetwork)
		}
	}
	return nil
}
