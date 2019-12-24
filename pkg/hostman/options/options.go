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

type SHostOptions struct {
	common_options.CommonOptions

	HostType        string   `help:"Host server type, either hypervisor or kubelet" default:"hypervisor"`
	ListenInterface string   `help:"Master address of host server"`
	BridgeDriver    string   `help:"Bridge driver, bridge or openvswitch" default:"openvswitch"`
	Networks        []string `help:"Network interface information"`
	Rack            string   `help:"Rack of host (optional)"`
	Slots           string   `help:"Slots of host (optional)"`
	Hostname        string   `help:"Customized host name"`

	ServersPath    string `help:"Path for virtual server configuration files" default:"/opt/cloud/workspace/servers"`
	ImageCachePath string `help:"Path for storing image caches" default:/opt/cloud/workspace/disks/image_cache"`
	// ImageCacheLimit int    `help:"Maximal storage space for image caching, in GB" default:"20"`
	AgentTempPath  string `help:"Path for ESXi agent"`
	AgentTempLimit int    `help:"Maximal storage space for ESXi agent, in GB" default:"10"`

	RecycleDiskfile         bool `help:"Recycle instead of remove deleted disk file" default:"true"`
	RecycleDiskfileKeepDays int  `help:"How long recycled files kept, default 28 days" default:"28"`

	EnableTemplateBacking    bool `help:"Use template as backing file"`
	AutoMergeBackingTemplate bool `help:"Automatically stream merging backing file"`
	AutoMergeDelaySeconds    int  `help:"Seconds to delay mergeing backing file after VM start, default 15 minutes" default:"900"`
	EnableFallocateDisk      bool `help:"Automatically allocate all spaces using fallocate"`

	EnableMonitor  bool `help:"Enable monitor"`
	ReportInterval int  `help:"Report interval in seconds" default:"60"`

	EnableTcBwlimit     bool `help:"Enable linux tc bandwidth limit"`
	BwDownloadBandwidth int  `help:"Default ingress bandwidth in mbit (0 disabled)" default:"10"`

	DnsServer       string `help:"Address of host DNS server"`
	DnsServerLegacy string `help:"Deprecated Address of host DNS server"`

	ChntpwPath           string `help:"path to chntpw tool" default:"/usr/local/bin/chntpw.static"`
	OvmfPath             string `help:"Path to OVMF.fd" default:"/opt/cloud/contrib/OVMF.fd"`
	LinuxDefaultRootUser bool   `help:"Default account for linux system is root"`

	BlockIoScheduler string `help:"Block IO scheduler, deadline or cfq" default:"deadline"`
	EnableKsm        bool   `help:"Enable Kernel Same Page Merging"`
	HugepagesOption  string `help:"Hugepages option: disable|native|transparent" default:"transparent"`
	EnableQmpMonitor bool   `help:"Enable qmp monitor" default:"true"`

	PrivatePrefixes []string `help:"IPv4 private prefixes"`
	LocalImagePath  []string `help:"Local image storage paths"`
	SharedStorages  []string `help:"Path of shared storages"`

	DefaultQemuVersion string `help:"Default qemu version" default:"2.12.1"`

	DhcpRelay       []string `help:"DHCP relay upstream"`
	DhcpLeaseTime   int      `default:"100663296" help:"DHCP lease time in seconds"`
	DhcpRenewalTime int      `default:"67108864" help:"DHCP renewal time in seconds"`

	TunnelPaddingBytes int64 `help:"Specify tunnel padding bytes" default:"0"`

	CheckSystemServices bool `help:"Check system services (ntpd, telegraf) on startup" default:"true"`

	DhcpServerPort int    `help:"Host dhcp server bind port" default:"67"`
	DiskIsSsd      bool   `default:"false"`
	FetcherfsPath  string `default:"/opt/yunion/fetchclient/bin/fetcherfs" help:"Fuse fetcherfs path"`

	DefaultImageSaveFormat string `default:"qcow2" help:"Default image save format, default is qcow2, canbe vmdk"`

	DefaultReadBpsPerCpu   int  `default:"163840000" help:"Default read bps per cpu for hard IO limit"`
	DefaultReadIopsPerCpu  int  `default:"1250" help:"Default read iops per cpu for hard IO limit"`
	DefaultWriteBpsPerCpu  int  `default:"54525952" help:"Default write bps per cpu for hard IO limit"`
	DefaultWriteIopsPerCpu int  `default:"416" help:"Default write iops per cpu for hard IO limit"`
	SetVncPassword         bool `default:"true" help:"Auto set vnc password after monitor connected"`
	UseBootVga             bool `default:"false" help:"Use boot VGA GPU for guest"`

	EnableCpuBinding         bool   `default:"true" help:"Enable cpu binding and rebalance"`
	EnableOpenflowController bool   `default:"false"`
	K8sClusterCidr           string `default:"10.43.0.0/16" help:"Kubernetes cluster IP range"`

	PingRegionInterval     int      `default:"60" help:"interval to ping region, deefault is 1 minute"`
	ManageNtpConfiguration bool     `default:"true"`
	LogSystemdUnits        []string `help:"Systemd units log collected by fluent-bit"`
	BandwidthLimit         int      `default:"50" help:"Bandwidth upper bound when migrating disk image in MB/sec"`

	SnapshotDirSuffix  string `help:"Snapshot dir name equal diskId concat snapshot dir suffix" default:"_snap"`
	SnapshotRecycleDay int    `default:"1" help:"Snapshot Recycle delete Duration day"`

	EnableTelegraf          bool `default:"true" help:"enable send monitoring data to telegraf"`
	WindowsDefaultAdminUser bool `default:"true" help:"Default account for Windows system is Administrator"`

	HostCpuPassthrough bool `default:"true" help:"if it is true, set qemu cpu type as -cpu host, otherwise, qemu64. default is true"`

	MaxReservedMemory int `default:"10240" help:"host reserved memory"`

	DeployServerSocketPath    string `help:"Deploy server listen socket path" default:"/var/run/deploy.sock"`
	DefaultRequestWorkerCount int    `default:"8" help:"default request worker count"`
	EnableRemoteExecutor      bool   `help:"Enable remote executor" default:"false"`
	ExecutorSocketPath        string `help:"Executor socket path" default:"/var/run/exec.sock"`
	CommonConfigFile          string `help:"common config file for container"`
}

var HostOptions SHostOptions
