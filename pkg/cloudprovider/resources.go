package cloudprovider

import (
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
)

type ICloudResource interface {
	GetId() string
	GetName() string
	GetGlobalId() string

	GetStatus() string

	Refresh() error

	IsEmulated() bool
}

type ICloudRegion interface {
	ICloudResource

	GetLatitude() float32
	GetLongitude() float32

	GetIZones() ([]ICloudZone, error)
	GetIVpcs() ([]ICloudVpc, error)

	GetIZoneById(id string) (ICloudZone, error)
	GetIVpcById(id string) (ICloudVpc, error)
	GetIHostById(id string) (ICloudHost, error)
	GetIStorageById(id string) (ICloudStorage, error)
	GetIStoragecacheById(id string) (ICloudStoragecache, error)

	CreateIVpc(name string, desc string, cidr string) (ICloudVpc, error)

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

	GetIStoragecache() ICloudStoragecache
}

type ICloudStoragecache interface {
	ICloudResource

	GetIImages() ([]ICloudImage, error)

	GetManagerId() string

	UploadImage(userCred mcclient.TokenCredential, imageId string, extId string, isForce bool) (string, error)
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

	GetManagerId() string

	CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, vswitchId string, ipAddr string, desc string,
		passwd string, storageType string, diskSizes []int, publicKey string) (ICloudVM, error)
}

type ICloudVM interface {
	ICloudResource

	GetCreateTime() time.Time
	GetIHost() ICloudHost

	GetIDisks() ([]ICloudDisk, error)
	GetINics() ([]ICloudNic, error)

	GetEIP() ICloudEIP

	// GetStatus() string
	GetRemoteStatus() string

	GetVcpuCount() int8
	GetVmemSizeMB() int //MB
	GetBootOrder() string
	GetVga() string
	GetVdi() string
	GetOSType() string
	GetOSName() string
	GetBios() string
	GetMachine() string

	GetHypervisor() string

	// GetSecurityGroup() ICloudSecurityGroup

	StartVM() error
	StopVM(isForce bool) error
	DeleteVM() error

	GetVNCInfo() (jsonutils.JSONObject, error)
}

type ICloudNic interface {
	GetIP() string
	GetMAC() string
	GetDriver() string
	GetINetwork() ICloudNetwork
}

type ICloudEIP interface {
	GetIP() string
	GetAllocationId() string
	GetChargeType() string
}

type ICloudSecurityGroup interface {
	ICloudResource
}

type ICloudDisk interface {
	ICloudResource

	GetIStorge() ICloudStorage

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
}

type ICloudVpc interface {
	ICloudResource

	GetRegion() ICloudRegion
	GetIsDefault() bool
	GetCidrBlock() string
	// GetStatus() string
	GetIWires() ([]ICloudWire, error)

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
}
