package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
)

type SWire struct {
	zone *SZone
	vpc  *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	panic("implement me")
}

func (self *SWire) GetName() string {
	panic("implement me")
}

func (self *SWire) GetGlobalId() string {
	panic("implement me")
}

func (self *SWire) GetStatus() string {
	panic("implement me")
}

func (self *SWire) Refresh() error {
	panic("implement me")
}

func (self *SWire) IsEmulated() bool {
	panic("implement me")
}

func (self *SWire) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	panic("implement me")
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	panic("implement me")
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	panic("implement me")
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

