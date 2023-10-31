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
	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/log"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/pending_delete"
)

type ComputeOptions struct {
	PortV2 int `help:"Listening port for region V2"`

	DNSServer    string   `help:"Address of DNS server"`
	DNSDomain    string   `help:"Domain suffix for virtual servers"`
	DNSResolvers []string `help:"Upstream DNS resolvers"`

	DefaultCPUOvercommitBound     float32 `default:"8.0" help:"Default cpu overcommit bound for host, default to 8"`
	DefaultMemoryOvercommitBound  float32 `default:"1.0" help:"Default memory overcommit bound for host, default to 1"`
	DefaultStorageOvercommitBound float32 `default:"1.0" help:"Default storage overcommit bound for storage, default to 1"`

	DefaultSecurityGroupId       string `help:"Default security rules" default:"default"`
	DefaultAdminSecurityGroupId  string `help:"Default admin security rules" default:""`
	CleanUselessKvmSecurityGroup bool   `help:"Clean useless kvm security groups when service start"`

	DefaultDiskSizeMB int `default:"10240" help:"Default disk size in MB if not specified, default to 10GiB" json:"default_disk_size"`

	pending_delete.SPendingDeleteOptions

	PrepaidExpireCheck              bool `default:"false" help:"clean expired servers or disks"`
	PrepaidDeleteExpireCheck        bool `default:"false" help:"check prepaid expired before delete"`
	PrepaidExpireCheckSeconds       int  `default:"600" help:"How long to wait to scan expired prepaid VM or disks, default is 10 minutes"`
	ExpiredPrepaidMaxCleanBatchSize int  `default:"50" help:"How many expired prepaid servers can be deleted in a batch"`

	PrepaidAutoRenew      bool `default:"true" help:"auto renew prepaid servers when server's auto_renew attr is true"`
	PrepaidAutoRenewHours int  `default:"3" help:"How long to wait to scan which need renew prepaid VMs, default is 3 hours"`

	LoadbalancerPendingDeleteCheckInterval int `default:"3600" help:"Interval between checks of pending deleted loadbalancer objects, defaults to 1h"`

	ImageCacheStoragePolicy string `default:"least_used" choices:"best_fit|least_used" help:"Policy to choose storage for image cache, best_fit or least_used"`
	MetricsRetentionDays    int32  `default:"30" help:"Retention days for monitoring metrics in influxdb"`

	DefaultBandwidth int `default:"1000" help:"Default bandwidth"`
	DefaultMtu       int `default:"1500" help:"Default network mtu"`
	OvnUnderlayMtu   int `help:"mtu of ovn underlay network" default:"1500"`

	DefaultServerQuota           int `default:"50" help:"Common Server quota per tenant, default 50"`
	DefaultCpuQuota              int `default:"200" help:"Common CPU quota per tenant, default 200"`
	DefaultMemoryQuota           int `default:"204800" help:"Common memory quota per tenant in MB, default 200G"`
	DefaultStorageQuota          int `default:"12288000" help:"Common storage quota per tenant in MB, default 12T"`
	DefaultPortQuota             int `default:"200" help:"Common network port quota per tenant, default 200"`
	DefaultEipQuota              int `default:"10" help:"Common floating IP quota per tenant, default 10"`
	DefaultEportQuota            int `default:"200" help:"Common exit network port quota per tenant, default 200"`
	DefaultBwQuota               int `default:"2000000" help:"Common network port bandwidth in mbps quota per tenant, default 200*10Gbps"`
	DefaultEbwQuota              int `default:"4000" help:"Common exit network port bandwidth quota per tenant, default 4Gbps"`
	DefaultKeypairQuota          int `default:"50" help:"Common keypair quota per tenant, default 50"`
	DefaultGroupQuota            int `default:"50" help:"Common group quota per tenant, default 50"`
	DefaultSecgroupQuota         int `default:"50" help:"Common security group quota per tenant, default 50"`
	DefaultIsolatedDeviceQuota   int `default:"200" help:"Common isolated device quota per tenant, default 200"`
	DefaultSnapshotQuota         int `default:"10" help:"Common snapshot quota per tenant, default 10"`
	DefaultInstanceSnapshotQuota int `default:"10" help:"Common instance snapshot quota per tenant, default 10"`

	DefaultBucketQuota    int `default:"100" help:"Common bucket quota per tenant, default 100"`
	DefaultObjectGBQuota  int `default:"500" help:"Common object size quota per tenant in GB, default 500GB"`
	DefaultObjectCntQuota int `default:"5000" help:"Common object count quota per tenant, default 5000"`

	DefaultLoadbalancerQuota int `default:"10" help:"Common loadbalancer quota per tenant, default 10"`
	DefaultRdsQuota          int `default:"10" help:"Common RDS quota per tenant, default 10"`
	DefaultCacheQuota        int `default:"10" help:"Common ElasticCache quota per tenant, default 10"`
	DefaultMongodbQuota      int `default:"10" help:"Common MongoDB quota per tenant, default 10"`

	DefaultGlobalvpcQuota    int `default:"10" help:"Common global Vpc quota per domain, default 10"`
	DefaultCloudaccountQuota int `default:"20" help:"Common cloud account quota per domain, default 20"`
	DefaultDnsZoneQuota      int `default:"20" help:"Common dns zone quota per domain, default 10"`

	DefaultHostQuota int `default:"500" help:"Common host quota per domain, default 500"`
	DefaultVpcQuota  int `default:"500" help:"Common vpc quota per domain, default 500"`

	DefaultDiskDriver    string `help:"default disk driver" choices:"scsi|virtio|ide" default:"scsi"`
	DefaultDiskCacheMode string `help:"default kvm disk cache mode" choices:"writeback|none|writethrough" default:"none"`

	SystemAdminQuotaCheck         bool `help:"Enable quota check for system admin, default False" default:"false"`
	CloudaccountHealthStatusCheck bool `help:"Enable cloudaccount health status check, default True" default:"true"`

	BaremetalPreparePackageUrl string `help:"Baremetal online register package"`

	// snapshot options
	AutoSnapshotDay               int `default:"1" help:"Days auto snapshot disks, default 1 day"`
	AutoSnapshotHour              int `default:"2" help:"What hour take sanpshot, default 02:00"`
	DefaultMaxSnapshotCount       int `default:"9" help:"Per Disk max snapshot count, default 9"`
	DefaultMaxManualSnapshotCount int `default:"2" help:"Per Disk max manual snapshot count, default 2"`

	//snapshot policy options
	RetentionDaysLimit  int `default:"49" help:"Days of snapshot retention, default 49 days"`
	TimePointsLimit     int `default:"1" help:"time point of every days, default 1 point"`
	RepeatWeekdaysLimit int `default:"7" help:"day point of every weekday, default 7 points"`

	ServerStatusSyncIntervalMinutes int `default:"5" help:"Interval to sync server status, defualt is 5 minutes"`

	ServerSkuSyncIntervalMinutes int `default:"60" help:"Interval to sync public cloud server skus, defualt is 1 hour"`
	SkuBatchSync                 int `default:"5" help:"How many skus can be sync in a batch"`

	// sku sync
	SyncSkusDay  int `default:"1" help:"Days auto sync skus data, default 1 day"`
	SyncSkusHour int `default:"3" help:"What hour start sync skus, default 03:00"`

	ConvertHypervisorDefaultTemplate string `help:"Kvm baremetal convert option"`
	ConvertEsxiDefaultTemplate       string `help:"ESXI baremetal convert option"`
	ConvertKubeletDockerVolumeSize   string `default:"256g" help:"Docker volume size"`

	// cloud image sync
	CloudImagesSyncIntervalHours int `default:"3" help:"Interval to sync public cloud image, defualt is 3 hour"`

	EnablePreAllocateIpAddr bool `help:"Enable private and public cloud private ip pre allocate, default false" default:"false"`

	// 创建虚拟机失败后, 自动使用其他相同配置套餐
	EnableAutoSwitchServerSku bool `help:"If the vm creation fails, use the same configuration server sku"`

	DefaultImageCacheDir string `default:"image_cache"`

	SnapshotCreateDiskProtocol string `help:"Snapshot create disk protocol" choices:"url|fuse" default:"fuse"`

	HostOfflineMaxSeconds        int `help:"Maximal seconds interval that a host considered offline during which it did not ping region, default is 3 minues" default:"180"`
	HostOfflineDetectionInterval int `help:"Interval to check offline hosts, default is half a minute" default:"30"`

	ManagedHostSyncStatusIntervalSeconds int `help:"interval to automatically sync status of managed hosts, default is 5 minutes" default:"300"`

	MinimalIpAddrReusedIntervalSeconds int `help:"Minimal seconds when a release IP address can be reallocate" default:"30"`

	CloudSyncWorkerCount         int `help:"how many current synchronization threads" default:"5"`
	CloudProviderSyncWorkerCount int `help:"how many current providers synchronize their regions, practically no limit" default:"10"`
	CloudAutoSyncIntervalSeconds int `help:"frequency to check auto sync tasks" default:"30"`
	DefaultSyncIntervalSeconds   int `help:"minimal synchronization interval, default 15 minutes" default:"900"`
	MaxCloudAccountErrorCount    int `help:"maximal consecutive error count allow for a cloud account" default:"5"`

	EnableSyncName bool `help:"enable name sync" default:"true"`

	EnableSyncPurge bool `help:"resources that shoud be purged immediately if found removed" default:"true"`

	DisconnectedCloudAccountRetryProbeIntervalHours int `help:"interval to wait to probe status of a disconnected cloud account" default:"2"`

	BaremetalServerReuseHostIp bool `help:"baremetal server reuse host IP address, default true" default:"true"`

	EnableHostHealthCheck bool `help:"enable host health check" default:"false"`
	HostHealthTimeout     int  `help:"second of wait host reconnect" default:"60"`

	GuestTemplateCheckInterval int `help:"interval between two consecutive inspections of Guest Template in hour unit" default:"12"`

	ReconcileGuestBackupIntervalSeconds int `help:"interval reconcile guest bakcups" default:"30"`

	EnableAutoRenameProject bool `help:"when it set true, auto create project will rename when cloud project name changed" default:"false"`

	SyncStorageCapacityUsedIntervalMinutes int  `help:"interval sync storage capacity used" default:"20"`
	LockStorageFromCachedimage             bool `help:"must use storage in where selected cachedimage when creating vm"`

	SyncExtDiskSnapshotIntervalMinutes int  `help:"sync snapshot for external disk" default:"20"`
	AutoReconcileBackupServers         bool `help:"auto reconcile backup servers" default:"false"`
	SetKVMServerAsDaemonOnCreate       bool `help:"set kvm guest as daemon server on create" default:"false"`

	SCapabilityOptions
	SASControllerOptions
	common_options.CommonOptions
	common_options.DBOptions

	DeleteSnapshotExpiredRelease bool `help:"Should the virtual machine be automatically deleted when the virtual machine expires?" default:"false"`
	DeleteEipExpiredRelease      bool `help:"Should the EIP  be automatically deleted when the virtual machine expires?" default:"false"`
	DeleteDisksExpiredRelease    bool `help:"Should the Disks be automatically deleted when the virtual machine expires?" default:"false"`

	DefaultNetworkGatewayAddressEsxi uint32 `help:"Default address for network gateway" default:"1"`

	DefaultVpcExternalAccessMode string `help:"default external access mode for on-premise vpc"`

	NoCheckOsTypeForCachedImage bool `help:"Don't check os type for cached image"`

	ProhibitRefreshingCloudImage bool `help:"Prohibit refreshing cloud image"`

	GlobalMacPrefix string `help:"Global prefix of MAC address, default to 00:22" default:"00:22"`

	DefaultIPAllocationDirection string `help:"default IP allocation direction" default:"stepdown"`

	KeepDeletedSnapshotDays int `help:"The day of cleanup snapshot" default:"30"`
	// 弹性伸缩中的ecs一般会有特殊的系统标签，通过指定这些标签可以忽略这部分ecs的同步, 指定多个key需要以 ',' 分隔
	SkipServerBySysTagKeys  string `help:"skip server,disk sync and create with system tags" default:""`
	SkipServerByUserTagKeys string `help:"skip server,disk sync and create with user tags" default:""`

	EnableMonitorAgent bool `help:"enable public cloud vm monitor agent" default:"false"`

	EnableTlsMigration bool `help:"Enable TLS migration" default:"false"`

	AliyunResourceGroups []string `help:"Only sync indicate resource group resource"`

	KvmMonitorAgentUseMetadataService bool   `help:"Monitor agent report metrics to metadata service on host" default:"true"`
	MonitorEndpointType               string `help:"specify monitor endpoint type" default:"public"`
	ForceUseOriginVnc                 bool   `help:"force openstack use origin vnc console" default:"true"`

	LocalDataDiskMinSizeGB int `help:"Data disk min size when using local storage" default:"10"`
	LocalDataDiskMaxSizeGB int `help:"Data disk max size when using local storage" default:"10240"`

	LocalSysDiskMinSizeGB int `help:"System disk min size when using local storage" default:"30"`
	LocalSysDiskMaxSizeGB int `help:"System disk max size when using local storage" default:"2048"`

	SkuMaxMemSize  int64 `help:"Sku max memory size GB" default:"1024"`
	SkuMaxCpuCount int64 `help:"Sku max cpu count" default:"256"`

	SaveCloudImageToGlance bool `help:"Auto save cloud vm image to glance" default:"true"`

	ResourceExpiredNotifyDays []int `help:"The notify of resource expired" default:"1,3,30"`

	esxi.EsxiOptions
}

type SCapabilityOptions struct {
	MinDataDiskCount   int `help:"Minimal data disk count" default:"0"`
	MaxDataDiskCount   int `help:"Maximal data disk count" default:"12"`
	MinNicCount        int `help:"Minimal nic count" default:"1"`
	MaxNormalNicCount  int `help:"Maximal nic count" default:"8"`
	MaxManagedNicCount int `help:"Maximal managed nic count" default:"1"`
}

type SASControllerOptions struct {
	TimerInterval       int `help:"The interval between the tow checks about timer, unit: s" default:"60"`
	ConcurrentUpper     int `help:"This represents the upper limit of concurrent sacling sctivities" default:"500"`
	CheckScaleInterval  int `help:"The interval between the two checks about scaling, unit: s" default:"60"`
	CheckHealthInterval int `help:"The interval bewteen the two check about instance's health unit: m" default:"1"`
}

var (
	Options ComputeOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*ComputeOptions)
	newOpts := newO.(*ComputeOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	if common_options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}

	if oldOpts.PendingDeleteCheckSeconds != newOpts.PendingDeleteCheckSeconds {
		if !oldOpts.IsSlaveNode {
			changed = true
		}
	}
	if oldOpts.EnableTlsMigration != newOpts.EnableTlsMigration {
		log.Debugf("enable_tls_migration changed from %v to %v", oldOpts.EnableTlsMigration, newOpts.EnableTlsMigration)
	}

	return changed
}
