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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	image_api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SStoragecache struct {
	multicloud.SResourceBase
	multicloud.CloudpodsTags
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

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, opts *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(opts.ExternalId) > 0 {
		image, err := self.region.GetImage(opts.ExternalId)
		if err != nil {
			if e, ok := errors.Cause(err).(*httputils.JSONClientError); ok && e.Code != 404 {
				return "", errors.Wrapf(err, "GetImage(%s)", opts.ExternalId)
			}
			log.Errorf("GetImage %s error: %v", opts.ExternalId, err)
		}
		if image == nil || image.Status != image_api.IMAGE_STATUS_ACTIVE || isForce {
			self.region.cli.delete(&modules.Images, opts.ExternalId)
		} else {
			return opts.ExternalId, nil
		}
	}
	return self.region.UploadImage(ctx, userCred, opts)
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
	return caches, self.list(&modules.Storagecaches, nil, &caches)
}

func (self *SRegion) GetStoragecache(id string) (*SStoragecache, error) {
	cache := &SStoragecache{region: self}
	return cache, self.cli.get(&modules.Storagecaches, id, nil, cache)
}
