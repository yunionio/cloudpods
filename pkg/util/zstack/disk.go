package zstack

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

type SDisk struct {
	storage *SStorage

	UUID               string    `json:"uuid"`
	Name               string    `json:"name"`
	PrimaryStorageUUID string    `json:"primaryStorageUuid"`
	VMInstanceUUID     string    `json:"vmInstanceUuid"`
	DiskOfferingUUID   string    `json:"diskOfferingUuid"`
	RootImageUUID      string    `json:"rootImageUuid"`
	InstallPath        string    `json:"installPath"`
	Type               string    `json:"Type"`
	Format             string    `json:"format"`
	Size               int       `json:"size"`
	ActualSize         int       `json:"actualSize"`
	DeviceID           float32   `json:"deviceId"`
	State              string    `json:"state"`
	Status             string    `json:"status"`
	CreateDate         time.Time `json:"createDate"`
	LastOpDate         time.Time `json:"lastOpDate"`
}

func (region *SRegion) GetDisks(storageId, diskId string) ([]SDisk, error) {
	disks := []SDisk{}
	params := []string{}
	if len(storageId) > 0 {
		params = append(params, "q=primaryStorageUuid="+storageId)
	}
	if len(diskId) > 0 {
		params = append(params, "q=uuid="+diskId)
	}
	err := region.client.listAll("volumes", params, &disks)
	if err != nil {
		return nil, err
	}
	return disks, nil
}

func (disk *SDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	//priceKey := fmt.Sprintf("%s::%s::%s", disk.RegionId, disk.Category, disk.Type)
	//data.Add(jsonutils.NewString(priceKey), "price_key")

	data.Add(jsonutils.NewString(models.HYPERVISOR_ALIYUN), "hypervisor")

	return data
}

func (disk *SDisk) GetId() string {
	return disk.UUID
}

func (disk *SDisk) Delete(ctx context.Context) error {
	_, err := disk.storage.zone.region.getDisk(disk.DiskId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			// 未找到disk, 说明disk已经被删除了. 避免回收站中disk-delete循环删除失败
			return nil
		}
		log.Errorf("Failed to find disk %s when delete: %s", disk.DiskId, err)
		return err
	}

	for {
		err := disk.storage.zone.region.DeleteDisk(disk.DiskId)
		if err != nil {
			if isError(err, "IncorrectDiskStatus") {
				log.Infof("The disk is initializing, try later ...")
				time.Sleep(10 * time.Second)
			} else {
				log.Errorf("DeleteDisk fail: %s", err)
				return err
			}
		} else {
			break
		}
	}
	return cloudprovider.WaitDeleted(disk, 10*time.Second, 300*time.Second) // 5minutes
}

func (disk *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return disk.storage.zone.region.resizeDisk(disk.DiskId, sizeMb)
}

func (disk *SDisk) GetName() string {
	if len(disk.DiskName) > 0 {
		return disk.DiskName
	}
	return disk.DiskId
}

func (disk *SDisk) GetGlobalId() string {
	return disk.DiskId
}

func (disk *SDisk) IsEmulated() bool {
	return false
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SDisk) GetStatus() string {
	// In_use Available Attaching Detaching Creating ReIniting All
	switch disk.Status {
	case "Creating", "ReIniting":
		return models.DISK_ALLOCATING
	default:
		return models.DISK_READY
	}
}

func (disk *SDisk) Refresh() error {
	new, err := disk.storage.zone.region.getDisk(disk.DiskId)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, new)
}

func (disk *SDisk) ResizeDisk(newSize int64) error {
	// newSize 单位为 GB. 范围在20 ～2000. 只能往大调。不能调小
	// https://help.aliyun.com/document_detail/25522.html?spm=a2c4g.11174283.6.897.aHwqkS
	return disk.storage.zone.region.resizeDisk(disk.DiskId, newSize)
}

func (disk *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.Size * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return disk.DeleteWithInstance
}

func (disk *SDisk) GetTemplateId() string {
	return disk.ImageId
}

func (disk *SDisk) GetDiskType() string {
	switch disk.Type {
	case "system":
		return models.DISK_TYPE_SYS
	case "data":
		return models.DISK_TYPE_DATA
	default:
		return models.DISK_TYPE_DATA
	}
}

func (disk *SDisk) GetFsFormat() string {
	return ""
}

func (disk *SDisk) GetIsNonPersistent() bool {
	return false
}

func (disk *SDisk) GetDriver() string {
	return "scsi"
}

func (disk *SDisk) GetCacheMode() string {
	return "none"
}

func (disk *SDisk) GetMountpoint() string {
	return ""
}

func (disk *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, desc string) (string, error) {
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

	body, err := disk.ecsRequest("CreateDisk", params)
	if err != nil {
		return "", err
	}
	return body.GetString("DiskId")
}

func (disk *SRegion) getDisk(diskId string) (*SDisk, error) {
	disks, total, err := disk.GetDisks("", "", "", []string{diskId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &disks[0], nil
}

func (disk *SRegion) DeleteDisk(diskId string) error {
	params := make(map[string]string)
	params["DiskId"] = diskId

	_, err := disk.ecsRequest("DeleteDisk", params)
	return err
}

func (disk *SRegion) resizeDisk(diskId string, sizeMb int64) error {
	sizeGb := sizeMb / 1024
	params := make(map[string]string)
	params["DiskId"] = diskId
	params["NewSize"] = fmt.Sprintf("%d", sizeGb)

	_, err := disk.ecsRequest("ResizeDisk", params)
	if err != nil {
		log.Errorf("resizing disk (%s) to %d GiB failed: %s", diskId, sizeGb, err)
		return err
	}

	return nil
}

func (disk *SRegion) resetDisk(diskId, snapshotId string) error {
	params := make(map[string]string)
	params["DiskId"] = diskId
	params["SnapshotId"] = snapshotId
	_, err := disk.ecsRequest("ResetDisk", params)
	if err != nil {
		log.Errorf("ResetDisk %s to snapshot %s fail %s", diskId, snapshotId, err)
		return err
	}

	return nil
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	if snapshotId, err := disk.storage.zone.region.CreateSnapshot(disk.DiskId, name, desc); err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	} else if snapshot, err := disk.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = disk.storage.zone.region
		if err := cloudprovider.WaitStatus(snapshot, models.SNAPSHOT_READY, 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return snapshot, nil
	}
}

func (disk *SRegion) CreateSnapshot(diskId, name, desc string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = disk.RegionId
	params["DiskId"] = diskId
	params["SnapshotName"] = name
	params["Description"] = desc

	if body, err := disk.ecsRequest("CreateSnapshot", params); err != nil {
		log.Errorf("CreateSnapshot fail %s", err)
		return "", err
	} else {
		return body.GetString("SnapshotId")
	}
}

func (disk *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if snapshot, err := disk.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = disk.storage.zone.region
		return snapshot, nil
	}
}

func (disk *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	if snapshots, total, err := disk.storage.zone.region.GetSnapshots("", "", "", []string{snapshotId}, 0, 1); err != nil {
		return nil, err
	} else if total != 1 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return &snapshots[0], nil
	}
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	for {
		if parts, total, err := disk.storage.zone.region.GetSnapshots("", disk.DiskId, "", []string{}, 0, 20); err != nil {
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
		snapshots[i].region = disk.storage.zone.region
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", disk.storage.zone.region.resetDisk(disk.DiskId, snapshotId)
}

func (disk *SDisk) GetBillingType() string {
	return convertChargeType(disk.DiskChargeType)
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return convertExpiredAt(disk.ExpiredTime)
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	err := disk.storage.zone.region.rebuildDisk(disk.DiskId)
	if err != nil {
		if isError(err, "IncorrectInstanceStatus") {
			return nil
		}
		log.Errorf("rebuild disk fail %s", err)
		return err
	}
	return nil
}

func (disk *SRegion) rebuildDisk(diskId string) error {
	params := make(map[string]string)
	params["DiskId"] = diskId
	_, err := disk.ecsRequest("ReInitDisk", params)
	if err != nil {
		log.Errorf("ReInitDisk %s fail %s", diskId, err)
		return err
	}
	return nil
}

func (disk *SDisk) GetProjectId() string {
	return ""
}
