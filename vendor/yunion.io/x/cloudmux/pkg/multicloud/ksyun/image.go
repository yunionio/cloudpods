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
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/imagetools"
)

/*
   {
     "ImageId": "IMG-8990a317-b0ff-4319-bf41-6dfc1732455c",
     "ContainerFormat": "ovf",
     "Type": "CommonImage",
     "Name": "ubuntu-16.04-gpu-20180102203641",
     "ImageState": "active",
     "CreationDate": "2018-02-27T13:48:32Z",
     "Platform": "ubuntu-16.04",
     "IsPublic": true,
     "IsNpe": true,
     "UserCategory": "common",
     "SysDisk": 20,
     "Progress": "100",
     "ImageSource": "system",
     "CloudInitSupport": false,
     "Ipv6Support": false,
     "IsModifyType": false,
     "FastBoot": false,
     "IsCloudMarket": false,
     "RealImageId": "ab836124-d008-404e-8e78-f3160e3b8df3",
     "OnlineExpansion": true,
     "BootMode": "BIOS(Legacy)",
     "ImageOpenstackDefinedPagePerVq": "false"
   }
*/

type SImage struct {
	multicloud.SImageBase
	SKsyunTags
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

	ImageId                        string
	ContainerFormat                string
	Type                           string
	Name                           string
	ImageState                     string
	CreationDate                   time.Time
	Platform                       string
	IsPublic                       string
	IsNpe                          string
	UserCategory                   string
	SysDisk                        int
	Progress                       string
	ImageSource                    string
	CloudInitSupport               string
	Ipv6Support                    string
	IsModifyType                   string
	FastBoot                       string
	IsCloudMarket                  string
	RealImageId                    string
	OnlineExpansion                string
	BootMode                       string
	ImageOpenstackDefinedPagePerVq string
	Architecture                   string
}

func (image *SImage) GetMinRamSizeMb() int {
	return 0
}

func (image *SImage) GetId() string {
	return image.ImageId
}

func (image *SImage) GetName() string {
	return image.Name
}

func (image *SImage) Delete(ctx context.Context) error {
	return image.storageCache.region.DeleteImage(image.ImageId)
}

func (image *SImage) GetGlobalId() string {
	return image.ImageId
}

func (image *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return image.storageCache
}

func (image *SImage) GetStatus() string {
	switch image.ImageState {
	case "creating":
		return api.CACHED_IMAGE_STATUS_SAVING
	case "active":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (image *SImage) GetImageStatus() string {
	switch image.ImageState {
	case "creating":
		return cloudprovider.IMAGE_STATUS_QUEUED
	case "active":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (image *SImage) Refresh() error {
	ret, err := image.storageCache.region.GetImage(image.ImageId)
	if err != nil {
		return err
	}
	return jsonutils.Update(image, ret)
}

func (image *SImage) GetImageType() cloudprovider.TImageType {
	switch image.Type {
	case "CommonImage":
		return cloudprovider.ImageTypeSystem
	case "CustomImage":
		return cloudprovider.ImageTypeCustomized
	case "MarketImage":
		return cloudprovider.ImageTypeMarket
	default:
		return cloudprovider.ImageTypeCustomized
	}
}

func (image *SImage) GetSizeByte() int64 {
	return int64(image.SysDisk) * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.getNormalizedImageInfo().OsType)
}

func (self *SImage) GetOsDist() string {
	return self.getNormalizedImageInfo().OsDistro
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(self.Platform, self.Architecture, "", "", "")
		self.imgInfo = &imgInfo
	}

	return self.imgInfo
}

func (self *SImage) GetFullOsName() string {
	return self.Name
}

func (self *SImage) GetOsVersion() string {
	return self.getNormalizedImageInfo().OsVersion
}

func (self *SImage) GetOsLang() string {
	return self.getNormalizedImageInfo().OsLang
}

func (self *SImage) GetOsArch() string {
	return self.getNormalizedImageInfo().OsArch
}

func (self *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.getNormalizedImageInfo().OsBios)
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return self.SysDisk
}

func (self *SImage) GetImageFormat() string {
	return self.ContainerFormat
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.CreationDate
}

func (region *SRegion) GetImage(imageId string) (*SImage, error) {
	images, err := region.GetImages(imageId, "")
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].ImageId == imageId {
			return &images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetImages(id string, imageType string) ([]SImage, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["ImageId"] = id
	}
	if len(imageType) > 0 {
		params["ImageType"] = imageType
	}
	resp, err := region.ecsRequest("DescribeImages", params)
	if err != nil {
		return nil, err
	}
	ret := struct {
		ImagesSet []SImage
		NextToken string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret.ImagesSet, nil
}

func (region *SRegion) DeleteImage(imageId string) error {
	params := map[string]string{
		"ImageId": imageId,
	}
	_, err := region.ecsRequest("DeleteImage", params)
	return err
}
