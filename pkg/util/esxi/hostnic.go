package esxi

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

func (nic *SHostNicInfo) IsLinkUp() bool {
	return nic.LinkUp
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
