package types

type SSHConfig struct {
	Username string `json:"username,omitempty"`
	RemoteIP string `json:"ip"`
	Password string `json:"password"`
}

type DMIInfo struct {
	Manufacture string `json:"manufacture"`
	Model       string `json:"model"`
	Version     string `json:"version,omitempty"`
	SN          string `json:"sn"`
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
	Dev   string `json:"dev"`
	Mac   string `json:"mac"`
	Speed int    `json:"speed"`
	Up    bool   `json:"up"`
	Mtu   int    `json:"mtu"`
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
