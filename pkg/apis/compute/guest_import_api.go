// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compute

const (
	SERVER_META_CONVERT_FROM_ESXI      = "__server_convert_from_esxi"
	SERVER_META_CONVERT_FROM_CLOUDPODS = "__server_convert_from_cloudpods"
	SERVER_META_CONVERTED_SERVER       = "__server_converted_to"
	DISK_META_REMOTE_ACCESS_PATH       = "__remote_access_path"
)

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
	MonitorPath string            `json:"monitor_path"`
}

type SLibvirtServerConfig struct {
	MacIp map[string]string `json:"mac_ip"`
}

type SLibvirtHostConfig struct {
	Servers     []SLibvirtServerConfig `json:"servers"`
	XmlFilePath string                 `json:"xml_file_path"`
	MonitorPath string                 `json:"monitor_path"`
	HostIp      string                 `json:"host_ip"`
}

type SLibvirtImportConfig struct {
	Hosts []SLibvirtHostConfig `json:"hosts"`
}
