package azure

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClassicInstanceNic struct {
	instance *SClassicInstance

	ID       string
	IP       string
	Name     string
	Type     string
	Location string
}

func (self *SClassicInstanceNic) GetIP() string {
	return self.IP
}

func (self *SClassicInstanceNic) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicInstanceNic) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(self.GetIP())
	return ip.ToMac("00:16:")
}

func (self *SClassicInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SClassicInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		log.Errorf("GetINetwork error: %v", err)
		return nil
	}
	for i := 0; i < len(wires); i++ {
		wire := wires[i].(*SClassicWire)
		if network := wire.getNetworkById(self.ID); network != nil {
			return network
		}
	}
	return nil
}
