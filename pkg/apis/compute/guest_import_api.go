package compute

type SImportNic struct {
	Index     int    `json:"index"`
	Bridge    string `json:"bridge"`
	Domain    string `json:"domain"`
	Ip        string `json:"ip"`
	Vlan      int    `json:"vlan"`
	Driver    string `json:"driver"`
	Masklen   int    `json:"masklen"`
	Virtual   bool   `json:"virtual"`
	Manual    bool   `json:"manual"`
	WireId    string `json:"wire_id"`
	NetId     string `json:"net_id"`
	Mac       string `json:"mac"`
	BandWidth int    `json:"bw"`
	Dns       string `json:"dns"`
	Net       string `json:"net"`
	Interface string `json:"interface"`
	Gateway   string `json:"gateway"`
	Ifname    string `json:"ifname"`
}

type SImportDisk struct {
	Index      int    `json:"index"`
	DiskId     string `json:"disk_id"`
	Driver     string `json:"driver"`
	CacheMode  string `json:"cache_mode"`
	AioMode    string `json:"aio_mode"`
	SizeMb     int    `json:"size"`
	Format     string `json:"format"`
	Fs         string `json:"fs"`
	Mountpoint string `json:"mountpoint"`
	Dev        string `json:"dev"`
	TemplateId string `json:"template_id"`
	AccessPath string `json:"AccessPath"`
	Backend    string `json:"Backend"`
}

type SImportGuestDesc struct {
	Id          string            `json:"uuid"`
	Name        string            `json:"name"`
	Nics        []SImportNic      `json:"nics"`
	Disks       []SImportDisk     `json:"disks"`
	Metadata    map[string]string `json:"metadata"`
	MemSizeMb   int               `json:"mem"`
	Cpu         int               `json:"cpu"`
	TemplateId  string            `json:"template_id"`
	ImagePath   string            `json:"image_path"`
	Vdi         string            `json:"vdi"`
	Hypervisor  string            `json:"hypervisor"`
	HostId      string            `json:"host"`
	BootOrder   string            `json:"boot_order"`
	IsSystem    bool              `json:"is_system"`
	Description string            `json:"description"`
}

type SLibvirtServerConfig struct {
	Uuid  string            `json:"uuid"`
	MacIp map[string]string `json:"mac_ip"`
}

type SLibvirtHostConfig struct {
	Servers     []SLibvirtServerConfig `json:"servers"`
	XmlFilePath string                 `json:"xml_file_path"`
	HostIp      string                 `json:"host_ip"`
}

type SLibvirtImportConfig struct {
	Hosts []SLibvirtHostConfig `json:"hosts"`
}
