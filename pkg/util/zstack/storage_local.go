package zstack

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLocalStorage struct {
	zone *SZone

	primaryStorageID          string
	HostUUID                  string `json:"hostUuid"`
	TotalCapacity             int64  `json:"totalCapacity"`
	AvailableCapacity         int64  `json:"availableCapacity"`
	TotalPhysicalCapacity     int64  `json:"totalPhysicalCapacity"`
	AvailablePhysicalCapacity int64  `json:"availablePhysicalCapacity"`
}

func (region *SRegion) GetLocalStorage(storageId string, hostId string) (*SLocalStorage, error) {
	storages, err := region.GetLocalStorages(storageId, hostId)
	if err != nil {
		return nil, err
	}
	if len(storages) == 1 {
		if storages[0].HostUUID == hostId {
			return &storages[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(storages) == 0 || len(storageId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetLocalStorages(storageId string, hostId string) ([]SLocalStorage, error) {
	localStorage := []SLocalStorage{}
	params := []string{}
	if len(hostId) > 0 {
		params = append(params, "hostUuid="+hostId)
	}
	err := region.client.listAll(fmt.Sprintf("primary-storage/local-storage/%s/capacities", storageId), params, &localStorage)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(localStorage); i++ {
		localStorage[i].primaryStorageID = storageId
	}
	return localStorage, nil
}

func (region *SRegion) getILocalStorages(zone *SZone, storageId, hostId string) ([]cloudprovider.ICloudStorage, error) {
	storages, err := region.GetLocalStorages(storageId, hostId)
	if err != nil {
		return nil, err
	}
	istorage := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
		storages[i].zone = zone
		istorage = append(istorage, &storages[i])
	}
	return istorage, nil
}

func (storage *SLocalStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (storage *SLocalStorage) GetId() string {
	return fmt.Sprintf("%s/%s", storage.primaryStorageID, storage.HostUUID)
}

func (storage *SLocalStorage) GetName() string {
	primaryStorage, err := storage.zone.region.GetPrimaryStorage(storage.primaryStorageID)
	if err != nil {
		return "Unknown"
	}
	host, err := storage.zone.region.GetHost(storage.zone.UUID, storage.HostUUID)
	if err != nil {
		return "Unknown"
	}
	return fmt.Sprintf("%s/%s", primaryStorage.Name, host.Name)
}

func (storage *SLocalStorage) GetGlobalId() string {
	return storage.GetId()
}

func (storage *SLocalStorage) IsEmulated() bool {
	return false
}

func (storage *SLocalStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SLocalStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	tags, err := storage.zone.region.GetSysTags("", "VolumeVO", "", "localStorage::hostUuid::"+storage.HostUUID)
	if err != nil {
		return nil, err
	}
	diskIds := []string{}
	for i := 0; i < len(tags); i++ {
		diskIds = append(diskIds, tags[i].ResourceUUID)
	}
	disks, err := storage.zone.region.GetDisks(storage.primaryStorageID, diskIds, "")
	if err != nil {
		return nil, err
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		disks[i].localStorage = storage
		disks[i].region = storage.zone.region
		idisks = append(idisks, &disks[i])
	}
	return idisks, nil
}

func (storage *SLocalStorage) GetStorageType() string {
	return strings.ToLower(string(StorageTypeLocal))
}

func (storage *SLocalStorage) GetMediumType() string {
	return api.DISK_TYPE_ROTATE
}

func (storage *SLocalStorage) GetCapacityMB() int64 {
	return storage.TotalCapacity / 1024 / 1024
}

func (storage *SLocalStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SLocalStorage) GetManagerId() string {
	return storage.zone.region.client.providerID
}

func (storage *SLocalStorage) GetStatus() string {
	primaryStorage, err := storage.zone.region.GetPrimaryStorage(storage.primaryStorageID)
	if err != nil {
		return api.STORAGE_OFFLINE
	}
	return primaryStorage.GetStatus()
}

func (storage *SLocalStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SLocalStorage) GetEnabled() bool {
	return true
}

func (storage *SLocalStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	storage.zone.region.GetIStoragecaches()
	return storage.zone.region.storageCache
}

func (storage *SLocalStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.CreateDisk(name, storage.primaryStorageID, storage.HostUUID, "", sizeGb, desc)
	if err != nil {
		return nil, err
	}
	disk.localStorage = storage
	return disk, nil
}

func (storage *SLocalStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	tags, err := storage.zone.region.GetSysTags("", "VolumeVO", diskId, "localStorage::hostUuid::"+storage.HostUUID)
	if err != nil {
		return nil, err
	}
	if len(tags) == 1 {
		disk, err := storage.zone.region.GetDisk(diskId)
		if err != nil {
			return nil, err
		}
		disk.localStorage = storage
		disk.region = storage.zone.region
		return disk, nil
	}
	if len(tags) == 0 || len(diskId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (storage *SLocalStorage) GetMountPoint() string {
	return ""
}

func (storage *SLocalStorage) IsSysDiskStore() bool {
	return false
}
