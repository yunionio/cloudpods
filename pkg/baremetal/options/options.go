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

package options

import common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"

type BaremetalOptions struct {
	common_options.CommonOptions

	ListenInterface          string `help:"Master net interface of baremetal server" default:"br0"`
	AccessAddress            string `help:"Management IP address of baremetal server, only need to use when multiple address bind to ListenInterface"`
	ListenAddress            string `help:"PXE serve IP address to select when multiple address bind to ListenInterface"`
	TftpRoot                 string `default:"/opt/cloud/yunion/baremetal" help:"tftp root directory"`
	AutoRegisterBaremetal    bool   `default:"true" help:"Automatically create a baremetal instance"`
	BaremetalsPath           string `default:"/opt/cloud/workspace/baremetals" help:"Path for baremetals configuration files"`
	LinuxDefaultRootUser     bool   `default:"false" help:"Default account for linux system is root"`
	IpmiLanPortShared        bool   `default:"false" help:"IPMI Lan port shared or dedicated"`
	Zone                     string `help:"Zone where the agent locates"`
	DhcpLeaseTime            int    `default:"100663296" help:"DHCP lease time in seconds"`  // 0x6000000
	DhcpRenewalTime          int    `default:"67108864" help:"DHCP renewal time in seconds"` // 0x4000000
	EnableGeneralGuestDhcp   bool   `default:"false" help:"Enable DHCP service for general guest, e.g. those on VMware ESXi or Xen"`
	ForceDhcpProbeIpmi       bool   `default:"false" help:"Force DHCP probe IPMI interface network connection"`
	TftpBlockSizeInBytes     int    `default:"1024" help:"tftp block size, default is 1024"`
	TftpMaxTimeoutRetries    int    `default:"50" help:"Maximal tftp timeout retries, default is 50"`
	LengthyWorkerCount       int    `default:"8" help:"Parallel worker count for lengthy tasks"`
	ShortWorkerCount         int    `default:"8" help:"Parallel worker count for short-lived tasks"`
	BaremetalTaskWorkerCount int    `default:"32" help:"Parallel worker count for baremetal tasks"`

	DefaultIpmiPassword       string `help:"Default IPMI passowrd"`
	DefaultStrongIpmiPassword string `help:"Default strong IPMI passowrd"`

	WindowsDefaultAdminUser bool `default:"true" help:"Default account for Windows system is Administrator"`
	// EnableTftpHttpDownload  bool `default:"true" help:"Pxelinux download file through http"`

	CachePath     string `help:"local image cache directory"`
	EnablePxeBoot bool   `help:"Enable DHCP PXE boot" default:"true"`
	BootIsoPath   string `help:"iso boot image path"`

	StatusProbeIntervalSeconds int `help:"interval to probe baremetal status, default is 60 seconds" default:"60"`
	LogFetchIntervalSeconds    int `help:"interval to fetch baremetal log, default is 900 seconds" default:"900"`
	SendMetricsIntervalSeconds int `help:"interval to send baremetal metrics, default is 300 seconds" default:"300"`

	TftpFileMap            map[string]string `help:"map of filename to real file path for tftp"`
	BootLoader             string            `help:"PXE boot loader" default:"grub"`
	EnableGrubTftpDownload bool              `help:"Enable grub using tftp to download kernel and initrd"`
	UseMegaRaidPerccli     bool              `help:"Use MegaRAID perccli" default:"false"`

	NfsBootRootfs string `help:"nfs root fs URL"`

	TftpBootServer   string `help:"customized tftp boot server"`
	TftpBootFilename string `help:"filename of tftp boot loader"`
	TftpBootFilesize int64  `help:"file size of tftp boot loader"`
}

const (
	BOOT_LOADER_SYSLINUX = "syslinux"
	BOOT_LOADER_GRUB     = "grub"
)

var (
	Options BaremetalOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*BaremetalOptions)
	newOpts := newO.(*BaremetalOptions)

	changed := false

	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}

	return changed
}
