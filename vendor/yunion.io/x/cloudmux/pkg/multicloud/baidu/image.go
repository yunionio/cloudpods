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

package baidu

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/imagetools"
)

type SImage struct {
	multicloud.SImageBase
	SBaiduTag

	cache   *SStoragecache
	imgInfo *imagetools.ImageInfo

	Id         string
	CreateTime time.Time
	Name       string
	Type       string
	OsType     string
	OsVersion  string
	OsLang     string
	OsName     string
	OsBuild    string
	OsArch     string
	Status     string
	Desc       string
	MinDiskGb  int64
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return self.Id
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.cache.region.DeleteImage(self.Id)
}

func (self *SImage) GetGlobalId() string {
	return self.Id
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetStatus() string {
	switch self.Status {
	case "Creating":
		return api.CACHED_IMAGE_STATUS_SAVING
	case "Available":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "CreatedFailed", "NotAvailable", "Error":
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return strings.ToLower(self.Status)
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.Status {
	case "Creating":
		return cloudprovider.IMAGE_STATUS_QUEUED
	case "Available":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "CreatedFailed", "NotAvailable", "Error":
		return cloudprovider.IMAGE_STATUS_DELETED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	image, err := self.cache.region.GetImage(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	switch self.Type {
	case "System":
		return cloudprovider.ImageTypeSystem
	case "Custom":
		return cloudprovider.ImageTypeCustomized
	case "Integration":
		return cloudprovider.ImageTypeMarket
	default:
		return cloudprovider.ImageTypeCustomized
	}
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.GetMinOsDiskSizeGb()) * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.getNormalizedImageInfo().OsType)
}

func (self *SImage) GetOsDist() string {
	return self.getNormalizedImageInfo().OsDistro
}

func getOsArch(osArch string) string {
	for _, key := range []string{"x86_64", "i386", "amd64", "aarch64"} {
		if strings.Contains(osArch, key) {
			return key
		}
	}
	return osArch
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(self.OsName, getOsArch(self.OsArch), self.OsType, self.OsName, self.OsVersion)
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
	if self.MinDiskGb > 0 {
		return int(self.MinDiskGb)
	}
	image, err := self.cache.region.GetImage(self.Id)
	if err != nil {
		if strings.EqualFold(self.OsType, "windows") {
			return 40
		}
		return 20
	}
	return int(image.MinDiskGb)
}

func (self *SImage) GetImageFormat() string {
	return "raw"
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (region *SRegion) GetImages(imageType string) ([]SImage, error) {
	params := url.Values{}
	if len(imageType) > 0 {
		params.Set("imageType", imageType)
	}
	ret := []SImage{}
	for {
		resp, err := region.bccList("v2/image", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextMarker string
			Images     []SImage
		}{}
		err = resp.Unmarshal(&part)
		ret = append(ret, part.Images...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) GetImage(id string) (*SImage, error) {
	resp, err := region.bccList(fmt.Sprintf("v2/image/%s", id), nil)
	if err != nil {
		return nil, err
	}
	ret := &SImage{}
	err = resp.Unmarshal(ret, "image")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) DeleteImage(id string) error {
	_, err := region.bccDelete(fmt.Sprintf("v2/image/%s", id), nil)
	return err
}
