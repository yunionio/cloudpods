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
}

type ICloudRegion interface {
	ICloudResource
	GetLatitude() float32
	GetLongitude() float32
	GetStatus() string

	GetIZones() ([]ICloudZone, error)
	GetIVpcs() ([]ICloudVpc, error)

	GetIZoneById(id string) (ICloudZone, error)
	GetIVpcById(id string) (ICloudVpc, error)
}

type ICloudZone interface {
	ICloudResource

	GetIRegion() ICloudRegion
	GetStatus() string

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)
}

type ICloudImage interface {
	ICloudResource

	GetIStoragecache() ICloudStoragecache
	GetStatus() string
}

type ICloudStoragecache interface {
	ICloudResource

	GetIImages() ([]ICloudImage, error)

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
	GetStatus() string
	GetEnabled() bool
}

type ICloudHost interface {
	ICloudResource

	GetIVMs() ([]ICloudVM, error)
	GetIVMById(id string) (ICloudVM, error)

	GetIWires() ([]ICloudWire, error)
	GetIStorages() ([]ICloudStorage, error)

	GetStatus() string     // os status
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

	CreateVM(name string, imgId string, cpu int, memMB int, vswitchId string, ipAddr string, desc string,
		passwd string, storageType string, diskSizes []int) (ICloudVM, error)
}

type ICloudVM interface {
	ICloudResource

	GetCreateTime() time.Time
	GetIHost() ICloudHost

	GetIDisks() ([]ICloudDisk, error)
	GetINics() ([]ICloudNic, error)

	GetEIP() ICloudEIP

	GetStatus() string
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

	GetStatus() string
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
	GetStatus() string
	GetIWires() ([]ICloudWire, error)
}

type ICloudWire interface {
	ICloudResource
	GetIVpc() ICloudVpc
	GetIZone() ICloudZone
	GetINetworks() ([]ICloudNetwork, error)
	GetBandwidth() int
}

type ICloudNetwork interface {
	ICloudResource

	GetIWire() ICloudWire
	GetStatus() string
	GetIpStart() string
	GetIpEnd() string
	GetIpMask() int8
	GetGateway() string
	GetServerType() string
	GetIsPublic() bool
}
