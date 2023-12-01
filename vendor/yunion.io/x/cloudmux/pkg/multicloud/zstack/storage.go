// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zstack

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type TStorageType string

type TCephPoolType string

const (
	StorageTypeCeph    = TStorageType("Ceph")
	StorageTypeLocal   = TStorageType("LocalStorage")
	StorageTypeVCenter = TStorageType("VCenter")

	CephPoolTypeData       = TCephPoolType("Data")
	CephPoolTypeRoot       = TCephPoolType("Root")
	CephPoolTypeImageCache = TCephPoolType("ImageCache")
)

type SPool struct {
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

type SStorage struct {
	multicloud.SStorageBase
	ZStackTags
	region *SRegion

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

	Pools []SPool `json:"pools"`

	ZStackTime
}

func (region *SRegion) getIStorages(zondId string) ([]cloudprovider.ICloudStorage, error) {
	primaryStorages, err := region.GetStorages(zondId, "", "")
	if err != nil {
		return nil, err
	}
	istorage := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(primaryStorages); i++ {
		primaryStorage := primaryStorages[i]
		switch primaryStorage.Type {
		case StorageTypeLocal:
			ilocalStorages, err := region.getILocalStorages(primaryStorage.UUID, "")
			if err != nil {
				return nil, err
			}
			istorage = append(istorage, ilocalStorages...)
		default:
			primaryStorage.region = region
			istorage = append(istorage, &primaryStorage)
		}
	}
	return istorage, nil
}

func (region *SRegion) GetStorage(storageId string) (*SStorage, error) {
	if len(storageId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	storages, err := region.GetStorages("", "", storageId)
	if err != nil {
		return nil, err
	}
	if len(storages) == 1 && storages[0].UUID == storageId {
		storages[0].region = region
		return &storages[0], nil
	}
	if len(storages) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetStorages(zoneId, clusterId, storageId string) ([]SStorage, error) {
	storages := []SStorage{}
	params := url.Values{}
	if len(zoneId) > 0 {
		params.Add("q", "zone.uuid="+zoneId)
	}
	if len(clusterId) > 0 {
		params.Add("q", "cluster.uuid="+clusterId)
	}
	if SkipEsxi {
		params.Add("q", "type!=VCenter")
	}
	if len(storageId) > 0 {
		params.Add("q", "uuid="+storageId)
	}
	return storages, region.client.listAll("primary-storage", params, &storages)
}

func (storage *SStorage) GetStatus() string {
	if storage.Status == "Connected" {
		return api.STORAGE_ONLINE
	}
	return api.STORAGE_OFFLINE
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

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	zone, err := storage.region.GetZone(storage.ZoneUUID)
	if err != nil {
		log.Errorf("failed to find zone for storage %s(%s)", storage.Name, storage.UUID)
		return nil
	}
	return zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.region.GetDisks(storage.UUID, []string{}, "")
	if err != nil {
		return nil, err
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		disks[i].storage = storage
		disks[i].region = storage.region
		idisks = append(idisks, &disks[i])
	}
	return idisks, nil
}

func (storage *SStorage) GetStorageType() string {
	return strings.ToLower(string(storage.Type))
}

func (storage *SStorage) GetMediumType() string {
	return api.DISK_TYPE_ROTATE
}

func (storage *SStorage) GetCapacityMB() int64 {
	return storage.TotalCapacity / 1024 / 1024
}

func (storage *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SStorage) GetEnabled() bool {
	return true
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return &SStoragecache{ZoneId: storage.ZoneUUID, region: storage.region}
}

func (storage *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	poolName, err := storage.GetDataPoolName()
	if err != nil {
		return nil, err
	}
	disk, err := storage.region.CreateDisk(conf.Name, storage.UUID, "", poolName, conf.SizeGb, conf.Desc)
	if err != nil {
		return nil, err
	}
	disk.storage = storage
	return disk, nil
}

func (storage *SStorage) GetIDiskById(diskId string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.region.GetDisk(diskId)
	if err != nil {
		return nil, err
	}
	if disk.PrimaryStorageUUID != storage.UUID {
		return nil, cloudprovider.ErrNotFound
	}
	disk.region = storage.region
	disk.storage = storage
	return disk, nil
}

func (storage *SStorage) GetMountPoint() string {
	poolName, _ := storage.GetDataPoolName()
	if len(poolName) > 0 {
		return fmt.Sprintf("ceph://%s", poolName)
	}
	return ""
}

func (storage *SStorage) IsSysDiskStore() bool {
	return true
}

func (storage *SStorage) GetDataPoolName() (string, error) {
	for _, pool := range storage.Pools {
		if pool.Type == CephPoolTypeData {
			return pool.PoolName, nil
		}
	}
	return "", fmt.Errorf("failed to found storage %s(%s) data pool name", storage.Name, storage.UUID)
}
