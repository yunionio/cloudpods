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

import (
	"net"
	"strings"
)

type SSHConfig struct {
	Username string `json:"username,omitempty"`
	RemoteIP string `json:"remote_ip"`
	Password string `json:"password"`
}

const (
	OEM_NAME_DELL       = "dell"
	OEM_NAME_HPE        = "hpe"
	OEM_NAME_HP         = "hp"
	OEM_NAME_HUAWEI     = "huawei"
	OEM_NAME_INSPUR     = "inspur"
	OEM_NAME_LENOVO     = "lenovo"
	OEM_NAME_FOXCONN    = "foxconn"
	OEM_NAME_QEMU       = "qemu"
	OEM_NAME_SUPERMICRO = "supermicro"
)

var (
	OEM_NAMES = []string{
		OEM_NAME_DELL,
		OEM_NAME_HPE,
		OEM_NAME_HP,
		OEM_NAME_HUAWEI,
		OEM_NAME_INSPUR,
		OEM_NAME_LENOVO,
		OEM_NAME_FOXCONN,
		OEM_NAME_QEMU,
		OEM_NAME_SUPERMICRO,
	}
)

func ManufactureOemName(manufacture string) string {
	manufacture = strings.ToLower(strings.TrimSpace(manufacture))
	for _, oem := range OEM_NAMES {
		if strings.Contains(manufacture, oem) {
			return oem
		}
	}
	return manufacture
}

type SSystemInfo struct {
	Manufacture string `json:"manufacture"`
	Model       string `json:"model"`
	SN          string `json:"sn"`
	Version     string `json:"version,omitempty"`
	BSN         string `json:"bsn"`

	OemName string `json:"oem_name"`
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
