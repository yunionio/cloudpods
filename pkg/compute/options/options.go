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
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/pending_delete"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
)

type ComputeOptions struct {
	PortV2 int `help:"Listening port for region V2"`

	DNSServer    string   `help:"Address of DNS server"`
	DNSDomain    string   `help:"Domain suffix for virtual servers"`
	DNSResolvers []string `help:"Upstream DNS resolvers"`

	IgnoreNonrunningGuests        bool    `default:"true" help:"Count memory for running guests only when do scheduling. Ignore memory allocation for non-running guests"`
	DefaultCPUOvercommitBound     float32 `default:"8.0" help:"Default cpu overcommit bound for host, default to 8"`
	DefaultMemoryOvercommitBound  float32 `default:"1.0" help:"Default memory overcommit bound for host, default to 1"`
	DefaultStorageOvercommitBound float32 `default:"1.0" help:"Default storage overcommit bound for storage, default to 1"`

	DefaultSecurityRules      string `help:"Default security rules" default:"allow any"`
	DefaultAdminSecurityRules string `help:"Default admin security rules" default:""`

	DefaultDiskSizeMB int `default:"10240" help:"Default disk size in MB if not specified, default to 10GiB" json:"default_disk_size"`

	pending_delete.SPendingDeleteOptions

	PrepaidExpireCheck              bool `default:"false" help:"clean expired servers or disks"`
	PrepaidDeleteExpireCheck        bool `default:"true" help:"check prepaid expired before delete"`
	PrepaidExpireCheckSeconds       int  `default:"600" help:"How long to wait to scan expired prepaid VM or disks, default is 10 minutes"`
	ExpiredPrepaidMaxCleanBatchSize int  `default:"50" help:"How many expired prepaid servers can be deleted in a batch"`

	PrepaidAutoRenew      bool `default:"true" help:"auto renew prepaid servers when server's auto_renew attr is true"`
	PrepaidAutoRenewHours int  `default:"3" help:"How long to wait to scan which need renew prepaid VMs, default is 3 hours"`

	LoadbalancerPendingDeleteCheckInterval int `default:"3600" help:"Interval between checks of pending deleted loadbalancer objects, defaults to 1h"`

	ImageCacheStoragePolicy string `default:"least_used" choices:"best_fit|least_used" help:"Policy to choose storage for image cache, best_fit or least_used"`
	MetricsRetentionDays    int32  `default:"30" help:"Retention days for monitoring metrics in influxdb"`

	DefaultBandwidth int `default:"1000" help:"Default bandwidth"`
	DefaultMtu       int `default:"1500" help:"Default network mtu"`

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

	DefaultGlobalvpcQuota    int `default:"10" help:"Common global Vpc quota per domain, default 10"`
	DefaultCloudaccountQuota int `default:"20" help:"Common cloud account quota per domain, default 20"`
	DefaultDnsZoneQuota      int `default:"20" help:"Common dns zone quota per domain, default 10"`

	DefaultHostQuota int `default:"500" help:"Common host quota per domain, default 500"`
	DefaultVpcQuota  int `default:"500" help:"Common vpc quota per domain, default 500"`

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

	// sku sync
	SyncSkusDay  int `default:"1" help:"Days auto sync skus data, default 1 day"`
	SyncSkusHour int `default:"3" help:"What hour start sync skus, default 03:00"`

	ConvertHypervisorDefaultTemplate string `help:"Kvm baremetal convert option"`
	ConvertEsxiDefaultTemplate       string `help:"ESXI baremetal convert option"`
	ConvertKubeletDockerVolumeSize   string `default:"256g" help:"Docker volume size"`

	// cloud image sync
	SyncCloudImagesDay  int `default:"1" help:"Days auto sync public cloud images data, default 1 day"`
	SyncCloudImagesHour int `default:"3" help:"What hour start sync public cloud images, default 03:00"`

	EnablePreAllocateIpAddr bool `help:"Enable private and public cloud private ip pre allocate, default true" default:"true"`

	DefaultImageCacheDir string `default:"image_cache"`

	SnapshotCreateDiskProtocol string `help:"Snapshot create disk protocol" choices:"url|fuse" default:"fuse"`

	HostOfflineMaxSeconds        int `help:"Maximal seconds interval that a host considered offline during which it did not ping region, default is 3 minues" default:"180"`
	HostOfflineDetectionInterval int `help:"Interval to check offline hosts, defualt is half a minute" default:"30"`

	MinimalIpAddrReusedIntervalSeconds int `help:"Minimal seconds when a release IP address can be reallocate" default:"30"`

	CloudSyncWorkerCount         int `help:"how many current synchronization threads" default:"5"`
	CloudProviderSyncWorkerCount int `help:"how many current providers synchronize their regions, practically no limit" default:"10"`
	CloudAutoSyncIntervalSeconds int `help:"frequency to check auto sync tasks" default:"30"`
	DefaultSyncIntervalSeconds   int `help:"minimal synchronization interval, default 15 minutes" default:"900"`
	MinimalSyncIntervalSeconds   int `help:"minimal synchronization interval, default 30 minutes" default:"1800"`
	MaxCloudAccountErrorCount    int `help:"maximal consecutive error count allow for a cloud account" default:"5"`

	NameSyncResources []string `help:"resources that need synchronization of name"`

	SyncPurgeRemovedResources []string `help:"resources that shoud be purged immediately if found removed" default:"server"`

	DisconnectedCloudAccountRetryProbeIntervalHours int `help:"interval to wait to probe status of a disconnected cloud account" default:"2"`

	BaremetalServerReuseHostIp bool `help:"baremetal server reuse host IP address, default true" default:"true"`

	EnableHostHealthCheck bool `help:"enable host health check" default:"true"`
	HostHealthTimeout     int  `help:"second of wait host reconnect" default:"60"`

	GuestTemplateCheckInterval int `help:"interval between two consecutive inspections of Guest Template in hour unit" default:"12"`

	ScheduledTaskQueueSize int `help:"the maximum number of scheduled tasks that are being executed simultaneously" default:"100"`

	ReconcileGuestBackupIntervalSeconds int `help:"interval reconcile guest bakcups" default:"30"`

	EnableAutoRenameProject bool `help:"when it set true, auto create project will rename when cloud project name changed" default:"false"`

	SyncStorageCapacityUsedIntervalMinutes int  `help:"interval sync storage capacity used" default:"20"`
	LockStorageFromCachedimage             bool `help:"must use storage in where selected cachedimage when creating vm"`

	SyncExtDiskSnapshotIntervalMinutes int  `help:"sync snapshot for external disk" default:"20"`
	AutoReconcileBackupServers         bool `help:"auto reconcile backup servers" default:"false"`

	SCapabilityOptions
	SASControllerOptions
	common_options.CommonOptions
	common_options.DBOptions

	EnableAutoMergeSecurityGroup bool `help:"Enable auto merge secgroup when sync security group from cloud, default False" default:"false"`
	DeleteSnapshotExpiredRelease bool `help:"Should the virtual machine be automatically deleted when the virtual machine expires?" default:"false"`

	DefaultNetworkGatewayAddressEsxi uint32 `help:"Default address for network gateway" default:"1"`

	DefaultVpcExternalAccessMode string `help:"default external access mode for on-premise vpc"`

	NoCheckOsTypeForCachedImage bool `help:"Don't check os type for cached image"`

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

	return changed
}
