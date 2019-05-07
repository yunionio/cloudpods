package zstack

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type TCephPoolType string

const (
	CephPoolTypeData       = TCephPoolType("Data")
	CephPoolTypeRoot       = TCephPoolType("Root")
	CephPoolTypeImageCache = TCephPoolType("ImageCache")
)

type SCephStorage struct {
	zone *SZone

	UUID               string        `json:"uuid"`
	PrimaryStorageUUID string        `json:"primaryStorageUuid"`
	PoolName           string        `json:"poolName"`
	Type               TCephPoolType `json:"type"`
	AvailableCapacity  int64         `json:"availableCapacity"`
	UsedCapacity       int64         `json:"usedCapacity"`
	ReplicatedSize     int64         `json:"replicatedSize"`
	TotalCapacity      int64         `json:"totalCapacity"`
	ZStackTime
}

func (region *SRegion) getICephStorages(zone *SZone, storageId string) ([]cloudprovider.ICloudStorage, error) {
	storage, err := region.GetPrimaryStorage(storageId)
	if err != nil {
		return nil, err
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storage.Pools); i++ {
		storage.Pools[i].zone = zone
		istorages = append(istorages, &storage.Pools[i])
	}
	return istorages, nil
}

func (storage *SCephStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (storage *SCephStorage) GetId() string {
	return fmt.Sprintf("%s/%s", storage.PrimaryStorageUUID, storage.UUID)
}

func (storage *SCephStorage) GetName() string {
	primaryStorage, err := storage.zone.region.GetPrimaryStorage(storage.PrimaryStorageUUID)
	if err != nil {
		return "Unknown"
	}
	return fmt.Sprintf("%s/%s", primaryStorage.Name, storage.PoolName)
}

func (storage *SCephStorage) GetGlobalId() string {
	return storage.GetId()
}

func (storage *SCephStorage) IsEmulated() bool {
	return false
}

func (storage *SCephStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SCephStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if storage.Type == CephPoolTypeImageCache {
		return []cloudprovider.ICloudDisk{}, nil
	}
	disks, err := storage.zone.region.GetDisks(storage.PrimaryStorageUUID, []string{}, string(storage.Type))
	if err != nil {
		return nil, err
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		disks[i].cephStorage = storage
		disks[i].region = storage.zone.region
		idisks = append(idisks, &disks[i])
	}
	return idisks, nil
}

func (storage *SCephStorage) GetStorageType() string {
	return strings.ToLower(string(storage.Type))
}

func (storage *SCephStorage) GetMediumType() string {
	return api.DISK_TYPE_ROTATE
}

func (storage *SCephStorage) GetCapacityMB() int64 {
	return storage.TotalCapacity / 1024 / 1024
}

func (storage *SCephStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SCephStorage) GetManagerId() string {
	return storage.zone.region.client.providerID
}

func (storage *SCephStorage) GetStatus() string {
	primaryStorage, err := storage.zone.region.GetPrimaryStorage(storage.PrimaryStorageUUID)
	if err != nil {
		return api.STORAGE_OFFLINE
	}
	return primaryStorage.GetStatus()
}

func (storage *SCephStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SCephStorage) GetEnabled() bool {
	return true
}

func (storage *SCephStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	storage.zone.region.GetIStoragecaches()
	return storage.zone.region.storageCache
}

func (storage *SCephStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	if storage.Type == CephPoolTypeData {
		disk, err := storage.zone.region.CreateDisk(name, storage.PrimaryStorageUUID, "", storage.PoolName, sizeGb, desc)
		if err != nil {
			return nil, err
		}
		disk.cephStorage = storage
		return disk, nil
	}
	return nil, cloudprovider.ErrNotSupported
}

func (storage *SCephStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks(storage.PrimaryStorageUUID, []string{diskId}, storage.PoolName)
	if err != nil {
		return nil, err
	}
	if len(disks) == 1 {
		if disks[0].UUID == diskId {
			disks[0].region = storage.zone.region
			disks[0].cephStorage = storage
			return &disks[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(disks) == 0 || len(diskId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (storage *SCephStorage) GetMountPoint() string {
	return "ceph://" + storage.PoolName
}

func (storage *SCephStorage) IsSysDiskStore() bool {
	return storage.Type == CephPoolTypeRoot
}
