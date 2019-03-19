package ucloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

// https://docs.ucloud.cn/api/udisk-api/describe_udisk
type SDisk struct {
	storage *SStorage

	Status        string `json:"Status"`
	DeviceName    string `json:"DeviceName"`
	UHostID       string `json:"UHostId"`
	Tag           string `json:"Tag"`
	Version       string `json:"Version"`
	Name          string `json:"Name"`
	Zone          string `json:"Zone"`
	UHostIP       string `json:"UHostIP"`
	DiskType      string `json:"DiskType"`
	UDataArkMode  string `json:"UDataArkMode"`
	SnapshotLimit int    `json:"SnapshotLimit"`
	ExpiredTime   int64  `json:"ExpiredTime"`
	SnapshotCount int    `json:"SnapshotCount"`
	IsExpire      string `json:"IsExpire"`
	UDiskID       string `json:"UDiskId"`
	ChargeType    string `json:"ChargeType"`
	UHostName     string `json:"UHostName"`
	CreateTime    int64  `json:"CreateTime"`
	SizeGB        int    `json:"Size"`
}

func (self *SDisk) GetProjectId() string {
	return self.storage.zone.region.client.projectId
}

func (self *SDisk) GetId() string {
	return self.UDiskID
}

func (self *SDisk) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) GetStatus() string {
	switch self.Status {
	case "Available":
		return models.DISK_READY
	case "Attaching":
		return models.DISK_ATTACHING
	case "InUse":
		return models.DISK_READY
	case "Detaching":
		return models.DISK_DETACHING
	case "Initializating":
		return models.DISK_ALLOCATING
	case "Failed":
		return models.DISK_ALLOC_FAILED
	case "Cloning":
		return models.DISK_CLONING
	case "Restoring":
		return models.DISK_RESET
	case "RestoreFailed":
		return models.DISK_RESET_FAILED
	default:
		return models.DISK_UNKNOWN
	}
}

func (self *SDisk) Refresh() error {
	new, err := self.storage.zone.region.GetDisk(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	// todo: add price key
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(models.HYPERVISOR_UCLOUD), "hypervisor")

	return data
}

// Year,Month,Dynamic,Trial
func (self *SDisk) GetBillingType() string {
	switch self.ChargeType {
	case "Year", "Month":
		return models.BILLING_TYPE_PREPAID
	default:
		return models.BILLING_TYPE_POSTPAID
	}
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Unix(self.ExpiredTime, 0)
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.SizeGB * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	if self.DiskType == "SystemDisk" {
		return true
	}

	return false
}

func (self *SDisk) GetTemplateId() string {
	if strings.Contains(self.DiskType, "SystemDisk") && len(self.UHostID) > 0 {
		ins, err := self.storage.zone.region.GetInstanceByID(self.UHostID)
		if err != nil {
			log.Errorf(err.Error())
		}

		return ins.ImageID
	}

	return ""
}

func (self *SDisk) GetDiskType() string {
	if strings.Contains(self.DiskType, "SystemDisk") {
		return models.DISK_TYPE_SYS
	}

	return models.DISK_TYPE_DATA
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

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshotById(snapshotId)
	return &snapshot, err
}

func (self *SDisk) GetISnapshot(idStr string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.getSnapshot(idStr)
	return snapshot, err
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.GetSnapshots(self.GetId(), "")
	if err != nil {
		return nil, err
	}

	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		return nil, fmt.Errorf("GetDisk id should not empty")
	}

	disks, err := self.GetDisks("", "", []string{diskId})
	if err != nil {
		return nil, err
	}

	if len(disks) == 1 {
		return &disks[0], nil
	} else if len(disks) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, fmt.Errorf("GetDisk %s %d found", diskId, len(disks))
	}
}

// https://docs.ucloud.cn/api/udisk-api/describe_udisk
// diskType DataDisk|SystemDisk (DataDisk表示数据盘，SystemDisk表示系统盘)
func (self *SRegion) GetDisks(zoneId string, diskType string, diskIds []string) ([]SDisk, error) {
	disks := make([]SDisk, 0)
	params := NewUcloudParams()
	if len(zoneId) > 0 {
		params.Set("Zone", zoneId)
	}

	if len(diskType) > 0 {
		params.Set("DiskType", diskType)
	}

	err := self.DoListAll("DescribeUDisk", params, &disks)
	if err != nil {
		return nil, err
	}

	if len(diskIds) > 0 {
		filtedDisks := make([]SDisk, 0)
		for _, disk := range disks {
			if utils.IsInStringArray(disk.UDiskID, diskIds) {
				filtedDisks = append(filtedDisks, disk)
			}
		}

		return filtedDisks, nil
	}

	return disks, nil
}
