package aliyun

import (
	"fmt"
	"time"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"
)

type SMountInstances struct {
	MountInstance []string
}

type STags struct {
	Tag []string
}

type SDisk struct {
	storage *SStorage

	AttachedTime                  time.Time
	AutoSnapshotPolicyId          string
	Category                      string
	CreationTime                  time.Time
	DeleteAutoSnapshot            bool
	DeleteWithInstance            bool
	Description                   string
	DetachedTime                  time.Time
	Device                        string
	DiskChargeType                string
	DiskId                        string
	DiskName                      string
	EnableAutoSnapshot            bool
	EnableAutomatedSnapshotPolicy bool
	Encrypted                     bool
	ExpiredTime                   time.Time
	ImageId                       string
	InstanceId                    string
	MountInstances                SMountInstances
	OperationLocks                SOperationLocks
	Portable                      bool
	ProductCode                   string
	RegionId                      string
	ResourceGroupId               string
	Size                          int
	SourceSnapshotId              string
	Status                        string
	Tags                          STags
	Type                          string
	ZoneId                        string
}

func (self *SRegion) GetDisks(instanceId string, zoneId string, category string, diskIds []string, offset int, limit int) ([]SDisk, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	if len(zoneId) > 0 {
		params["ZoneId"] = zoneId
	}
	if len(category) > 0 {
		params["Category"] = category
	}
	if diskIds != nil && len(diskIds) > 0 {
		params["DiskIds"] = jsonutils.Marshal(diskIds).String()
	}

	body, err := self.ecsRequest("DescribeDisks", params)
	if err != nil {
		log.Errorf("GetDisks fail %s", err)
		return nil, 0, err
	}

	disks := make([]SDisk, 0)
	err = body.Unmarshal(&disks, "Disks", "Disk")
	if err != nil {
		log.Errorf("Unmarshal disk details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return disks, int(total), nil
}

func (self *SDisk) GetId() string {
	return self.DiskId
}

func (self *SDisk) GetName() string {
	return self.DiskId
}

func (self *SDisk) GetGlobalId() string {
	return self.DiskId
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetIStorge() cloudprovider.ICloudStorage {
	return self.storage
}

func (self *SDisk) GetStatus() string {
	// In_use Available Attaching Detaching Creating ReIniting All
	switch self.Status {
	case "Creating", "ReIniting":
		return models.DISK_ALLOCATING
	default:
		return models.DISK_READY
	}
}

func (self *SDisk) Refresh() error {
	new, err := self.storage.zone.region.getDisk(self.DiskId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
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
	switch self.Type {
	case "system":
		return models.DISK_TYPE_SYS
	case "data":
		return models.DISK_TYPE_DATA
	default:
		return models.DISK_TYPE_DATA
	}
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

func (self *SRegion) createDisk(zoneId string, category string, name string, sizeGb int, desc string) (string, error) {
	params := make(map[string]string)
	params["ZoneId"] = zoneId
	params["DiskName"] = name
	if len(desc) > 0 {
		params["Description"] = desc
	}
	params["Encrypted"] = "false"
	params["DiskCategory"] = category
	params["Size"] = fmt.Sprintf("%d", sizeGb)
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := self.ecsRequest("CreateDisk", params)
	if err != nil {
		return "", err
	}
	return body.GetString("DiskId")
}

func (self *SRegion) getDisk(diskId string) (*SDisk, error) {
	disks, total, err := self.GetDisks("", "", "", []string{diskId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &disks[0], nil
}

func (self *SRegion) deleteDisk(diskId string) error {
	params := make(map[string]string)
	params["DiskId"] = diskId

	_, err := self.ecsRequest("DeleteDisk", params)
	return err
}
