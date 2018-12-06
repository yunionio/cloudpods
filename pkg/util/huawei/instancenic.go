package huawei

import (
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/huawei/client/manager"
	"yunion.io/x/pkg/util/netutils"
)

// ===========================================
type Interface struct {
	PortState string    `json:"port_state"`
	FixedIPS  []FixedIP `json:"fixed_ips"`
	NetID     string    `json:"net_id"`
	PortID    string    `json:"port_id"`
	MACAddr   string    `json:"mac_addr"`
}

type FixedIP struct {
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address"`
}

// ===========================================

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
	instanceId := self.instance.GetId()
	subnets, err := self.instance.host.zone.region.getSubnetIdsByInstanceId(instanceId)
	if err != nil || len(subnets) == 0 {
		log.Errorf("getSubnetIdsByInstanceId error: %s", err.Error())
		return nil
	}

	wires, err := self.instance.host.GetIWires()
	if err != nil {
		return nil
	}
	for i := 0; i < len(wires); i += 1 {
		wire := wires[i].(*SWire)
		net := wire.getNetworkById(subnets[0])
		if net != nil {
			return net
		}
	}
	return nil
}

func (self *SRegion) getSubnetIdsByInstanceId(instanceId string) ([]string, error) {
	ctx := &manager.ManagerContext{InstanceManager: self.ecsClient.Servers, InstanceId: instanceId}
	interfaces := make([]Interface, 0)
	err := DoListInContext(self.ecsClient.Interface.ListInContext, ctx, nil, &interfaces)
	if err != nil {
		return nil, err
	}

	subnets := make([]string, 0)
	for _, i := range interfaces {
		for _, net := range i.FixedIPS {
			subnets = append(subnets, net.SubnetID)
		}
	}

	return subnets, nil
}
