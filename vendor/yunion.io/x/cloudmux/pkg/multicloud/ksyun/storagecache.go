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

package ksyun

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	SKsyunTags
	region *SRegion
}

func (cache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.cpcfg.Id, cache.region.GetId())
}

func (cache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", cache.region.client.cpcfg.Name, cache.region.GetId())
}

func (cache *SStoragecache) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (cache *SStoragecache) Refresh() error {
	return nil
}

func (cache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.cpcfg.Id, cache.region.GetGlobalId())
}

func (cache *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := cache.region.GetImages("", "custom")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		images[i].storageCache = cache
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (cache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	img, err := cache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	img.storageCache = cache
	return img, nil
}

func (cache *SStoragecache) GetPath() string {
	return ""
}

func (cache *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (region *SRegion) getStoragecache() *SStoragecache {
	return &SStoragecache{region: region}
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	cache := region.getStoragecache()
	return []cloudprovider.ICloudStoragecache{cache}, nil
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	if id == storageCache.GetGlobalId() {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}
