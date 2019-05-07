package zstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SDisk struct {
	localStorage *SLocalStorage
	cephStorage  *SCephStorage
	region       *SRegion

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

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, err := region.GetDisks("", []string{diskId}, "")
	if err != nil {
		return nil, err
	}
	if len(disks) == 1 {
		if disks[0].UUID == diskId {
			return &disks[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(disks) == 0 || len(diskId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetDiskWithStorage(diskId string) (*SDisk, error) {
	disk, err := region.GetDisk(diskId)
	if err != nil {
		return nil, err
	}
	disk.region = region
	storage, err := region.GetPrimaryStorage(disk.PrimaryStorageUUID)
	if err != nil {
		return nil, err
	}
	switch storage.Type {
	case StorageTypeLocal:
		tags, err := region.GetSysTags("", "VolumeVO", disk.UUID, "")
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(tags); i++ {
			if strings.HasPrefix(tags[i].Tag, "localStorage::hostUuid::") {
				hostInfo := strings.Split(tags[i].Tag, "localStorage::hostUuid::")
				if len(hostInfo) == 2 {
					localStorage, err := region.GetLocalStorage(storage.UUID, hostInfo[1])
					if err != nil {
						return nil, err
					}
					disk.localStorage = localStorage
					return disk, nil
				}
				return nil, fmt.Errorf("invalid host info %s from disk %s", tags[i].Tag, disk.Name)
			}
		}
		return nil, cloudprovider.ErrNotFound
	case StorageTypeCeph:
		storage, err := region.GetPrimaryStorage(disk.PrimaryStorageUUID)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(storage.Pools); i++ {
			if strings.Contains(disk.InstallPath, storage.Pools[i].PoolName) {
				zone, err := region.GetZone(storage.ZoneUUID)
				if err != nil {
					return nil, err
				}
				cephStorage := storage.Pools[i]
				cephStorage.zone = zone
				disk.cephStorage = &cephStorage
				return disk, nil
			}
		}
		return nil, fmt.Errorf("failed to found ceph storage for disk %s", disk.Name)
	default:
		return nil, fmt.Errorf("Unsupport StorageType %s", storage.Type)
	}
}

func (region *SRegion) GetDisks(storageId string, diskIds []string, diskType string) ([]SDisk, error) {
	disks := []SDisk{}
	params := []string{}
	if len(storageId) > 0 {
		params = append(params, "q=primaryStorageUuid="+storageId)
	}
	if len(diskIds) > 0 {
		params = append(params, "q=uuid?="+strings.Join(diskIds, ","))
	}
	if len(diskType) > 0 {
		params = append(params, "q=type="+diskType)
	}
	return disks, region.client.listAll("volumes", params, &disks)
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
	return disk.region.DeleteDisk(disk.UUID)
}

func (disk *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return disk.region.ResizeDisk(disk.UUID, sizeMb)
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
	if disk.localStorage != nil {
		return disk.localStorage, nil
	}
	if disk.cephStorage != nil {
		return disk.cephStorage, nil
	}
	return nil, cloudprovider.ErrNotFound
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
	new, err := disk.region.GetDisk(disk.UUID)
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

func (region *SRegion) CreateDisk(name string, storageId string, hostId string, poolName string, sizeGb int, desc string) (*SDisk, error) {
	offerings, err := region.GetDiskOfferings(sizeGb)
	if err != nil {
		return nil, err
	}
	diskOfferingUuid := ""
	if len(offerings) > 0 {
		diskOfferingUuid = offerings[0].UUID
	} else {
		offering, err := region.CreateDiskOffering(sizeGb)
		if err != nil {
			return nil, err
		}
		diskOfferingUuid = offering.UUID
		defer region.DeleteDiskOffering(diskOfferingUuid)
	}
	params := map[string]interface{}{
		"params": map[string]string{
			"name":               name,
			"description":        desc,
			"diskOfferingUuid":   diskOfferingUuid,
			"primaryStorageUuid": storageId,
		},
	}
	if len(hostId) > 0 {
		params["systemTags"] = []string{"localStorage::hostUuid::" + hostId}
	}
	if len(poolName) > 0 {
		params["systemTags"] = []string{"ceph::pool::" + poolName}
	}
	resp, err := region.client.post("volumes/data", jsonutils.Marshal(params))
	disk := &SDisk{}
	return disk, resp.Unmarshal(disk, "inventory")
}

func (region *SRegion) DeleteDisk(diskId string) error {
	return region.client.delete("volumes", diskId, "Enforcing")
}

func (region *SRegion) ResizeDisk(diskId string, sizeMb int64) error {
	disk, err := region.GetDisk(diskId)
	if err != nil {
		return err
	}
	params := jsonutils.Marshal(map[string]interface{}{
		fmt.Sprintf("resize%sVolume", disk.Type): map[string]int64{
			"size": sizeMb * 1024 * 1024,
		},
	})
	resource := "volumes/resize"
	if disk.Type == "Data" {
		resource = "volumes/data/resize"
	}
	_, err = region.client.put(resource, diskId, params)
	return err
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return disk.region.CreateSnapshot(name, disk.UUID, desc)
}

func (disk *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.region.GetSnapshots(snapshotId, disk.UUID)
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
	snapshots, err := disk.region.GetSnapshots("", disk.UUID)
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
	_, err := disk.region.ResetDisks(snapshotId)
	return disk.UUID, err
}

func (region *SRegion) ResetDisks(snapshotId string) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"revertVolumeFromSnapshot": jsonutils.NewDict(),
	}
	return region.client.put("volume-snapshots", snapshotId, jsonutils.Marshal(params))
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
	return disk.region.RebuildDisk(disk.UUID)
}

func (region *SRegion) RebuildDisk(diskId string) error {
	params := map[string]interface{}{
		"recoverDataVolume": jsonutils.NewDict(),
	}
	_, err := region.client.put("volumes", diskId, jsonutils.Marshal(params))
	return err
}

func (disk *SDisk) GetProjectId() string {
	return ""
}
