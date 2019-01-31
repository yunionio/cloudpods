package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

type BaremetalOptions struct {
	cloudcommon.CommonOptions

	ListenInterface        string `help:"Master net interface of baremetal server" default:"br0"`
	AccessAddress          string `help:"Management IP address of baremetal server, only need to use when multiple address bind to ListenInterface"`
	ListenAddress          string `help:"PXE serve IP address to select when multiple address bind to ListenInterface" default:"0.0.0.0"`
	TftpRoot               string `help:"tftp root directory"`
	AutoRegisterBaremetal  bool   `default:"true" help:"Automatically create a baremetal instance"`
	BaremetalsPath         string `default:"/opt/cloud/workspace/baremetals" help:"Path for baremetals configuration files"`
	LinuxDefaultRootUser   bool   `default:"false" help:"Default account for linux system is root"`
	IpmiLanPortShared      bool   `default:"false" help:"IPMI Lan port shared or dedicated"`
	Zone                   string `help:"Zone where the agent locates"`
	DhcpLeaseTime          int    `default:"100663296" help:"DHCP lease time in seconds"`  // 0x6000000
	DhcpRenewalTime        int    `default:"67108864" help:"DHCP renewal time in seconds"` // 0x4000000
	EnableGeneralGuestDhcp bool   `default:"false" help:"Enable DHCP service for general guest, e.g. those on VMware ESXi or Xen"`
	ForceDhcpProbeIpmi     bool   `default:"false" help:"Force DHCP probe IPMI interface network connection"`
	TftpMaxTimeoutRetries  int    `default:"20" help:"Maximal tftp timeout retries, default is 20"`
	LengthyWorkerCount     int    `default:"8" help:"Parallel worker count for lengthy tasks"`
	ShortWorkerCount       int    `default:"8" help:"Parallel worker count for short-lived tasks"`

	DefaultIpmiPassword       string `help:"Default IPMI passowrd"`
	DefaultStrongIpmiPassword string `help:"Default strong IPMI passowrd"`

	WindowsDefaultAdminUser bool `default:"true" help:"Default account for Windows system is Administrator"`
}

var (
	Options BaremetalOptions
)
