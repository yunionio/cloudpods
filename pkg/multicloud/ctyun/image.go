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

package ctyun

import (
	"context"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

const (
	ImageOwnerPublic string = "gold"    // 公共镜像：gold
	ImageOwnerSelf   string = "private" // 私有镜像：private
	ImageOwnerShared string = "shared"  // 共享镜像：shared
)

// http://ctyun-api-url/apiproxy/v3/order/getImages
type SImage struct {
	storageCache *SStoragecache

	ID        string `json:"id"`
	OSType    string `json:"osType"`
	Platform  string `json:"platform"`
	Name      string `json:"name"`
	OSBit     int64  `json:"osBit"`
	MinRAM    int64  `json:"minRam"`
	MinDisk   int64  `json:"minDisk"`
	ImageType string `json:"imageType"`
	Virtual   bool   `json:"virtual"`
}

func (self *SImage) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SImage) GetId() string {
	return self.ID
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

func (self *SImage) GetStatus() string {
	return api.CACHED_IMAGE_STATUS_READY
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

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	if self.OSBit > 0 {
		data.Add(jsonutils.NewString(self.GetOsArch()), "os_arch")
	}
	if len(self.OSType) > 0 {
		data.Add(jsonutils.NewString(self.GetOsType()), "os_name")
	}
	if len(self.Platform) > 0 {
		data.Add(jsonutils.NewString(self.GetOsDist()), "os_distribution")
	}

	data.Add(jsonutils.NewString(self.GetOsVersion()), "os_version")

	return data
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SImage) GetSizeByte() int64 {
	return self.MinDisk * 1024 * 1024 * 1024
}

func (self *SImage) GetImageType() string {
	switch self.ImageType {
	case "gold":
		return cloudprovider.CachedImageTypeSystem
	case "private":
		return cloudprovider.CachedImageTypeCustomized
	case "shared":
		return cloudprovider.CachedImageTypeShared
	default:
		return cloudprovider.CachedImageTypeCustomized
	}
}

func (self *SImage) GetImageStatus() string {
	return cloudprovider.IMAGE_STATUS_ACTIVE
}

func (self *SImage) GetOsType() string {
	return self.OSType
}

func (self *SImage) GetOsDist() string {
	return self.Platform
}

func (self *SImage) GetOsVersion() string {
	return imagetools.NormalizeImageInfo(self.Name, "", "", "", "").OsVersion
}

func (self *SImage) GetOsArch() string {
	return strconv.Itoa(int(self.OSBit))
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.MinDisk)
}

func (self *SImage) GetMinRamSizeMb() int {
	return int(self.MinRAM)
}

func (self *SImage) GetImageFormat() string {
	return ""
}

func (self *SImage) GetCreateTime() time.Time {
	return time.Time{}
}

func (self *SRegion) GetImages(imageType string) ([]SImage, error) {
	params := map[string]string{
		"regionId":  self.GetId(),
		"imageType": imageType,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/order/getImages", params)
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetImages.DoGet")
	}

	images := make([]SImage, 0)
	err = resp.Unmarshal(&images, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "Region.GetImages.Unmarshal")
	}

	for i := range images {
		images[i].storageCache = &SStoragecache{
			region: self,
		}
	}

	return images, nil
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	images, err := self.GetImages("")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetImage.GetImages")
	}

	for i := range images {
		if images[i].GetId() == imageId {
			images[i].storageCache = &SStoragecache{
				region: self,
			}

			return &images[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetImage")
}
