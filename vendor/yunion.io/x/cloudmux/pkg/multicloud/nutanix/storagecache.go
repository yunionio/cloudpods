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

package nutanix

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	multicloud.STagBase

	storage *SStorage
	region  *SRegion
}

func (self *SStoragecache) GetName() string {
	return self.storage.GetName()
}

func (self *SStoragecache) GetId() string {
	return self.storage.GetId()
}

func (self *SStoragecache) GetGlobalId() string {
	return self.storage.GetGlobalId()
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := self.region.GetImages()
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		if images[i].StorageContainerUUID != self.storage.GetGlobalId() {
			continue
		}
		images[i].cache = self
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) GetIImageById(id string) (cloudprovider.ICloudImage, error) {
	image, err := self.region.GetImage(id)
	if err != nil {
		return nil, err
	}
	image.cache = self
	return image, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) UploadImage(ctx context.Context, opts *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	reader, size, err := opts.GetReader(opts.ImageId, string(qemuimgfmt.QCOW2))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	image, err := self.region.CreateImage(self.storage.StorageContainerUUID, opts, size, reader, callback)
	if err != nil {
		return "", err
	}
	if callback != nil {
		callback(100.0)
	}
	image.cache = self
	return image.GetGlobalId(), nil
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storages, err := self.GetStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStoragecache{}
	for i := range storages {
		cache := &SStoragecache{storage: &storages[i], region: self}
		ret = append(ret, cache)
	}
	return ret, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storage, err := self.GetStorage(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage")
	}
	return &SStoragecache{region: self, storage: storage}, nil
}
