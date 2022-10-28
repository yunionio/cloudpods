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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	DEFAULT_STORAGE_TYPE = "scheduler"
)

type SExtraSpecs struct {
	VolumeBackendName string
}

type SStorage struct {
	multicloud.SStorageBase
	zone       *SZone
	Name       string
	ExtraSpecs SExtraSpecs
	ID         string
}

func (storage *SStorage) GetId() string {
	return storage.ID
}

func (storage *SStorage) GetName() string {
	return storage.Name
}

func (storage *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", storage.zone.GetGlobalId(), storage.ID)
}

func (storage *SStorage) IsEmulated() bool {
	return false
}

func (storage *SStorage) GetIZone() cloudprovider.ICloudZone {
	return storage.zone
}

func (storage *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := storage.zone.region.GetDisks()
	if err != nil {
		return nil, err
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		if disks[i].AvailabilityZone == storage.zone.ZoneName && (disks[i].VolumeType == storage.Name || strings.HasSuffix(disks[i].Host, "#"+storage.ExtraSpecs.VolumeBackendName)) {
			disks[i].storage = storage
			idisks = append(idisks, &disks[i])
		}
	}
	return idisks, nil
}

func (storage *SStorage) GetStorageType() string {
	if len(storage.ExtraSpecs.VolumeBackendName) == 0 {
		return DEFAULT_STORAGE_TYPE
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

func (storage *SStorage) GetCapacityUsedMB() int64 {
	return 0
}

func (storage *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (storage *SStorage) GetStatus() string {
	ok, err := storage.zone.region.IsStorageAvailable(storage.GetStorageType())
	if err != nil || !ok {
		return api.STORAGE_OFFLINE
	}
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
	if disk.AvailabilityZone != storage.zone.ZoneName {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "disk %s not in zone %s", disk.Name, storage.zone.ZoneName)
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

func (region *SRegion) GetStorageTypes() ([]SStorage, error) {
	resource := "/types"
	storages := []SStorage{}
	query := url.Values{}
	for {
		resp, err := region.bsList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "bsReqest")
		}
		part := struct {
			VolumeTypes     []SStorage
			VolumeTypeLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		storages = append(storages, part.VolumeTypes...)
		marker := part.VolumeTypeLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return storages, nil
}

type SCinderService struct {
	ActiveBackendId string
	// cinder-volume
	Binary            string
	DisabledReason    string
	Frozen            string
	Host              string
	ReplicationStatus string
	State             string
	Status            string
	UpdatedAt         time.Time
	Zone              string
}

func (region *SRegion) GetCinderServices() ([]SCinderService, error) {
	resp, err := region.bsList("/os-services", nil)
	if err != nil {
		return nil, errors.Wrap(err, "bsList")
	}
	services := []SCinderService{}
	err = resp.Unmarshal(&services, "services")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return services, nil
}

func (region *SRegion) IsStorageAvailable(storageType string) (bool, error) {
	if utils.IsInStringArray(storageType, []string{DEFAULT_STORAGE_TYPE, api.STORAGE_OPENSTACK_NOVA}) {
		return true, nil
	}
	services, err := region.GetCinderServices()
	if err != nil {
		return false, errors.Wrap(err, "GetCinderServices")
	}
	for _, service := range services {
		if service.Binary == "cinder-volume" && strings.Contains(service.Host, "@") {
			hostInfo := strings.Split(service.Host, "@")
			if hostInfo[len(hostInfo)-1] == storageType {
				if service.State == "up" && service.Status == "enabled" {
					return true, nil
				}
			}
		}
	}
	log.Errorf("storage %s offline", storageType)
	return false, nil
}

type SCapabilities struct {
}

type SPool struct {
	Name         string
	Capabilities SCapabilities
}

func (region *SRegion) GetSchedulerStatsPool() ([]SPool, error) {
	resp, err := region.bsList("/scheduler-stats/get_pools", nil)
	if err != nil {
		return nil, errors.Wrap(err, "bsList")
	}
	pools := []SPool{}
	err = resp.Unmarshal(&pools, "pools")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return pools, nil
}
