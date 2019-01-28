package types

import "net"

type SSHConfig struct {
	Username string `json:"username,omitempty"`
	RemoteIP string `json:"ip"`
	Password string `json:"password"`
}

type SDMISystemInfo struct {
	Manufacture string `json:"manufacture"`
	Model       string `json:"model"`
	Version     string `json:"version,omitempty"`
	SN          string `json:"sn"`
}

func (info *SDMISystemInfo) ToIPMISystemInfo() *SIPMISystemInfo {
	return &SIPMISystemInfo{
		Manufacture: info.Manufacture,
		Model:       info.Model,
		Version:     info.Version,
		SN:          info.SN,
	}
}

type SCPUInfo struct {
	Count int    `json:"count"`
	Model string `json:"desc"`
	Freq  int    `json:"freq"`
	Cache int    `json:"cache"`
}

type SDMICPUInfo struct {
	Nodes int `json:"nodes"`
}

type SDMIMemInfo struct {
	Total int `json:"total"`
}

type SNicDevInfo struct {
	Dev   string           `json:"dev"`
	Mac   net.HardwareAddr `json:"mac"`
	Speed int              `json:"speed"`
	Up    bool             `json:"up"`
	Mtu   int              `json:"mtu"`
}

func getMac(macStr string) net.HardwareAddr {
	mac, _ := net.ParseMAC(macStr)
	return mac
}

type SDiskInfo struct {
	Dev        string `json:"dev"`
	Sector     int64  `json:"sector"`
	Block      int64  `json:"block"`
	Size       int64  `json:"size"`
	Rotate     bool   `json:"rotate"`
	ModuleInfo string `json:"module"`
	Kernel     string `json:"kernel"`
	PCIClass   string `json:"pci_class"`
	Driver     string `json:"driver"`
}

type SIPMISystemInfo struct {
	Manufacture string `json:"manufacture"`
	Model       string `json:"model"`
	SN          string `json:"sn"`
	Version     string `json:"version"`
	BSN         string `json:"bsn"`
}

type SIPMILanConfig struct {
	IPSrc   string           `json:"ipsrc"`
	IPAddr  string           `json:"ipaddr"`
	Netmask string           `json:"netmask"`
	Mac     net.HardwareAddr `json:"mac"`
	Gateway string           `json:"gateway"`
}

type SIPMIBootFlags struct {
	Dev string `json:"dev"`
	Sol *bool  `json:"sol"`
}
