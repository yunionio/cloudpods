// Copyright 2023 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volcengine

import (
	"fmt"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SStorage struct {
	multicloud.SStorageBase
	VolcEngineTags
	zone        *SZone
	storageType string
}

func (storage *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.region.client.cpcfg.Id, storage.zone.GetId(), storage.storageType)
}

func (storage *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.region.client.cpcfg.Name, storage.zone.GetId(), storage.storageType)
}

func (storage *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.region.client.cpcfg.Id, storage.zone.GetGlobalId(), storage.storageType)
}

func (storage *SStorage) IsEmulated() bool {
	return true
}

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks("", storage.zone.GetId(), storage.storageType, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisks")
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i += 1 {
		disks[i].storage = storage
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (storage *SStorage) GetStorageType() string {
	return storage.storageType
}

func (storage *SStorage) GetCapacityMB() int64 {
	return 0
}

func (storage *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (storage *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SStorage) GetStatus() string {
	if storage.storageType == api.STORAGE_VOLCENGINE_PTSSD {
		return api.STORAGE_OFFLINE
	}
	return api.STORAGE_ONLINE
}

func (storage *SStorage) GetEnabled() bool {
	return true
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return storage.zone.region.getStoragecache()
}

func (storage *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	diskId, err := storage.zone.region.CreateDisk(storage.zone.ZoneId, storage.storageType, conf.Name, conf.SizeGb, conf.Desc, conf.ProjectId)
	if err != nil {
		log.Errorf("createDisk fail %s", err)
		return nil, err
	}
	err = cloudprovider.Wait(5*time.Second, time.Minute, func() (bool, error) {
		_, err := storage.zone.region.GetDisk(diskId)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		}
		return true, err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find disk after create")
	}
	disk, err := storage.zone.region.GetDisk(diskId)
	if err != nil {
		return nil, err
	}
	disk.storage = storage
	return disk, nil
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
