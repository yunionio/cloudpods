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

package types

import "net"

type SSHConfig struct {
	Username string `json:"username,omitempty"`
	RemoteIP string `json:"remote_ip"`
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
	Count     int    `json:"count"`
	Model     string `json:"desc"`
	Freq      int    `json:"freq"`
	Cache     int    `json:"cache"`
	Microcode string `json:"microcode"`
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

	VlanId    int `json:"vlan_id"`
	SpeedMbps int `json:"speed_mbps"`
}

type SIPMIBootFlags struct {
	Dev string `json:"dev"`
	Sol *bool  `json:"sol"`
}
