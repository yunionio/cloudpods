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

package cloudpods

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SStoragecache struct {
	multicloud.SResourceBase
	CloudpodsTags
	region *SRegion

	api.StoragecacheDetails
}

func (self *SStoragecache) GetName() string {
	return self.Name
}

func (self *SStoragecache) GetId() string {
	return self.Id
}

func (self *SStoragecache) GetGlobalId() string {
	return self.Id
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
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
	return self.Path
}

func (self *SStoragecache) CreateIImage(snapshotId, name, osType, desc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) UploadImage(ctx context.Context, opts *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	id, err := self.region.UploadImage(ctx, opts, callback)
	if err != nil {
		return "", errors.Wrapf(err, "UploadImage")
	}
	image, err := self.GetIImageById(id)
	if err != nil {
		return "", errors.Wrapf(err, "GetIImageById(%s)", id)
	}
	err = cloudprovider.WaitStatus(image, cloudprovider.IMAGE_STATUS_ACTIVE, time.Second*5, time.Minute*10)
	if err != nil {
		return "", errors.Wrapf(err, "WaitStatus")
	}
	if callback != nil {
		callback(100)
	}
	return id, nil
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	caches, err := self.GetStoragecaches()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStoragecache{}
	for i := range caches {
		caches[i].region = self
		ret = append(ret, &caches[i])
	}
	return ret, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	cache, err := self.GetStoragecache(id)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func (self *SRegion) GetStoragecaches() ([]SStoragecache, error) {
	caches := []SStoragecache{}
	return caches, self.list(&compute.Storagecaches, nil, &caches)
}

func (self *SRegion) GetStoragecache(id string) (*SStoragecache, error) {
	cache := &SStoragecache{region: self}
	return cache, self.cli.get(&compute.Storagecaches, id, nil, cache)
}
