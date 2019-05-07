package zstack

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type TStorageType string

const (
	StorageTypeCeph    = TStorageType("Ceph")
	StorageTypeLocal   = TStorageType("LocalStorage")
	StorageTypeVCenter = TStorageType("VCenter")
)

type SPrimaryStorage struct {
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

	Pools []SCephStorage `json:"pools"`

	ZStackTime
}

func (region *SRegion) GetPrimaryStorage(storageId string) (*SPrimaryStorage, error) {
	storages, err := region.GetPrimaryStorages("", "", storageId)
	if err != nil {
		return nil, err
	}
	if len(storages) == 1 {
		if storages[0].UUID == storageId {
			return &storages[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(storages) == 0 || len(storageId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetPrimaryStorages(zoneId, clusterId, storageId string) ([]SPrimaryStorage, error) {
	storages := []SPrimaryStorage{}
	params := []string{}
	if len(zoneId) > 0 {
		params = append(params, "q=zone.uuid="+zoneId)
	}
	if len(clusterId) > 0 {
		params = append(params, "q=cluster.uuid="+clusterId)
	}
	if SkipEsxi {
		params = append(params, "q=type!=VCenter")
	}
	if len(storageId) > 0 {
		params = append(params, "q=uuid="+storageId)
	}
	return storages, region.client.listAll("primary-storage", params, &storages)
}

func (storage *SPrimaryStorage) GetStatus() string {
	if storage.Status == "Connected" {
		return api.STORAGE_ONLINE
	}
	return api.STORAGE_OFFLINE
}
