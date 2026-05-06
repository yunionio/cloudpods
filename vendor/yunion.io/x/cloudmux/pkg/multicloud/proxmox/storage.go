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

package proxmox

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStorage struct {
	multicloud.SStorageBase
	ProxmoxTags

	cli *SProxmoxClient

	Storage string `json:"storage"`
	Status  string
	Id      string
	Node    string

	Shared     int    `json:"shared"`
	Content    string `json:"content"`
	MaxDisk    int64  `json:"maxdisk"`
	Disk       int64  `json:"disk"`
	PluginType string `json:"plugintype"`
}

func (self *SStorage) GetName() string {
	if self.Shared == 0 {
		return fmt.Sprintf("%s-%s", self.Node, self.Storage)
	}
	return self.Storage
}

func (self *SStorage) GetId() string {
	if self.Shared == 0 {
		return self.Id
	}
	return self.Storage
}

func (self *SStorage) GetGlobalId() string {
	return self.GetId()
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.cli.GetDisks(self.Node, self.Storage)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SStorage) GetCapacityMB() int64 {
	return int64(self.MaxDisk / 1024 / 1024)
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return int64(self.Disk / 1024 / 1024)
}

func (self *SStorage) GetEnabled() bool {
	if strings.Contains(self.Content, "images") {
		return true
	}
	return false
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disks, err := self.GetIDisks()
	if err != nil {
		return nil, err
	}
	for i := range disks {
		if disks[i].GetGlobalId() == id {
			return disks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return &SStoragecache{
		client: self.cli,
		Node:   self.Node,
	}
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStatus() string {
	if self.Status != "available" {
		return api.STORAGE_OFFLINE
	}
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	ret, err := self.cli.GetStorage(self.GetGlobalId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetStorageType() string {
	return strings.ToLower(self.PluginType)
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SProxmoxClient) GetStorages() ([]SStorage, error) {
	storages := []SStorage{}
	resources, err := self.GetClusterResources("storage")
	if err != nil {
		return nil, err
	}

	err = jsonutils.Update(&storages, resources)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Update")
	}

	storageMap := map[string]bool{}
	ret := []SStorage{}
	for i := range storages {
		storages[i].cli = self
		if storages[i].Shared == 0 {
			ret = append(ret, storages[i])
			continue
		}
		if _, ok := storageMap[storages[i].Storage]; !ok {
			ret = append(ret, storages[i])
			storageMap[storages[i].Storage] = true
		}
	}

	return ret, nil
}

func (self *SProxmoxClient) GetStoragesByHost(node string) ([]SStorage, error) {
	storages, err := self.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []SStorage{}
	for i := range storages {
		if storages[i].Shared == 1 || storages[i].Node == node {
			ret = append(ret, storages[i])
		}
	}
	return ret, nil
}

func (self *SProxmoxClient) GetImages(host, storage, content string) ([]SImage, error) {
	images := []SImage{}
	params := url.Values{}
	params.Add("content", content)
	path := fmt.Sprintf("/nodes/%s/storage/%s/content", host, storage)
	err := self.get(path, params, &images)
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (self *SStorage) GetImages() ([]SImage, error) {
	ret := []SImage{}
	for _, content := range []string{"iso", "import"} {
		if !strings.Contains(self.Content, content) {
			continue
		}
		images, err := self.cli.GetImages(self.Node, self.Storage, content)
		if err != nil {
			return nil, err
		}
		ret = append(ret, images...)
	}
	return ret, nil
}

func (self *SStorage) SearchImage(content, imageId, imageName, format string) (*SImage, error) {
	files, err := self.cli.GetImages(self.Node, self.Storage, content)
	if err != nil {
		return nil, err
	}
	name1 := fmt.Sprintf("%s:%s/%s", self.Storage, content, imageName)
	if !strings.HasSuffix(name1, format) {
		name1 = fmt.Sprintf("%s.%s", name1, format)
	}
	name2 := fmt.Sprintf("%s:%s/%s", self.Storage, content, imageId)
	if !strings.HasSuffix(name2, format) {
		name2 = fmt.Sprintf("%s.%s", name2, format)
	}
	for i := range files {
		if files[i].Volid == name1 || files[i].Volid == name2 {
			return &files[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SProxmoxClient) ImportImage(host, storage, imagId, format string, reader io.Reader) error {
	err := self.upload(host, storage, imagId, format, reader)
	if err != nil {
		return err
	}
	return nil
}

func (self *SProxmoxClient) GetStorage(id string) (*SStorage, error) {
	storages, err := self.GetStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			return &storages[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}
