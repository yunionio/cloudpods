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
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	multicloud.ProxmoxTags

	zone *SZone
}

func (self *SStoragecache) GetGlobalId() string {
	return self.zone.GetGlobalId()
}

func (self *SStoragecache) GetId() string {
	return self.zone.GetId()
}

func (self *SStoragecache) GetName() string {
	return self.zone.GetName()
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	ret := []cloudprovider.ICloudImage{}
	images, err := self.zone.region.GetImageList()
	if err != nil {
		return nil, err
	}
	for i := range images {
		images[i].cache = self
		ret = append(ret, &images[i])
	}

	return ret, nil
}

func (seflf *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) GetIImageById(id string) (cloudprovider.ICloudImage, error) {
	images, err := self.GetICloudImages()
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].GetGlobalId() == id {
			return images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) CreateIImage(snpId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	return "", cloudprovider.ErrNotSupported
}
