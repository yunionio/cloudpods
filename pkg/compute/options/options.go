package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
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

	DefaultDiskSize int `default:"30720" help:"Default disk size in MB if not specified, default to 30GiB"`

	EnablePendingDelete            bool `default:"true" help:"Turn on/off pending delete VM and disk, default is on"`
	PendingDeleteCheckSeconds      int  `default:"3600" help:"How long to wait to scan pending delete VM or disks, default is 1 hour"`
	PendingDeleteExpireSeconds     int  `default:"259200" help:"How long a pending delete VM/disks cleaned automatically, default 3 days"`
	PendingDeleteMaxCleanBatchSize int  `default:"50" help:"How many pending delete servers can be clean in a batch"`

	PrepaidExpireCheckSeconds int `default:"600" help:"How long to wait to scan expired prepaid VM or disks, default is 10 minutes"`
	ExpiredPrepaidMaxCleanBatchSize int  `default:"50" help:"How many expired prepaid servers can be deleted in a batch"`

	LoadbalancerPendingDeleteCheckInterval int `default:"3600" help:"Interval between checks of pending deleted loadbalancer objects, defaults to 1h"`

	ImageCacheStoragePolicy string `default:"least_used" choices:"best_fit|least_used" help:"Policy to choose storage for image cache, best_fit or least_used"`
	MetricsRetentionDays    int32  `default:"30" help:"Retention days for monitoring metrics in influxdb"`

	DefaultBandwidth int `default:"1000" help:"Default bandwidth"`

	DefaultCpuQuota            int `help:"Common CPU quota per tenant, default 200" default:"200"`
	DefaultMemoryQuota         int `default:"204800" help:"Common memory quota per tenant in MB, default 200G"`
	DefaultStorageQuota        int `default:"12288000" help:"Common storage quota per tenant in MB, default 12T"`
	DefaultPortQuota           int `default:"200" help:"Common network port quota per tenant, default 200"`
	DefaultEipQuota            int `default:"10" help:"Common floating IP quota per tenant, default 10"`
	DefaultEportQuota          int `default:"200" help:"Common exit network port quota per tenant, default 200"`
	DefaultBwQuota             int `default:"2000000" help:"Common network port bandwidth in mbps quota per tenant, default 200*10Gbps"`
	DefaultEbwQuota            int `default:"4000" help:"Common exit network port bandwidth quota per tenant, default 4Gbps"`
	DefaultKeypairQuota        int `default:"50" help:"Common keypair quota per tenant, default 50"`
	DefaultImageQuota          int `default:"5" help:"Common image quota per tenant, default 5"`
	DefaultGroupQuota          int `default:"50" help:"Common group quota per tenant, default 50"`
	DefaultSecgroupQuota       int `default:"50" help:"Common security group quota per tenant, default 50"`
	DefaultIsolatedDeviceQuota int `default:"200" help:"Common isolated device quota per tenant, default 200"`
	DefaultSnapshotQuota       int `default:"10" help:"Common snapshot quota per tenant, default 10"`

	SystemAdminQuotaCheck bool `help:"Enable quota check for system admin, default False" default:"false"`

	BaremetalPreparePackageUrl string `help:"Baremetal online register package"`

	// snapshot options
	AutoSnapshotDay               int `default:"1" help:"Days auto snapshot disks, default 1 day"`
	AutoSnapshotHour              int `default:"2" help:"What hour take sanpshot, default 02:00"`
	DefaultMaxSnapshotCount       int `default:"9" help:"Per Disk max snapshot count, default 9"`
	DefaultMaxManualSnapshotCount int `default:"2" help:"Per Disk max manual snapshot count, default 2"`

	// sku sync
	SyncSkusDay  int `default:"1" help:"Days auto sync skus data, default 1 day"`
	SyncSkusHour int `default:"3" help:"What hour start sync skus, default 03:00"`

	// aws instance type file
	DefaultAwsInstanceTypeFile string `default:"/etc/yunion/aws_instance_types.json" help:"aws instance type json file"`

	ConvertHypervisorDefaultTemplate string `default:"Default template" help:"Kvm baremetal convert option"`
	ConvertEsxiDefaultTemplate       string `default:"Default template" help:"ESXI baremetal convert option"`
	ConvertKubeletDockerVolumeSize   string `default:"256g" help:"Docker volume size"`

	NfsDefaultImageCacheDir string `default:"image_cache"`

	cloudcommon.DBOptions
}

var (
	Options ComputeOptions
)
