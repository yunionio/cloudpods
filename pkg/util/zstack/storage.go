package zstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type TStorageType string

const (
	StorageTypeCeph    = TStorageType("Ceph")
	StorageTypeLocal   = TStorageType("LocalStorage")
	StorageTypeVCenter = TStorageType("VCenter")
)

type SStorage struct {
	zone *SZone

	Name string
	UUID string

	VCenterUUID               string       `json:"VCenterUuid"`
	Datastore                 string       `json:"datastore"`
	ZoneUUID                  string       `json:"zoneUuid"`
	URL                       string       `json:"url"`
	TotalCapacity             int          `json:"totalCapacity"`
	AvailableCapacity         int          `json:"availableCapacity"`
	TotalPhysicalCapacity     int          `json:"totalPhysicalCapacity"`
	AvailablePhysicalCapacity int          `json:"availablePhysicalCapacity"`
	Type                      TStorageType `json:"type"`
	State                     string       `json:"state"`
	Status                    string       `json:"status"`
	MountPath                 string       `json:"mountPath"`
	AttachedClusterUUIDs      []string     `json:"attachedClusterUuids"`

	//CreateDate                time.Time `json:"createDate"`
	//LastOpDate                time.Time `json:"lastOpDate"`
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
	return fmt.Sprintf("ZStack/%s", storage.Type)
}

func (storage *SStorage) GetMediumType() string {
	return models.DISK_TYPE_ROTATE
}

func (storage *SStorage) GetCapacityMB() int {
	return storage.TotalCapacity / 1024
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
		return models.STORAGE_ONLINE
	}
	return models.STORAGE_OFFLINE
}

func (storage *SStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SStorage) GetEnabled() bool {
	return storage.State == "Enabled"
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
	//return storage.zone.region.getStoragecache()
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

func (storage *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
	// if disk, err := storage.zone.region.getDisk(idStr); err != nil {
	// 	return nil, err
	// } else {
	// 	disk.storage = storage
	// 	return disk, nil
	// }
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
