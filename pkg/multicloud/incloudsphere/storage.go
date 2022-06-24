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

package incloudsphere

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type DataCenterOrHostDto struct {
	DataCenterOrHost string `json:"dataCenterOrHost"`
	DataCenterName   string `json:"dataCenterName"`
	HostName         string `json:"hostName"`
	Status           string `json:"status"`
}

type BlockDeviceDto struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PartitionFlag   bool   `json:"partitionFlag"`
	PartitionNumber string `json:"partitionNumber"`
	Capacity        int    `json:"capacity"`
	DisplayCapacity string `json:"displayCapacity"`
	DiskType        string `json:"diskType"`
	AbsolutePath    string `json:"absolutePath"`
	Transport       string `json:"transport"`
	Used            string `json:"used"`
	IsUsed          bool   `json:"isUsed"`
}

type SStorage struct {
	multicloud.SStorageBase
	multicloud.InCloudSphereTags

	zone *SZone

	Id                  string              `json:"id"`
	Name                string              `json:"name"`
	MountPath           string              `json:"mountPath"`
	DataStoreType       string              `json:"dataStoreType"`
	Capacity            float64             `json:"capacity"`
	UsedCapacity        float64             `json:"usedCapacity"`
	AvailCapacity       float64             `json:"availCapacity"`
	DataCenterId        string              `json:"dataCenterId"`
	HostId              string              `json:"hostId"`
	MountStatus         string              `json:"mountStatus"`
	HostIP              string              `json:"hostIp"`
	UUID                string              `json:"uuid"`
	AbsolutePath        string              `json:"absolutePath"`
	DataCenterName      string              `json:"dataCenterName"`
	DataCenterOrHostDto DataCenterOrHostDto `json:"dataCenterOrHostDto"`
	BlockDeviceDto      BlockDeviceDto      `json:"blockDeviceDto"`
	XactiveStoreName    string              `json:"xactiveStoreName"`
	XactiveStoreId      string              `json:"xactiveStoreId"`
	DataCenterDto       string              `json:"dataCenterDto"`
	HostNumbers         int                 `json:"hostNumbers"`
	VMNumbers           int                 `json:"vmNumbers"`
	VolumesNumbers      int                 `json:"volumesNumbers"`
	VMTemplateNumbers   int                 `json:"vmTemplateNumbers"`
	Tags                string              `json:"tags"`
	MaxSlots            int                 `json:"maxSlots"`
	Creating            bool                `json:"creating"`
	StorageBackUp       bool                `json:"storageBackUp"`
	ExtensionType       string              `json:"extensionType"`
	CanBeImageStorage   bool                `json:"canBeImageStorage"`
	MultiplexRatio      float64             `json:"multiplexRatio"`
	Oplimit             bool                `json:"oplimit"`
	Maxop               int                 `json:"maxop"`
	MountStateCount     string              `json:"mountStateCount"`
	DatastoreRole       string              `json:"datastoreRole"`
	CanCreateXactive    bool                `json:"canCreateXactive"`
	BlockDeviceUUId     string              `json:"blockDeviceUuid"`
	OpHostIP            string              `json:"opHostIp"`
	IsMount             string              `json:"isMount"`
	HostDto             string              `json:"hostDto"`
	ScvmOn              bool                `json:"scvmOn"`
}

func (self *SStorage) GetName() string {
	return self.Name
}

func (self *SStorage) GetId() string {
	return self.Id
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].region = self.zone.region
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) GetCapacityMB() int64 {
	return int64(self.Capacity) * 1024
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return int64(self.UsedCapacity) * 1024
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	if disk.DataStoreId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	return disk, nil
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	//TODO
	return nil
	//return &SStoragecache{storage: self, region: self.zone.region}
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	ret, err := self.zone.region.GetStorage(self.GetGlobalId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetStorageType() string {
	return strings.ToLower(self.DataStoreType)
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.GetStorage(id)
}

func (self *SRegion) GetStoragesByDc(dcId string) ([]SStorage, error) {
	storages := []SStorage{}
	res := fmt.Sprintf("/datacenters/%s/storages", dcId)
	return storages, self.list(res, url.Values{}, &storages)
}

func (self *SRegion) GetStoragesByHost(hostId string) ([]SStorage, error) {
	storages := []SStorage{}
	res := fmt.Sprintf("/hosts/%s/storages", hostId)
	return storages, self.list(res, url.Values{}, &storages)
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	ret := &SStorage{}
	res := fmt.Sprintf("/storages/%s", id)
	return ret, self.get(res, url.Values{}, ret)
}
