package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type TStorageType string

const (
	StorageTypeCeph    = TStorageType("Ceph")
	StorageTypeLocal   = TStorageType("LocalStorage")
	StorageTypeVCenter = TStorageType("VCenter")
)

type SStorage struct {
	zone *SZone

	ZStackBasic
	VCenterUUID               string       `json:"VCenterUuid"`
	Datastore                 string       `json:"datastore"`
	ZoneUUID                  string       `json:"zoneUuid"`
	URL                       string       `json:"url"`
	TotalCapacity             int64        `json:"totalCapacity"`
	AvailableCapacity         int          `json:"availableCapacity"`
	TotalPhysicalCapacity     int          `json:"totalPhysicalCapacity"`
	AvailablePhysicalCapacity int          `json:"availablePhysicalCapacity"`
	Type                      TStorageType `json:"type"`
	State                     string       `json:"state"`
	Status                    string       `json:"status"`
	MountPath                 string       `json:"mountPath"`
	AttachedClusterUUIDs      []string     `json:"attachedClusterUuids"`
	ZStackTime
}

func (storage *SStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (storage *SStorage) GetId() string {
	return storage.UUID
}

func (storage *SStorage) GetName() string {
	return storage.Name
}

func (storage *SStorage) GetGlobalId() string {
	return storage.GetId()
}

func (storage *SStorage) IsEmulated() bool {
	return false
}

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks(storage.UUID, "")
	if err != nil {
		return nil, err
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		disks[i].storage = storage
		idisks = append(idisks, &disks[i])
	}
	return idisks, nil
}

func (storage *SStorage) GetStorageType() string {
	return string(storage.Type)
}

func (storage *SStorage) GetMediumType() string {
	return api.DISK_TYPE_ROTATE
}

func (storage *SStorage) GetCapacityMB() int64 {
	return storage.TotalCapacity / 1024 / 1024
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SStorage) GetManagerId() string {
	return storage.zone.region.client.providerID
}

func (storage *SStorage) GetStatus() string {
	if storage.Status == "Connected" {
		return api.STORAGE_ONLINE
	}
	return api.STORAGE_OFFLINE
}

func (storage *SStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SStorage) GetEnabled() bool {
	return storage.State == "Enabled"
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	storage.zone.region.GetIStoragecaches()
	return storage.zone.region.storageCache
}

func (storage *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	// diskId, err := storage.zone.region.CreateDisk(storage.zone.ZoneId, storage.storageType, name, sizeGb, desc)
	// if err != nil {
	// 	log.Errorf("createDisk fail %s", err)
	// 	return nil, err
	// }
	// disk, err := storage.zone.region.getDisk(diskId)
	// if err != nil {
	// 	log.Errorf("getDisk fail %s", err)
	// 	return nil, err
	// }
	// disk.storage = storage
	// return disk, nil
	return nil, cloudprovider.ErrNotImplemented
}

func (storage *SStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks(storage.UUID, diskId)
	if err != nil {
		return nil, err
	}
	if len(disks) == 1 {
		if disks[0].UUID == diskId {
			disks[0].storage = storage
			return &disks[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(disks) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (storage *SStorage) GetMountPoint() string {
	switch storage.Type {
	case StorageTypeLocal:
		return storage.MountPath
	case StorageTypeCeph:
		return ""
	case StorageTypeVCenter:
		return storage.URL
	default:
		log.Errorf("Unknown storage type %s", storage.Type)
	}
	return ""
}

func (storage *SStorage) IsSysDiskStore() bool {
	return false
}
