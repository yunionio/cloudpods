package zstack

import (
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/jsonutils"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance

	UUID           string   `json:"uuid"`
	VMInstanceUUID string   `json:"vmInstanceUuid"`
	L3NetworkUUID  string   `json:"l3NetworkUuid"`
	IP             string   `json:"ip"`
	Mac            string   `json:"mac"`
	HypervisorType string   `json:"hypervisorType"`
	IPVersion      int      `json:"ipVersion"`
	UsedIps        []string `json:"usedIps"`
	InternalName   string   `json:"internalName"`
	DeviceID       int      `json:"deviceId"`
	ZStackTime
}

func (nic *SInstanceNic) GetIP() string {
	return nic.IP
}

func (nic *SInstanceNic) GetMAC() string {
	return nic.Mac
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	networks, err := nic.instance.host.zone.region.GetNetworks(nic.instance.host.zone.UUID, "", nic.L3NetworkUUID, "")
	if err != nil {
		log.Errorf("failed to found networks for nic %v error: %v", jsonutils.Marshal(nic).String(), err)
		return nil
	}
	ip, err := netutils.NewIPV4Addr(nic.IP)
	if err != nil {
		log.Errorf("Invalid ip address %s error: %v", nic.IP, err)
		return nil
	}
	for i := 0; i < len(networks); i++ {
		if networks[i].GetIPRange().Contains(ip) {
			l3Network, err := nic.instance.host.zone.region.GetL3Network(nic.instance.host.zone.UUID, "", networks[i].L3NetworkUUID)
			if err != nil {
				log.Errorf("failed to found l3Network for network %v error: %v", jsonutils.Marshal(networks[i]).String(), err)
				return nil
			}
			wire, err := nic.instance.host.zone.region.GetWire(l3Network.L2NetworkUUID)
			if err != nil {
				log.Errorf("failed to found wire for l3Network %v error: %v", jsonutils.Marshal(l3Network).String(), err)
				return nil
			}
			networks[i].wire = wire
			return &networks[i]
		}
	}
	return nil
}
