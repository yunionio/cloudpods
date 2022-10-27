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
	"context"
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	InCloudSphereTags

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
	iss, err := self.zone.region.GetImageStorages(self.zone.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImageStorages")
	}
	for i := range iss {
		images, err := self.zone.region.GetImageList(iss[i].Id)
		if err != nil {
			return nil, err
		}
		for j := range images {
			if utils.IsInStringArray(images[j].GetImageFormat(), []string{"iso", "ova"}) {
				continue
			}
			images[j].cache = self
			ret = append(ret, &images[j])
		}
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

type SImageStorage struct {
	Id   string
	Name string
}

func (self *SRegion) GetImageStorages(dsId string) ([]SImageStorage, error) {
	params := url.Values{}
	params.Set("type", "imagestorage")
	res := fmt.Sprintf("/datacenters/%s/storages", dsId)
	ret := []SImageStorage{}
	return ret, self.get(res, params, &ret)
}

func (self *SRegion) GetImageList(storageId string) ([]SImage, error) {
	res := fmt.Sprintf("/storages/%s/files", storageId)
	ret := []SImage{}
	return ret, self.list(res, url.Values{}, &ret)
}
