package qcloud

import (
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	ipAddr   string
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(self.ipAddr)
	return ip.ToMac("00:16:")
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	networkId := self.instance.VirtualPrivateCloud.SubnetId
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		return nil
	}
	for i := 0; i < len(wires); i++ {
		wire := wires[i].(*SWire)
		net := wire.getNetworkById(networkId)
		if net != nil {
			return net
		}
	}
	return nil
}
