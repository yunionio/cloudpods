package cloudprovider

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type ICloudResource interface {
	GetId() string
	GetName() string
	GetGlobalId() string

	GetStatus() string

	Refresh() error

	IsEmulated() bool
	GetMetadata() *jsonutils.JSONDict
}

type IBillingResource interface {
	GetBillingType() string
	GetExpiredAt() time.Time
}

type ICloudRegion interface {
	ICloudResource

	// GetLatitude() float32
	// GetLongitude() float32
	GetGeographicInfo() SGeographicInfo

	GetIZones() ([]ICloudZone, error)
	GetIVpcs() ([]ICloudVpc, error)
	GetIEips() ([]ICloudEIP, error)
	GetIVpcById(id string) (ICloudVpc, error)
	GetIZoneById(id string) (ICloudZone, error)
	GetIEipById(id string) (ICloudEIP, error)

	DeleteSecurityGroup(vpcId, secgroupId string) error
	SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error)

	CreateIVpc(name string, desc string, cidr string) (ICloudVpc, error)
	CreateEIP(name string, bwMbps int, chargeType string) (ICloudEIP, error)

	GetISnapshots() ([]ICloudSnapshot, error)
	GetISnapshotById(snapshotId string) (ICloudSnapshot, error)

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)

	GetILoadBalancers() ([]ICloudLoadbalancer, error)
	GetILoadbalancerAcls() ([]ICloudLoadbalancerAcl, error)
	GetILoadbalancerCertificates() ([]ICloudLoadbalancerCertificate, error)

	GetProvider() string
}

type ICloudZone interface {
	ICloudResource

	GetIRegion() ICloudRegion

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)
}

type ICloudImage interface {
	ICloudResource

	Delete(ctx context.Context) error
	GetIStoragecache() ICloudStoragecache
}

type ICloudStoragecache interface {
	ICloudResource

	GetIImages() ([]ICloudImage, error)
	GetIImageById(extId string) (ICloudImage, error)

	GetPath() string

	GetManagerId() string

	CreateIImage(snapshotId, imageName, osType, imageDesc string) (ICloudImage, error)

	DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error)

	UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error)
}

type ICloudStorage interface {
	ICloudResource

	GetIStoragecache() ICloudStoragecache

	GetIZone() ICloudZone
	GetIDisks() ([]ICloudDisk, error)

	GetStorageType() string
	GetMediumType() string
	GetCapacityMB() int // MB
	GetStorageConf() jsonutils.JSONObject
	GetEnabled() bool

	GetManagerId() string

	CreateIDisk(name string, sizeGb int, desc string) (ICloudDisk, error)

	GetIDiskById(idStr string) (ICloudDisk, error)

	GetMountPoint() string
}

type ICloudHost interface {
	ICloudResource

	GetIVMs() ([]ICloudVM, error)
	GetIVMById(id string) (ICloudVM, error)

	GetIWires() ([]ICloudWire, error)
	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)

	// GetStatus() string     // os status
	GetEnabled() bool      // is enabled
	GetHostStatus() string // service status
	GetAccessIp() string   //
	GetAccessMac() string  //
	GetSysInfo() jsonutils.JSONObject
	GetSN() string
	GetCpuCount() int8
	GetNodeCount() int8
	GetCpuDesc() string
	GetCpuMhz() int
	GetMemSizeMB() int
	GetStorageSizeMB() int
	GetStorageType() string
	GetHostType() string

	GetIsMaintenance() bool
	GetVersion() string

	GetManagerId() string

	CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, vswitchId string, ipAddr string, desc string,
		passwd string, storageType string, diskSizes []int, publicKey string, extSecGrpId string, userData string, billingCycle *billing.SBillingCycle) (ICloudVM, error)
	// 使用instanceType创建实例。
	CreateVM2(name string, imgId string, sysDiskSize int, instanceType string, vswitchId string, ipAddr string, desc string,
		passwd string, storageType string, diskSizes []int, publicKey string, extSecGrpId string, userData string, billingCycle *billing.SBillingCycle) (ICloudVM, error)

	GetIHostNics() ([]ICloudHostNetInterface, error)
}

type ICloudVM interface {
	ICloudResource
	IBillingResource

	GetCreateTime() time.Time
	GetIHost() ICloudHost

	GetIDisks() ([]ICloudDisk, error)
	GetINics() ([]ICloudNic, error)

	GetIEIP() (ICloudEIP, error)

	// GetStatus() string
	// GetRemoteStatus() string

	GetVcpuCount() int8
	GetVmemSizeMB() int //MB
	GetBootOrder() string
	GetVga() string
	GetVdi() string
	GetOSType() string
	GetOSName() string
	GetBios() string
	GetMachine() string
	GetInstanceType() string

	AssignSecurityGroup(secgroupId string) error
	AssignSecurityGroups(secgroupIds []string) error

	GetHypervisor() string

	// GetSecurityGroup() ICloudSecurityGroup

	StartVM(ctx context.Context) error
	StopVM(ctx context.Context, isForce bool) error
	DeleteVM(ctx context.Context) error

	UpdateVM(ctx context.Context, name string) error

	UpdateUserData(userData string) error

	RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error)

	DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error

	ChangeConfig(ctx context.Context, ncpu int, vmem int) error
	ChangeConfig2(ctx context.Context, instanceType string) error // instanceType support

	GetVNCInfo() (jsonutils.JSONObject, error)
	AttachDisk(ctx context.Context, diskId string) error
	DetachDisk(ctx context.Context, diskId string) error

	CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error

	Renew(bc billing.SBillingCycle) error
}

type ICloudNic interface {
	GetIP() string
	GetMAC() string
	GetDriver() string
	GetINetwork() ICloudNetwork
}

type ICloudEIP interface {
	ICloudResource

	GetIpAddr() string
	GetMode() string
	GetAssociationType() string
	GetAssociationExternalId() string

	GetBandwidth() int

	GetInternetChargeType() string

	GetManagerId() string

	Delete() error

	Associate(instanceId string) error
	Dissociate() error

	ChangeBandwidth(bw int) error
}

type ICloudSecurityGroup interface {
	ICloudResource
	GetDescription() string
	GetRules() ([]secrules.SecurityRule, error)
	GetVpcId() string
}

type ICloudRouteTable interface {
	ICloudResource
	GetManagerId() string

	GetDescription() string
	GetRegionId() string
	GetVpcId() string
	GetType() string
	GetIRoutes() ([]ICloudRoute, error)
}

type ICloudRoute interface {
	GetType() string
	GetCidr() string
	GetNextHopType() string
	GetNextHop() string
}

type ICloudDisk interface {
	ICloudResource
	IBillingResource

	GetIStorage() (ICloudStorage, error)

	// GetStatus() string
	GetDiskFormat() string
	GetDiskSizeMB() int // MB
	GetIsAutoDelete() bool
	GetTemplateId() string
	GetDiskType() string
	GetFsFormat() string
	GetIsNonPersistent() bool

	GetDriver() string
	GetCacheMode() string
	GetMountpoint() string

	GetAccessPath() string

	Delete(ctx context.Context) error

	CreateISnapshot(ctx context.Context, name string, desc string) (ICloudSnapshot, error)
	GetISnapshot(idStr string) (ICloudSnapshot, error)
	GetISnapshots() ([]ICloudSnapshot, error)

	Resize(ctx context.Context, newSizeMB int64) error
	Reset(ctx context.Context, snapshotId string) (string, error)

	Rebuild(ctx context.Context) error
}

type ICloudSnapshot interface {
	ICloudResource
	GetSize() int32
	GetDiskId() string
	GetDiskType() string
	Delete() error
}

type ICloudVpc interface {
	ICloudResource

	GetRegion() ICloudRegion
	GetIsDefault() bool
	GetCidrBlock() string
	// GetStatus() string
	GetIWires() ([]ICloudWire, error)
	GetISecurityGroups() ([]ICloudSecurityGroup, error)
	GetIRouteTables() ([]ICloudRouteTable, error)

	GetManagerId() string

	Delete() error

	GetIWireById(wireId string) (ICloudWire, error)
}

type ICloudWire interface {
	ICloudResource
	GetIVpc() ICloudVpc
	GetIZone() ICloudZone
	GetINetworks() ([]ICloudNetwork, error)
	GetBandwidth() int

	GetINetworkById(netid string) (ICloudNetwork, error)

	CreateINetwork(name string, cidr string, desc string) (ICloudNetwork, error)
}

type ICloudNetwork interface {
	ICloudResource

	GetIWire() ICloudWire
	// GetStatus() string
	GetIpStart() string
	GetIpEnd() string
	GetIpMask() int8
	GetGateway() string
	GetServerType() string
	GetIsPublic() bool

	Delete() error

	GetAllocTimeoutSeconds() int
}

type ICloudHostNetInterface interface {
	GetDevice() string
	GetDriver() string
	GetMac() string
	GetIndex() int8
	IsLinkUp() tristate.TriState
	GetIpAddr() string
	GetMtu() int16
	GetNicType() string
}

type ICloudLoadbalancer interface {
	ICloudResource

	GetAddress() string
	GetAddressType() string
	GetNetworkType() string
	GetNetworkId() string
	GetVpcId() string
	GetZoneId() string

	GetILoadbalancerListeners() ([]ICloudLoadbalancerListener, error)
	GetILoadbalancerBackendGroups() ([]ICloudLoadbalancerBackendGroup, error)
}

type ICloudLoadbalancerListener interface {
	ICloudResource

	GetListenerType() string
	GetListenerPort() int
	GetScheduler() string
	GetAclStatus() string
	GetAclType() string
	GetAclId() string

	GetHealthCheck() string
	GetHealthCheckType() string
	GetHealthCheckTimeout() int
	GetHealthCheckInterval() int
	GetHealthCheckRise() int
	GetHealthCheckFail() int

	GetHealthCheckReq() string
	GetHealthCheckExp() string

	GetBackendGroupId() string

	// HTTP && HTTPS
	GetHealthCheckDomain() string
	GetHealthCheckURI() string
	GetHealthCheckCode() string
	GetILoadbalancerListenerRules() ([]ICloudLoadbalancerListenerRule, error)
	GetStickySession() string
	GetStickySessionType() string
	GetStickySessionCookie() string
	GetStickySessionCookieTimeout() int
	XForwardedForEnabled() bool
	GzipEnabled() bool

	// HTTPS
	GetCertificateId() string
	GetTLSCipherPolicy() string
	HTTP2Enabled() bool
}

type ICloudLoadbalancerListenerRule interface {
	ICloudResource

	GetDomain() string
	GetPath() string
	GetBackendGroupId() string
}

type ICloudLoadbalancerBackendGroup interface {
	ICloudResource

	IsDefault() bool
	GetType() string
	GetILoadbalancerBackends() ([]ICloudLoadbalancerBackend, error)
}

type ICloudLoadbalancerBackend interface {
	ICloudResource

	GetWeight() int
	GetPort() int
	GetBackendType() string
	GetBackendRole() string
	GetBackendId() string
}

type ICloudLoadbalancerCertificate interface {
	ICloudResource

	GetCommonName() string
	GetSubjectAlternativeNames() string
	GetFingerprint() string // return value format: <algo>:<fingerprint>，比如sha1:7454a14fdb8ae1ea8b2f72e458a24a76bd23ec19
	GetExpireTime() time.Time
}

type ICloudLoadbalancerAcl interface {
	ICloudResource

	// return array data like this:
	// [
	// 	{
	// 		"cidr":"10.10.12.0/24",
	// 		"comment":"test data"
	// 	},
	// 	{
	// 		"cidr":"192.168.10.12",
	// 		"comment":"test data2"
	// 	}
	// ]
	GetAclEntries() *jsonutils.JSONArray
}
