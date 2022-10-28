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
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNovaStorage struct {
	multicloud.SStorageBase
	OpenStackTags
	host *SHypervisor
	zone *SZone
}

func (storage *SNovaStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", storage.zone.GetGlobalId(), storage.host.GetId(), storage.GetName())
}

func (storage *SNovaStorage) GetName() string {
	return api.STORAGE_OPENSTACK_NOVA
}

func (storage *SNovaStorage) GetGlobalId() string {
	return storage.GetId()
}

func (storage *SNovaStorage) IsEmulated() bool {
	return true
}

func (storage *SNovaStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SNovaStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return []cloudprovider.ICloudDisk{}, nil
}

func (storage *SNovaStorage) GetStorageType() string {
	return api.STORAGE_OPENSTACK_NOVA
}

func (storage *SNovaStorage) GetMediumType() string {
	return api.DISK_TYPE_ROTATE
}

func (storage *SNovaStorage) GetCapacityMB() int64 {
	return int64(storage.host.GetStorageSizeMB())
}

func (storage *SNovaStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (storage *SNovaStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SNovaStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (storage *SNovaStorage) Refresh() error {
	// do nothing
	return nil
}

func (storage *SNovaStorage) GetEnabled() bool {
	return true
}

func (storage *SNovaStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return storage.zone.region.getStoragecache()
}

func (storage *SNovaStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (storage *SNovaStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	return &SNovaDisk{region: storage.zone.region, storage: storage, instanceId: idStr}, nil
}

func (storage *SNovaStorage) GetMountPoint() string {
	return ""
}

func (storage *SNovaStorage) IsSysDiskStore() bool {
	return true
}

func (self *SNovaStorage) DisableSync() bool {
	return true
}
