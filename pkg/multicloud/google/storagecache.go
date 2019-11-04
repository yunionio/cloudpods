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

package google

import (
	"context"
	"fmt"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (cache *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (cache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerId, cache.region.GetId())
}

func (cache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerName, cache.region.GetId())
}

func (cache *SStoragecache) GetStatus() string {
	return "available"
}

func (cache *SStoragecache) Refresh() error {
	return nil
}

func (cache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerId, cache.region.GetGlobalId())
}

func (cache *SStoragecache) IsEmulated() bool {
	return true
}

func (cache *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	images, err := cache.region.fetchImages()
	if err != nil {
		return nil, err
	}
	iimages := []cloudprovider.ICloudImage{}
	for i := range images {
		images[i].storagecache = cache
		iimages = append(iimages, &images[i])
	}
	return iimages, nil
}

func (cache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := cache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (cache *SStoragecache) GetPath() string {
	return ""
}

func (cache *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SRegion) CheckBucket(bucketName string) (*oss.Bucket, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SRegion) CreateImage(snapshoutId, imageName, imageDesc string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	cache := &SStoragecache{region: region}
	return []cloudprovider.ICloudStoragecache{cache}, nil
}

func (region *SRegion) getStoragecache() cloudprovider.ICloudStoragecache {
	return &SStoragecache{region: region}
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	cache := region.getStoragecache()
	if id == cache.GetGlobalId() {
		return cache, nil
	}
	return nil, cloudprovider.ErrNotFound
}
