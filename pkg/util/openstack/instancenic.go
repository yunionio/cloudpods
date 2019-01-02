package openstack

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	MacAddr  string `json:"OS-EXT-IPS-MAC:mac_addr"`
	Version  int    `json:"version"`
	Addr     string `json:"addr"`
	Type     string `json:"OS-EXT-IPS:type"`
}

func (nic *SInstanceNic) GetIP() string {
	return nic.Addr
}

func (nic *SInstanceNic) GetMAC() string {
	return nic.MacAddr
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	ports, err := nic.instance.getRegion().GetPorts(nic.MacAddr)
	if err == nil {
		for i := 0; i < len(ports); i++ {
			for j := 0; j < len(ports[i].FixedIps); j++ {
				if ports[i].FixedIps[j].IpAddress == nic.Addr {
					network, err := nic.instance.getRegion().GetNetwork(ports[i].FixedIps[j].SubnetID)
					if err != nil {
						return nil
					}
					wires, err := nic.instance.getZone().GetIWires()
					if err != nil {
						return nil
					}
					for k := 0; k < len(wires); k++ {
						wire := wires[i].(*SWire)
						if net, _ := wire.GetINetworkById(network.ID); net != nil {
							return net
						}
					}
					return nil
				}
			}
		}
	}
	return nil
}
