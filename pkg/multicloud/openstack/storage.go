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

package openstack

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	DEFAULT_STORAGE_TYPE = "scheduler"
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
	return false
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
	if len(storage.ExtraSpecs.VolumeBackendName) == 0 {
		storage.ExtraSpecs.VolumeBackendName = DEFAULT_STORAGE_TYPE
	}
	return storage.ExtraSpecs.VolumeBackendName
}

func (storage *SStorage) GetMediumType() string {
	if strings.Contains(storage.Name, "SSD") {
		return api.DISK_TYPE_SSD
	}
	return api.DISK_TYPE_ROTATE
}

func (storage *SStorage) GetCapacityMB() int64 {
	return 0 // unlimited
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SStorage) GetStatus() string {
	if utils.IsInStringArray(storage.GetStorageType(), storage.zone.getAvailableStorages()) {
		return api.STORAGE_ONLINE
	}
	return api.STORAGE_OFFLINE
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

func (storage *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.CreateDisk("", storage.Name, conf.Name, conf.SizeGb, conf.Desc, conf.ProjectId)
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
