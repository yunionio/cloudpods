package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SMountInstances struct {
	MountInstance []string
}

type STags struct {
	Tag []string
}

type SDisk struct {
	storage *SStorage

	RegionId string
	ZoneId   string // AvailabilityZone
	DiskId   string // VolumeId

	DiskName         string // Tag Name
	Size             int    // Size
	Category         string // VolumeType?
	Type             string // VolumeType
	Status           string // State
	Device           string // Device
	InstanceId       string // InstanceId
	Encrypted        bool   // Encrypted
	SourceSnapshotId string // SnapshotId
	Iops             int    // Iops
	Tags             STags

	CreationTime time.Time // CreateTime
	AttachedTime time.Time // AttachTime
	DetachedTime time.Time

	DeleteWithInstance            bool // DeleteOnTermination
	EnableAutoSnapshot            bool
	EnableAutomatedSnapshotPolicy bool

	/*下面这些字段也许不需要*/
	AutoSnapshotPolicyId string
	DeleteAutoSnapshot   bool
	Description          string
	DiskChargeType       InstanceChargeType
	ExpiredTime          time.Time
	ImageId              string
	MountInstances       SMountInstances
	OperationLocks       SOperationLocks
	Portable             bool
	ProductCode          string
	ResourceGroupId      string
}

func (self *SDisk) GetId() string {
	panic("implement me")
}

func (self *SDisk) GetName() string {
	panic("implement me")
}

func (self *SDisk) GetGlobalId() string {
	panic("implement me")
}

func (self *SDisk) GetStatus() string {
	panic("implement me")
}

func (self *SDisk) Refresh() error {
	panic("implement me")
}

func (self *SDisk) IsEmulated() bool {
	panic("implement me")
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SDisk) GetBillingType() string {
	panic("implement me")
}

func (self *SDisk) GetExpiredAt() time.Time {
	panic("implement me")
}

func (self *SDisk) GetIStorge() cloudprovider.ICloudStorage {
	panic("implement me")
}

func (self *SDisk) GetDiskFormat() string {
	panic("implement me")
}

func (self *SDisk) GetDiskSizeMB() int {
	panic("implement me")
}

func (self *SDisk) GetIsAutoDelete() bool {
	panic("implement me")
}

func (self *SDisk) GetTemplateId() string {
	panic("implement me")
}

func (self *SDisk) GetDiskType() string {
	panic("implement me")
}

func (self *SDisk) GetFsFormat() string {
	panic("implement me")
}

func (self *SDisk) GetIsNonPersistent() bool {
	panic("implement me")
}

func (self *SDisk) GetDriver() string {
	panic("implement me")
}

func (self *SDisk) GetCacheMode() string {
	panic("implement me")
}

func (self *SDisk) GetMountpoint() string {
	panic("implement me")
}

func (self *SDisk) Delete() error {
	panic("implement me")
}

func (self *SDisk) CreateISnapshot(name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	panic("implement me")
}

func (self *SDisk) GetISnapshot(idStr string) (cloudprovider.ICloudSnapshot, error) {
	panic("implement me")
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	panic("implement me")
}

func (self *SDisk) Resize(newSize int64) error {
	panic("implement me")
}

func (self *SDisk) Reset(snapshotId string) error {
	panic("implement me")
}
