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

package baidu

import (
	"fmt"
	"net/url"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SStorage struct {
	multicloud.SStorageBase

	zone        *SZone
	StorageType string
	MinDiskSize int
	MaxDiskSize int
}

var storageMap = map[string]string{
	"enhanced_ssd_pl1": "enhanced_ssd_pl1",
	"enhanced_ssd_pl2": "enhanced_ssd_pl2",
	"enhanced_ssd_pl3": "enhanced_ssd_pl3",
	"cloud_hp1":        "premium_ssd",
	"hp1":              "ssd",
}

func isMatchStorageType(v1, v2 string) bool {
	e1, _ := storageMap[v1]
	e2, _ := storageMap[v2]
	return v1 == v2 || v1 == e2 || e1 == v2 || e1 == e2
}

func (storage *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.region.client.cpcfg.Id, storage.zone.GetId(), storage.StorageType)
}

func (storage *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.region.client.cpcfg.Name, storage.zone.GetId(), storage.StorageType)
}

func (storage *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.region.client.cpcfg.Id, storage.zone.GetGlobalId(), storage.StorageType)
}

func (storage *SStorage) IsEmulated() bool {
	return true
}

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks(storage.StorageType, storage.zone.ZoneName, "")
	if err != nil {
		return nil, errors.Wrap(err, "region.GetDisks")
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].storage = storage
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (storage *SStorage) GetStorageType() string {
	return storage.StorageType
}

func (storage *SStorage) GetMediumType() string {
	if strings.Contains(strings.ToLower(storage.StorageType), "ssd") {
		return api.DISK_TYPE_SSD
	}
	return api.DISK_TYPE_ROTATE
}

func (storage *SStorage) GetCapacityMB() int64 {
	return 0
}

func (storage *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (storage *SStorage) GetEnabled() bool {
	return true
}

func (storage *SStorage) CreateIDisk(opts *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.CreateDisk(storage.StorageType, storage.zone.ZoneName, opts)
	if err != nil {
		return nil, err
	}
	disk.storage = storage
	return disk, nil
}

func (storage *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := storage.zone.region.GetDisk(id)
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

func (storage *SStorage) DisableSync() bool {
	return false
}

func (storage *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return storage.zone.region.getStoragecache()
}

func (storage *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (region *SRegion) GetStorageTypes(zoneName string) ([]SStorage, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	if len(zoneName) > 0 {
		params.Set("zoneName", zoneName)
	}
	resp, err := region.bccList("v2/volume/disk", params)
	if err != nil {
		return nil, err
	}
	ret := struct {
		DiskZoneResources []struct {
			ZoneName  string
			DiskInfos []SStorage
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	storages := []SStorage{}
	for _, diskInfos := range ret.DiskZoneResources {
		storages = append(storages, diskInfos.DiskInfos...)
	}
	return storages, nil
}
