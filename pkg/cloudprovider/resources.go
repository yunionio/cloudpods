package cloudprovider

import (
	"time"

	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/secrules"
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

	GetLatitude() float32
	GetLongitude() float32

	GetIZones() ([]ICloudZone, error)
	GetIVpcs() ([]ICloudVpc, error)
	GetIEips() ([]ICloudEIP, error)
	GetISnapshots() ([]ICloudSnapshot, error)

	GetISnapshotById(snapshotId string) (ICloudSnapshot, error)
	GetIZoneById(id string) (ICloudZone, error)
	GetIVpcById(id string) (ICloudVpc, error)
	GetIHostById(id string) (ICloudHost, error)
	GetIStorageById(id string) (ICloudStorage, error)
	GetIStoragecacheById(id string) (ICloudStoragecache, error)

	CreateIVpc(name string, desc string, cidr string) (ICloudVpc, error)

	CreateEIP(name string, bwMbps int, chargeType string) (ICloudEIP, error)

	GetIEipById(id string) (ICloudEIP, error)

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

	Delete() error
	GetIStoragecache() ICloudStoragecache
}

type ICloudStoragecache interface {
	ICloudResource

	GetIImages() ([]ICloudImage, error)

	GetManagerId() string

	CreateIImage(snapshotId, imageName, osType, imageDesc string) (ICloudImage, error)

	DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error)

	UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, extId string, isForce bool) (string, error)
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
	GetIDisk(idStr string) (ICloudDisk, error)
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
		passwd string, storageType string, diskSizes []int, publicKey string, extSecGrpId string, userData string) (ICloudVM, error)

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

	SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) error
	GetHypervisor() string

	// GetSecurityGroup() ICloudSecurityGroup

	StartVM(ctx context.Context) error
	StopVM(ctx context.Context, isForce bool) error
	DeleteVM(ctx context.Context) error

	UpdateVM(ctx context.Context, name string) error

	UpdateUserData(userData string) error

	RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error)

	DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error

	ChangeConfig(ctx context.Context, instanceId string, ncpu int, vmem int) error
	GetVNCInfo() (jsonutils.JSONObject, error)
	AttachDisk(ctx context.Context, diskId string) error
	DetachDisk(ctx context.Context, diskId string) error
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
	Delete(ctx context.Context) error

	CreateISnapshot(ctx context.Context, name string, desc string) (ICloudSnapshot, error)
	GetISnapshot(idStr string) (ICloudSnapshot, error)
	GetISnapshots() ([]ICloudSnapshot, error)

	Resize(ctx context.Context, newSize int64) error
	Reset(ctx context.Context, snapshotId string) error
}

type ICloudSnapshot interface {
	ICloudResource
	GetManagerId() string
	GetSize() int32
	GetDiskId() string
	GetDiskType() string
	Delete() error
	GetRegionId() string
}

type ICloudVpc interface {
	ICloudResource

	GetRegion() ICloudRegion
	GetIsDefault() bool
	GetCidrBlock() string
	// GetStatus() string
	GetIWires() ([]ICloudWire, error)
	GetISecurityGroups() ([]ICloudSecurityGroup, error)

	GetManagerId() string

	Delete() error

	GetIWireById(wireId string) (ICloudWire, error)

	SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) (string, error)
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
	IsLinkUp() bool
	GetIpAddr() string
	GetMtu() int16
	GetNicType() string
}
