package types

import "net"

type SSHConfig struct {
	Username string `json:"username,omitempty"`
	RemoteIP string `json:"ip"`
	Password string `json:"password"`
}

type DMISystemInfo struct {
	Manufacture string `json:"manufacture"`
	Model       string `json:"model"`
	Version     string `json:"version,omitempty"`
	SN          string `json:"sn"`
}

func (info *DMISystemInfo) ToIPMISystemInfo() *IPMISystemInfo {
	return &IPMISystemInfo{
		Manufacture: info.Manufacture,
		Model:       info.Model,
		Version:     info.Version,
		SN:          info.SN,
	}
}

type CPUInfo struct {
	Count int    `json:"count"`
	Model string `json:"desc"`
	Freq  int    `json:"freq"`
	Cache int    `json:"cache"`
}

type DMICPUInfo struct {
	Nodes int `json:"nodes"`
}

type DMIMemInfo struct {
	Total int `json:"total"`
}

type NicDevInfo struct {
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

type DiskInfo struct {
	Dev        string `json:"dev"`
	Sector     int    `json:"sector"`
	Block      int    `json:"block"`
	Size       int    `json:"size"`
	Rotate     bool   `json:"rotate"`
	ModuleInfo string `json:"module"`
	Kernel     string `json:"kernel"`
	PCIClass   string `json:"pci_class"`
	Driver     string `json:"driver"`
}

type IPMISystemInfo struct {
	Manufacture string `json:"manufacture"`
	Model       string `json:"model"`
	SN          string `json:"sn"`
	Version     string `json:"version"`
	BSN         string `json:"bsn"`
}

type IPMILanConfig struct {
	IPSrc   string           `json:"ipsrc"`
	IPAddr  string           `json:"ipaddr"`
	Netmask string           `json:"netmask"`
	Mac     net.HardwareAddr `json:"mac"`
	Gateway string           `json:"gateway"`
}

type IPMIBootFlags struct {
	Dev string `json:"dev"`
	Sol *bool  `json:"sol"`
}
