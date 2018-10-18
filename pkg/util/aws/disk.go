package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/onecloud/pkg/compute/models"
	"github.com/coredns/coredns/plugin/pkg/log"
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
	Category         string // VolumeType
	Type             string // system | data
	Status           string // State
	AttachmentStatus           string // attachment.status
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
	Portable             bool
	ProductCode          string
	ResourceGroupId      string
}

func (self *SDisk) GetId() string {
	return self.DiskId
}

func (self *SDisk) GetName() string {
	if len(self.DiskName) > 0 {
		return self.DiskName
	}
	return self.DiskId
}

func (self *SDisk) GetGlobalId() string {
	return self.DiskId
}

func (self *SDisk) GetStatus() string {
	// creating | available | in-use | deleting | deleted | error
	switch self.Status {
	case "creating":
		return models.DISK_ALLOCATING
	case "deleting":
		return models.DISK_DEALLOC
	case "error":
		return models.DISK_ALLOC_FAILED
	default:
		return models.DISK_READY
	}
}

func (self *SDisk) Refresh() error {
	new, err := self.storage.zone.region.GetDisk(self.DiskId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SDisk) IsEmulated() bool {
	return false
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
	return self.storage
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.Size * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.DeleteWithInstance
}

func (self *SDisk) GetTemplateId() string {
	return self.ImageId
}

func (self *SDisk) GetDiskType() string {
	return self.Type
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return "scsi"
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) Delete() error {
	if _, err := self.storage.zone.region.GetDisk(self.DiskId); err == cloudprovider.ErrNotFound {
		log.Errorf("Failed to find disk %s when delete", self.DiskId)
		return nil
	}
	return self.storage.zone.region.DeleteDisk(self.DiskId)
}

func (self *SDisk) CreateISnapshot(name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	if snapshotId, err := self.storage.zone.region.CreateSnapshot(self.DiskId, name, desc); err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	} else if snapshot, err := self.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = self.storage.zone.region
		if err := cloudprovider.WaitStatus(snapshot, models.SNAPSHOT_READY, 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return snapshot, nil
	}
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if snapshot, err := self.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = self.storage.zone.region
		return snapshot, nil
	}
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	for {
		if parts, total, err := self.storage.zone.region.GetSnapshots("", self.DiskId, "", []string{}, 0, 20); err != nil {
			log.Errorf("GetDisks fail %s", err)
			return nil, err
		} else {
			snapshots = append(snapshots, parts...)
			if len(snapshots) >= total {
				break
			}
		}
	}
	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self.storage.zone.region
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (self *SDisk) Resize(newSize int64) error {
	return self.storage.zone.region.resizeDisk(self.DiskId, newSize)
}

func (self *SDisk) Reset(snapshotId string) error {
	panic("implement me")
}

func (self *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	if snapshots, total, err := self.storage.zone.region.GetSnapshots("", "", "", []string{snapshotId}, 0, 1); err != nil {
		return nil, err
	} else if total != 1 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return &snapshots[0], nil
	}
}

func (self *SRegion) GetDisks(instanceId string, zoneId string, storageType string, diskIds []string, offset int, limit int) ([]SDisk, int, error) {
	params := &ec2.DescribeVolumesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(instanceId) > 0 {
		filters = AppendSingleValueFilter(filters, "attachment.instance-id", instanceId)
	}

	if len(zoneId) > 0 {
		filters = AppendSingleValueFilter(filters, "availability-zone", zoneId)
	}

	if len(storageType) > 0 {
		filters = AppendSingleValueFilter(filters, "volume-type", storageType)
	}

	params.SetFilters(filters)

	if len(diskIds) > 0 {
		params.SetVolumeIds(ConvertedList(diskIds))
	}

	ret, err := self.ec2Client.DescribeVolumes(params)
	if err != nil {
		return nil, 0 , err
	}

	disks := []SDisk{}
	for _, item := range ret.Volumes {
		disk := SDisk{}
		disk.ZoneId = *item.AvailabilityZone
		disk.Status = *item.State
		disk.Size = int(*item.Size)
		disk.Category = *item.VolumeType
		disk.RegionId = self.RegionId
		disk.SourceSnapshotId = *item.SnapshotId
		disk.Encrypted = *item.Encrypted
		disk.DiskId = *item.VolumeId
		disk.Iops = int(*item.Iops)
		disk.CreationTime = *item.CreateTime
		if len(item.Attachments) > 0 {
			disk.DeleteWithInstance = *item.Attachments[0].DeleteOnTermination
			disk.AttachedTime = *item.Attachments[0].AttachTime
			disk.AttachmentStatus = *item.Attachments[0].State
			disk.Device = *item.Attachments[0].Device
			disk.InstanceId = *item.Attachments[0].InstanceId
			// todo: 需要通过describe-instances 的root device 判断是否是系统盘
			if len(disk.InstanceId) > 0 {
				instance, err := self.GetInstance(disk.InstanceId)
				if err != nil {
					log.Debug(err)
				}

				if disk.Device == instance.RootDeviceName {
					disk.Type = models.DISK_TYPE_SYS
				} else {
					disk.Type = models.DISK_TYPE_DATA
				}
			} else {
				disk.Type = models.DISK_TYPE_DATA
			}
		}

		disks = append(disks, disk)
	}
	return disks, len(disks), nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, total, err := self.GetDisks("", "", "", []string{diskId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &disks[0], nil
}

func (self *SRegion) DeleteDisk(diskId string) error {
	return nil
}

func (self *SRegion) resizeDisk(diskId string, size int64) error {
	return nil
}

func (self *SRegion) resetDisk(diskId, snapshotId string) error {
	return nil
}

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (string, error) {
	return "", nil
}

func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, desc string) (string, error) {
	return "", nil
}