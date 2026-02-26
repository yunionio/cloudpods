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

import (
	"fmt"
	"os"
	"path/filepath"

	"yunion.io/x/log"
	"yunion.io/x/structarg"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/ovnutils"
)

type SHostBaseOptions struct {
	common_options.HostCommonOptions

	ManageNtpConfiguration bool `help:"host agent manage ntp service on host" default:"true"`

	DisableSecurityGroup bool `help:"disable security group" default:"false"`

	HostCpuPassthrough              bool  `default:"true" help:"if it is true, set qemu cpu type as -cpu host, otherwise, qemu64. default is true"`
	LiveMigrateCpuThrottleMax       int64 `default:"99" help:"live migrate auto converge cpu throttle max"`
	LiveMigrateCpuThrottleInitial   int64 `default:"60" help:"live migrate auto convert cpu throttle initial"`
	LiveMigrateCpuThrottleIncrement int64 `default:"20" help:"live migrate auto convert cpu throttle increment"`

	DefaultQemuVersion string `help:"Default qemu version" default:"10.0.7"`
	NoHpet             bool   `help:"Disable qemu hpet timer" default:"true"`

	CdromCount  int `help:"cdrom count" default:"1"`
	FloppyCount int `help:"floppy count" default:"1"`

	DisableLocalVpc bool `help:"disable local VPC support" default:"false"`

	EnableDmesgCollect bool `default:"true" help:"Enable dmesg collect or not, default true"`

	DhcpLeaseTime   int `default:"100663296" help:"DHCP lease time in seconds"`
	DhcpRenewalTime int `default:"67108864" help:"DHCP renewal time in seconds"`

	Dhcp6RouterAdvertisementIntervalSecs int `default:"3" help:"DHCPv6 router advertisement interval in seconds, default 3 seconds"`
	Dhcp6RouterAdvertisementAttempts     int `default:"3" help:"DHCPv6 router advertisement attempts, default 3 attempts"`
	Dhcp6RouterLifetimeSeconds           int `default:"9000" help:"DHCPv6 router lifetime in seconds, default 9000 seconds"`

	Ext4LargefileSizeGb int `default:"4096" help:"Use largefile options when the ext4 fs greater than this size"`
	Ext4HugefileSizeGb  int `default:"512" help:"Use huge options when the ext4 fs greater than this size"`

	ImageCacheExpireDays        int  `help:"Image cache expire duration in days" default:"30"`
	ImageCacheCleanupPercentage int  `help:"The cleanup threshold ratio of image cache size v.s. total storage size" default:"12"`
	ImageCacheCleanupOnStartup  bool `help:"Cleanup image cache on host startup" default:"false"`
	ImageCacheCleanupDryRun     bool `help:"Dry run cleanup image cache" default:"false"`

	TelegrafKafkaOutputTopic         string `json:"telegraf_kafka_output_topic" help:"telegraf kafka output topic"`
	TelegrafKafkaOutputSaslUsername  string `json:"telegraf_kafka_output_sasl_username" help:"telegraf kafka output sasl_username"`
	TelegrafKafkaOutputSaslPassword  string `json:"telegraf_kafka_output_sasl_password" help:"telegraf kafka output sasl_password"`
	TelegrafKafkaOutputSaslMechanism string `json:"telegraf_kafka_output_sasl_mechanism" help:"telegraf kafka output sasl_mechanism"`

	BackupTaskWorkerCount int `default:"3" help:"backup task worker count"`
}

type SHostOptions struct {
	common_options.EtcdOptions

	SHostBaseOptions

	CommonConfigFile string `help:"common config file for container"`
	LocalConfigFile  string `help:"local config file" default:"/etc/yunion/host_local.conf"`

	HostType        string   `help:"Host server type, either hypervisor or container" default:"hypervisor" choices:"hypervisor|container"`
	ListenInterface string   `help:"Master address of host server"`
	BridgeDriver    string   `help:"Bridge driver, bridge or openvswitch" default:"openvswitch"`
	Networks        []string `help:"Network interface information"`
	Rack            string   `help:"Rack of host (optional)"`
	Slots           string   `help:"Slots of host (optional)"`
	Hostname        string   `help:"Customized host name"`

	ServersPath         string `help:"Path for virtual server configuration files" default:"/opt/cloud/workspace/servers"`
	ImageCachePath      string `help:"Path for storing image caches" default:"/opt/cloud/workspace/disks/image_cache"`
	MemorySnapshotsPath string `help:"Path for memory snapshot stat files" default:"/opt/cloud/workspace/memory_snapshots"`
	HostFilesPath       string `help:"Path for host files" default:"/opt/cloud/workspace/host_files"`
	// ImageCacheLimit int    `help:"Maximal storage space for image caching, in GB" default:"20"`
	AgentTempPath  string `help:"Path for ESXi agent"`
	AgentTempLimit int    `help:"Maximal storage space for ESXi agent, in GB" default:"10"`

	RecycleDiskfile         bool `help:"Recycle instead of remove deleted disk file" default:"true"`
	RecycleDiskfileKeepDays int  `help:"How long recycled files kept, default 28 days" default:"28"`
	AlwaysRecycleDiskfile   bool `help:"Always recycle disk files, no matter what" default:"true"`

	ZeroCleanDiskData bool `help:"Clean disk data by writing zeros" default:"false"`

	EnableTemplateBacking    bool `help:"Use template as backing file"`
	AutoMergeBackingTemplate bool `help:"Automatically stream merging backing file"`
	AutoMergeDelaySeconds    int  `help:"Seconds to delay mergeing backing file after VM start, default 15 minutes" default:"900"`
	EnableFallocateDisk      bool `help:"Automatically allocate all spaces using fallocate"`

	EnableMonitor  bool `help:"Enable monitor"`
	ReportInterval int  `help:"Report interval in seconds" default:"60"`

	BwDownloadBandwidth int `help:"Default ingress bandwidth in mbit (0 disabled)" default:"1000"`

	DnsServer       string `help:"Address of host DNS server"`
	DnsServerLegacy string `help:"Deprecated Address of host DNS server"`

	ChntpwPath          string `help:"path to chntpw tool" default:"/usr/local/bin/chntpw.static"`
	OvmfPath            string `help:"Path to OVMF.fd" default:"/opt/cloud/contrib/OVMF.fd"`
	OvmfVarsPath        string `help:"Path to OVMF_VARS.fd" default:"/opt/cloud/contrib/OVMF_VARS.fd"`
	SecbootOvmfPath     string `help:"Path to secboot ovmf fd" default:"/opt/cloud/contrib/OVMF_CODE_4M.secboot.fd"`
	SecbootOvmfVarsPath string `help:"Path to secboot ovmf vars fd" default:"/opt/cloud/contrib/OVMF_VARS_4M.fd"`

	LinuxDefaultRootUser    bool `help:"Default account for linux system is root"`
	WindowsDefaultAdminUser bool `default:"true" help:"Default account for Windows system is Administrator"`

	BlockIoScheduler    string `help:"HDD Block IO scheduler, deadline or cfq" default:"deadline"`
	SsdBlockIoScheduler string `help:"SSD Block IO scheduler, none deadline or cfq" default:"none"`
	EnableKsm           bool   `help:"Enable Kernel Same Page Merging"`
	HugepagesOption     string `help:"Hugepages option: disable|native|transparent" default:"transparent"`
	HugepageSizeMb      int    `help:"hugepage size mb default 1G" default:"1024"`

	// PrivatePrefixes []string `help:"IPv4 private prefixes"`
	LocalImagePath  []string `help:"Local image storage paths"`
	SharedStorages  []string `help:"Path of shared storages"`
	LVMVolumeGroups []string `help:"LVM Volume Groups(vgs)"`

	DhcpRelay  []string `help:"DHCP relay upstream"`
	Dhcp6Relay []string `help:"DHCPv6 relay upstream"`

	TunnelPaddingBytes int64 `help:"Specify tunnel padding bytes" default:"0"`

	CheckSystemServices bool `help:"Check system services (ntpd, telegraf) on startup" default:"true"`

	DhcpServerPort     int    `help:"Host dhcp server bind port" default:"67"`
	Dhcp6ServerPort    int    `help:"Host dhcp6 server bind port" default:"547"`
	FetcherfsPath      string `default:"/opt/yunion/fetchclient/bin/fetcherfs" help:"Fuse fetcherfs path"`
	FetcherfsBlockSize int    `default:"16" help:"Fuse fetcherfs fetch chunk_size MB"`

	DefaultImageSaveFormat string `default:"qcow2" help:"Default image save format, default is qcow2, canbe vmdk"`

	DefaultReadBpsPerCpu   int  `default:"163840000" help:"Default read bps per cpu for hard IO limit"`
	DefaultReadIopsPerCpu  int  `default:"1250" help:"Default read iops per cpu for hard IO limit"`
	DefaultWriteBpsPerCpu  int  `default:"54525952" help:"Default write bps per cpu for hard IO limit"`
	DefaultWriteIopsPerCpu int  `default:"416" help:"Default write iops per cpu for hard IO limit"`
	SetVncPassword         bool `default:"true" help:"Auto set vnc password after monitor connected"`
	UseBootVga             bool `default:"false" help:"Use boot VGA GPU for guest"`

	EnableStrictCpuBind         bool   `default:"false" help:"Enable strict cpu bind, one vcpu bind one pcpu"`
	EnableHostAgentNumaAllocate bool   `default:"true" help:"Enable host agent numa allocate"`
	EnableCpuBinding            bool   `default:"true" help:"Enable cpu binding and rebalance"`
	EnableOpenflowController    bool   `default:"false"`
	BootVgaPciAddr              string `help:"Specific boot vga pci addr incase detect wrong device"`

	PingRegionInterval int      `default:"60" help:"interval to ping region, default is 1 minute"`
	LogSystemdUnits    []string `help:"Systemd units log collected by fluent-bit"`
	// 更改默认带宽限速为400GBps, qiujian
	BandwidthLimit int `default:"400000" help:"Bandwidth upper bound when migrating disk image in MB/sec, default 400GBps"`
	// 热迁移带宽，预期不低于8MBps, 1G Memory takes 128 seconds
	MigrateExpectRate        int `default:"32" help:"Expected memory migration rate in MB/sec, default 32MBps"`
	MinMigrateTimeoutSeconds int `default:"30" help:"minimal timeout for a migration process, default 30 seconds"`

	SnapshotDirSuffix  string `help:"Snapshot dir name equal diskId concat snapshot dir suffix" default:"_snap"`
	SnapshotRecycleDay int    `default:"1" help:"Snapshot Recycle delete Duration day"`

	EnableTelegraf bool `default:"true" help:"enable send monitoring data to telegraf"`

	DisableSetCgroup bool `default:"false" help:"disable cgroup for guests"`

	MaxReservedMemory int `default:"10240" help:"host reserved memory"`

	DefaultRequestWorkerCount int `default:"8" help:"default request worker count"`
	ImageCacheWorkerCount     int `default:"8" help:"default request worker count"`
	ContainerStartWorkerCount int `default:"1" help:"container start worker count"`
	ContainerStopWorkerCount  int `default:"1" help:"container stop worker count"`

	AllowSwitchVMs bool `help:"allow machines run as switch (spoof mac)" default:"true"`
	AllowRouterVMs bool `help:"allow machines run as router (spoof ip)" default:"true"`

	SdnPidFile        string `help:"pid file for sdnagent" default:"$SDN_PID_FILE|/var/run/onecloud/sdnagent.pid"`
	SdnSocketPath     string `help:"sdnagent listen socket path" default:"/var/run/onecloud/sdnagent.sock"`
	SdnEnableGuestMan bool   `help:"enable guest network manager in sdnagent" default:"$SDN_ENABLE_GUEST_MAN|true"`
	SdnEnableEipMan   bool   `help:"enable eip network manager in sdnagent" default:"$SDN_ENABLE_EIP_MAN|false"`
	SdnEnableTcMan    bool   `help:"enable TC manager in sdnagent" default:"$SDN_ENABLE_TC_MAN|true"`

	SdnEnableTapMan bool   `help:"enable tap service" default:"$SDN_ENABLE_TAP_MAN|true"`
	TapBridgeName   string `help:"bridge name for tap service" default:"brtap"`

	SdnAllowConntrackInvalid       bool `help:"allow packets marked by conntrack as INVALID to pass" default:"$SDN_ALLOW_CONNTRACK_INVALID|false"`
	SdnFetchDataFromComputeService bool `help:"fetch network releated data from compute service" default:"$SDN_FETCH_DATA_FROM_COMPUTE_SERVICE|true"`

	ovnutils.SOvnOptions

	// EnableRemoteExecutor bool `help:"Enable remote executor" default:"false"`
	HostHealthTimeout int `help:"host health timeout" default:"30"`
	HostLeaseTimeout  int `help:"lease timeout" default:"10"`

	SyncStorageInfoDurationSecond int `help:"sync storage size duration, unit is second, default is every 2 minutes" default:"120"`

	DisableProbeKubelet bool   `help:"Disable probe kubelet config" default:"false"`
	KubeletRunDirectory string `help:"Kubelet config file path" default:"/var/lib/kubelet"`

	DisableKVM bool `help:"force disable KVM" default:"false" json:"disable_kvm"`

	DisableGPU          bool     `help:"force disable GPU detect" default:"false" json:"disable_gpu"`
	DisableCustomDevice bool     `help:"force disable custom pci device detect" default:"false" json:"disable_custom_device"`
	DisableUSB          bool     `help:"force disable USB detect" default:"true" json:"disable_usb"`
	SRIOVNics           []string `help:"nics enable sriov" json:"sriov_nics"`
	OvsOffloadNics      []string `help:"nics enable ovs offload" json:"ovs_offload_nics"`
	PTNVMEConfigs       []string `help:"passthrough nvme disk pci address and size"`
	AMDVgpuPFs          []string `help:"amd vgpu pf pci addresses"`
	NVIDIAVgpuPFs       []string `help:"nvidia vgpu pf pci addresses"`

	EthtoolEnableGso bool `help:"use ethtool to turn on or off GSO(generic segment offloading)" default:"true" json:"ethtool_enable_gso"`

	EthtoolEnableGsoInterfaces  []string `help:"use ethtool to turn on GSO for the specific interfaces" json:"ethtool_enable_gso_interfaces"`
	EthtoolDisableGsoInterfaces []string `help:"use ethtool to turn off GSO for the specific interfaces" json:"ethtool_disable_gso_interfaces"`

	EnableVmUuid bool `help:"enable vm UUID" default:"true" json:"enable_vm_uuid"`

	EnableVirtioRngDevice bool `help:"enable qemu virtio-rng device" default:"true"`

	RestrictQemuImgConvertWorker bool `help:"restrict qemu-img convert worker" default:"false"`

	DefaultLiveMigrateDowntime float32 `help:"allow downtime in seconds for live migrate" default:"5.0"`

	LocalBackupStoragePath string `help:"path for mounting backup nfs storage" default:"/opt/cloud/workspace/backupstorage"`
	LocalBackupTempPath    string `help:"the local temporary directory for backup" default:"/opt/cloud/workspace/run/backups"`

	BinaryMemcleanPath string `help:"execute binary memclean path" default:"/opt/yunion/bin/memclean"`
	BinarySwtpmPath    string `help:"swtpm binary path" default:"/usr/bin/swtpm"`

	MaxHotplugVCpuCount int    `help:"maximal possible vCPU count that the platform kvm supports"`
	PcieRootPortCount   int    `help:"pcie root port count" default:"2"`
	EnableQemuDebugLog  bool   `help:"enable qemu debug logs" default:"false"`
	ResetDiskTmpDir     string `help:"auto reset disk after guest shutdown will write disk to tmpdir"`

	GuestMaxMemSizeMb int `help:"guest maximal mem size, default 0 is not set" default:"0"`

	// container related endpoint
	// EnableContainerRuntime   bool   `help:"enable container runtime" default:"false"`
	ContainerRuntimeEndpoint                 string `help:"endpoint of container runtime service" default:"unix:///var/run/onecloud/containerd/containerd.sock"`
	ContainerDeviceConfigFile                string `help:"container device configuration file path"`
	LxcfsPath                                string `help:"lxcfs directory path" default:"/var/lib/lxcfs"`
	ContainerSystemCpufreqSimulateConfigFile string `help:"container system cpu simulate config file path" default:"/etc/yunion/container_cpufreq_simulate.conf"`
	EnableRealtimeCpufreqSimulate            bool   `help:"realtime cpufreq simulate" default:"true"`
	RealtimeCpufreqSimulateInterval          int    `help:"realtime cpufreq simulate interval(second)" default:"2"`

	EnableCudaMPS        bool   `help:"enable cuda mps" default:"false"`
	CudaMPSPipeDirectory string `help:"cuda mps pipe dir" default:"/tmp/nvidia-mps/pipe"`
	CudaMPSLogDirectory  string `help:"cuda mps log dir" default:"/tmp/nvidia-mps/log"`
	CudaMPSReplicas      int    `help:"cuda mps replicas" default:"10"`

	SkipCheckKernelMods []string `help:"skip check kernel modules"`

	EnableContainerAscendNPU bool `help:"enable container npu" default:"false"`

	EnableDirtyRecoverySeconds int  `help:"Seconds to delay enable dirty guests recovery feature, default 15 minutes" default:"900"`
	EnableContainerCniPortmap  bool `help:"Use container cni portmap plugin" default:"false"`
	DisableReconcileContainer  bool `help:"disable reconcile container" default:"false"`
}

func (o SHostOptions) HostLocalNetconfPath(br string) string {
	return filepath.Join(o.ServersPath, fmt.Sprintf("host_local_netconf_%s.json", br))
}

var (
	HostOptions SHostOptions
)

func Parse() SHostOptions {
	var hostOpts SHostOptions
	common_options.ParseOptions(&hostOpts, os.Args, "host.conf", "host")
	if len(hostOpts.CommonConfigFile) > 0 && fileutils2.Exists(hostOpts.CommonConfigFile) {
		commonCfg := &SHostBaseOptions{}
		commonCfg.Config = hostOpts.CommonConfigFile
		common_options.ParseOptions(commonCfg, []string{os.Args[0]}, "common.conf", "host")
		baseOpt := hostOpts.BaseOptions.BaseOptions
		hostOpts.SHostBaseOptions = *commonCfg
		// keep base options
		hostOpts.BaseOptions.BaseOptions = baseOpt
	}
	if len(hostOpts.LocalConfigFile) > 0 && fileutils2.Exists(hostOpts.LocalConfigFile) {
		log.Infof("Use local configuration file: %s", hostOpts.Config)
		parser, err := structarg.NewArgumentParser(&hostOpts, "", "", "")
		if err != nil {
			log.Fatalf("fail to create local parse %s", err)
		}
		err = parser.ParseFile(hostOpts.LocalConfigFile)
		if err != nil {
			log.Fatalf("Parse local configuration file: %v", err)
		}
	}

	return hostOpts
}

func Init() {
	HostOptions = Parse()
}
