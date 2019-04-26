package zstack

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDisk struct {
	storage *SStorage

	ZStackBasic
	PrimaryStorageUUID string  `json:"primaryStorageUuid"`
	VMInstanceUUID     string  `json:"vmInstanceUuid"`
	DiskOfferingUUID   string  `json:"diskOfferingUuid"`
	RootImageUUID      string  `json:"rootImageUuid"`
	InstallPath        string  `json:"installPath"`
	Type               string  `json:"Type"`
	Format             string  `json:"format"`
	Size               int     `json:"size"`
	ActualSize         int     `json:"actualSize"`
	DeviceID           float32 `json:"deviceId"`
	State              string  `json:"state"`
	Status             string  `json:"status"`

	ZStackTime
}

func (region *SRegion) GetDiskWithStorage(diskId string) (*SDisk, error) {
	disk, err := region.GetDisk(diskId)
	if err != nil {
		log.Errorf("failed to found disk %s error: %v", diskId, err)
		return nil, err
	}
	storage, err := region.GetStorageWithZone(disk.PrimaryStorageUUID)
	if err != nil {
		log.Errorf("failed to found storage %s for disk %s error: %v", disk.PrimaryStorageUUID, disk.Name, err)
		return nil, err
	}
	disk.storage = storage
	return disk, nil
}

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, err := region.GetDisks("", diskId)
	if err != nil {
		return nil, err
	}
	if len(disks) == 1 {
		if disks[0].UUID == diskId {
			return &disks[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(disks) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
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

	data.Add(jsonutils.NewString(api.HYPERVISOR_ZSTACK), "hypervisor")
	return data
}

func (disk *SDisk) GetId() string {
	return disk.UUID
}

func (disk *SDisk) Delete(ctx context.Context) error {
	return disk.storage.zone.region.DeleteDisk(disk.UUID)
}

func (disk *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return disk.storage.zone.region.ResizeDisk(disk.UUID, disk.GetDiskType(), sizeMb)
}

func (disk *SDisk) GetName() string {
	return disk.Name
}

func (disk *SDisk) GetGlobalId() string {
	return disk.UUID
}

func (disk *SDisk) IsEmulated() bool {
	return false
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case "Ready":
		return api.DISK_READY
	case "NotInstantiated":
		return api.DISK_READY
	default:
		log.Errorf("Unknown disk %s(%s) status %s", disk.Name, disk.UUID, disk.Status)
		return api.DISK_UNKNOWN
	}
}

func (disk *SDisk) Refresh() error {
	new, err := disk.storage.zone.region.GetDisks("", disk.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, new)
}

func (disk *SDisk) GetDiskFormat() string {
	return disk.Format
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.Size / 1024 / 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return false
}

func (disk *SDisk) GetTemplateId() string {
	return disk.RootImageUUID
}

func (disk *SDisk) GetDiskType() string {
	switch disk.Type {
	case "Root":
		return api.DISK_TYPE_SYS
	default:
		return api.DISK_TYPE_DATA
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

func (region *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, desc string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeleteDisk(diskId string) error {
	_, err := region.client.delete("volumes", diskId, "Enforcing")
	return err
}

func (region *SRegion) ResizeDisk(diskId string, diskType string, sizeMb int64) error {
	switch diskType {
	case api.DISK_TYPE_SYS:
		diskType = "Root"
	default:
		diskType = "Data"
	}
	params := jsonutils.Marshal(map[string]interface{}{
		fmt.Sprintf("resize%sVolume", diskType): map[string]int64{
			"size": sizeMb * 1024 * 1024,
		},
	})
	_, err := region.client.put("volumes/resize", diskId, params)
	return err
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (disk *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots(snapshotId, disk.UUID)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 1 {
		if snapshots[0].UUID == snapshotId {
			return &snapshots[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(snapshots) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return nil, cloudprovider.ErrNotFound
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots("", disk.UUID)
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (disk *SDisk) GetBillingType() string {
	return ""
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreateDate
}

func (disk *SDisk) GetAccessPath() string {
	return disk.InstallPath
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return disk.storage.zone.region.RebuildDisk(disk.UUID)
}

func (region *SRegion) RebuildDisk(diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (disk *SDisk) GetProjectId() string {
	return ""
}
