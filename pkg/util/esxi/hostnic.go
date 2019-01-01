package esxi

import (
	"yunion.io/x/pkg/tristate"
)

type SHostNicInfo struct {
	Dev     string
	Driver  string
	Mac     string
	Index   int8
	LinkUp  bool
	IpAddr  string
	Mtu     int16
	NicType string
}

func (nic *SHostNicInfo) GetDevice() string {
	return nic.Dev
}

func (nic *SHostNicInfo) GetDriver() string {
	return nic.Driver
}

func (nic *SHostNicInfo) GetMac() string {
	return nic.Mac
}

func (nic *SHostNicInfo) GetIndex() int8 {
	return nic.Index
}

func (nic *SHostNicInfo) IsLinkUp() tristate.TriState {
	if nic.LinkUp {
		return tristate.True
	}
	return tristate.False
}

func (nic *SHostNicInfo) GetIpAddr() string {
	return nic.IpAddr
}

func (nic *SHostNicInfo) GetMtu() int16 {
	return nic.Mtu
}

func (nic *SHostNicInfo) GetNicType() string {
	return nic.NicType
}
