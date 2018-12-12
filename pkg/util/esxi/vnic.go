package esxi

import (
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SVirtualNIC struct {
	SVirtualDevice
}

func NewVirtualNIC(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualNIC {
	return SVirtualNIC{
		NewVirtualDevice(vm, dev, index),
	}
}

func (nic *SVirtualNIC) getVirtualEthernetCard() *types.VirtualEthernetCard {
	card := types.VirtualEthernetCard{}
	if FetchAnonymousFieldValue(nic.dev, &card) {
		return &card
	}
	return nil
}

func (nic *SVirtualNIC) GetIP() string {
	guestIps := nic.vm.getGuestIps()
	if ip, ok := guestIps[nic.GetMAC()]; ok {
		return ip
	}
	log.Warningf("cannot find ip for mac %s", nic.GetMAC())
	return ""
}

func (nic *SVirtualNIC) GetMAC() string {
	return netutils.FormatMacAddr(nic.getVirtualEthernetCard().MacAddress)
}

func (nic *SVirtualNIC) GetINetwork() cloudprovider.ICloudNetwork {
	return nil
}
