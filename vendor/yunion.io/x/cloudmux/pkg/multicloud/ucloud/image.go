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

package ucloud

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SImageBase
	UcloudTags
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

	Zone             string `json:"Zone"`
	ImageDescription string `json:"ImageDescription"`
	OSName           string `json:"OsName"`
	ImageID          string `json:"ImageId"`
	State            string `json:"State"`
	ImageName        string `json:"ImageName"`
	OSType           string `json:"OsType"`
	CreateTime       int64  `json:"CreateTime"`
	ImageType        string `json:"ImageType"`
	ImageSizeGB      int64  `json:"ImageSize"`
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return self.ImageID
}

func (self *SImage) GetName() string {
	if len(self.ImageName) == 0 {
		return self.GetId()
	}

	return self.ImageName
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

// 镜像状态， 可用：Available，制作中：Making， 不可用：Unavailable
func (self *SImage) GetStatus() string {
	switch self.State {
	case "Available":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "Making":
		return api.CACHED_IMAGE_STATUS_CACHING
	case "Unavailable":
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) Refresh() error {
	new, err := self.storageCache.region.GetImage(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.GetId())
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SImage) GetSizeByte() int64 {
	return self.ImageSizeGB * 1024 * 1024 * 1024
}

// 镜像类型。标准镜像：Base，镜像市场：Business， 自定义镜像：Custom，默认返回所有类型
func (self *SImage) GetImageType() cloudprovider.TImageType {
	switch self.ImageType {
	case "Base":
		return cloudprovider.ImageTypeSystem
	case "Custom":
		return cloudprovider.ImageTypeCustomized
	case "Business":
		return cloudprovider.ImageTypeShared
	default:
		return cloudprovider.ImageTypeCustomized
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.State {
	case "Available":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "Making":
		return cloudprovider.IMAGE_STATUS_QUEUED
	case "Unavailable":
		return cloudprovider.IMAGE_STATUS_KILLED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(self.ImageName, "", "", "", "")
		self.imgInfo = &imgInfo
	}

	return self.imgInfo
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.getNormalizedImageInfo().OsType)
}

func (self *SImage) GetOsDist() string {
	return self.getNormalizedImageInfo().OsDistro
}

func (self *SImage) GetOsVersion() string {
	return self.getNormalizedImageInfo().OsVersion
}

func (self *SImage) GetOsArch() string {
	return self.getNormalizedImageInfo().OsArch
}

func (img *SImage) GetOsLang() string {
	return img.getNormalizedImageInfo().OsLang
}

func (img *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(img.getNormalizedImageInfo().OsBios)
}

func (img *SImage) GetFullOsName() string {
	return img.ImageName
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.ImageSizeGB)
}

func (self *SImage) GetImageFormat() string {
	return ""
}

func (self *SImage) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime, 0)
}

// https://docs.ucloud.cn/api/uhost-api/describe_image
func (self *SRegion) GetImage(imageId string) (SImage, error) {
	params := NewUcloudParams()
	params.Set("ImageId", imageId)

	images := make([]SImage, 0)
	err := self.DoListAll("DescribeImage", params, &images)
	if err != nil {
		return SImage{}, err
	}

	if len(images) == 1 {
		return images[0], nil
	} else if len(images) == 0 {
		return SImage{}, cloudprovider.ErrNotFound
	} else {
		return SImage{}, fmt.Errorf("GetImage %s %d found.", imageId, len(images))
	}
}

// https://docs.ucloud.cn/api/uhost-api/describe_image
// ImageType 标准镜像：Base，镜像市场：Business， 自定义镜像：Custom，默认返回所有类型
func (self *SRegion) GetImages(imageType string, imageId string) ([]SImage, error) {
	params := NewUcloudParams()

	if len(imageId) > 0 {
		params.Set("ImageId", imageId)
	}

	if len(imageType) > 0 {
		params.Set("ImageType", imageType)
	}

	images := make([]SImage, 0)
	err := self.DoListAll("DescribeImage", params, &images)
	return images, err
}

// https://docs.ucloud.cn/api/uhost-api/terminate_custom_image
func (self *SRegion) DeleteImage(imageId string) error {
	params := NewUcloudParams()
	params.Set("ImageId", imageId)

	return self.DoAction("TerminateCustomImage", params, nil)
}
