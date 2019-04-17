package openstack

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SExtraSpecs struct {
	VolumeBackendName string
}

type SStorage struct {
	zone       *SZone
	Name       string
	ExtraSpecs SExtraSpecs
	ID         string
}

func (storage *SStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (storage *SStorage) GetId() string {
	return storage.ID
}

func (storage *SStorage) GetName() string {
	return storage.Name
}

func (storage *SStorage) GetGlobalId() string {
	return storage.ID
}

func (storage *SStorage) IsEmulated() bool {
	return true
}

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks(storage.Name, storage.ExtraSpecs.VolumeBackendName)
	if err != nil {
		return nil, err
	}
	iDisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		disks[i].storage = storage
		iDisks = append(iDisks, &disks[i])
	}
	return iDisks, nil
}

func (storage *SStorage) GetStorageType() string {
	return strings.ToLower(storage.Name)
}

func (storage *SStorage) GetMediumType() string {
	if strings.Contains(storage.Name, "SSD") {
		return api.DISK_TYPE_SSD
	}
	return api.DISK_TYPE_ROTATE
}

func (storage *SStorage) GetCapacityMB() int {
	return 0 // unlimited
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SStorage) GetManagerId() string {
	return storage.zone.region.client.providerID
}

func (storage *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (storage *SStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SStorage) GetEnabled() bool {
	return true
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return storage.zone.region.getStoragecache()
}

func (storage *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.CreateDisk("", storage.Name, name, sizeGb, desc)
	if err != nil {
		log.Errorf("createDisk fail %v", err)
		return nil, err
	}
	disk.storage = storage
	return disk, cloudprovider.WaitStatus(disk, api.DISK_READY, time.Second*5, time.Minute*5)
}

func (storage *SStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.GetDisk(idStr)
	if err != nil {
		return nil, err
	}
	disk.storage = storage
	return disk, nil
}

func (storage *SStorage) GetMountPoint() string {
	return ""
}

func (storage *SStorage) IsSysDiskStore() bool {
	return true
}
