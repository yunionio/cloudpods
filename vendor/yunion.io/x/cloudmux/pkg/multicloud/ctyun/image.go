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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SImageBase
	CtyunTags
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

	Architecture     string
	AzName           string
	BootMode         string
	ContainerFormat  string
	CreatedTime      int
	Description      string
	DestinationUser  string
	DiskFormat       string
	DiskId           string
	DiskSize         int
	ImageClass       string
	ImageId          string
	ImageName        string
	ImageType        string
	MaximumRAM       string
	MinimumRAM       string
	OsDistro         string
	OsType           string
	OsVersion        string
	ProjectId        string
	SharedListLength string
	Size             int64
	SourceServerId   string
	SourceUser       string
	Status           string
	Tags             string
	UpdatedTime      string
	Visibility       string
}

func (self *SImage) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SImage) GetId() string {
	return self.ImageId
}

func (self *SImage) GetName() string {
	return self.ImageName
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

func (self *SImage) GetStatus() string {
	switch self.Status {
	case "accepted", "active":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "deactivated", "deactivating", "deleted", "deleting", "pending_delete", "rejected":
		return api.CACHED_IMAGE_STATUS_DELETING
	case "error", "killed":
		return api.CACHED_IMAGE_STATUS_UNKNOWN
	case "importing", "queued", "reactivating", "saving", "syncing", "uploading", "waiting":
		return api.CACHED_IMAGE_STATUS_CACHING
	}
	return api.CACHED_IMAGE_STATUS_UNKNOWN
}

func (self *SImage) Refresh() error {
	image, err := self.storageCache.region.GetImage(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SImage) GetSizeByte() int64 {
	return self.Size
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	switch self.Visibility {
	case "private":
		return cloudprovider.ImageTypeCustomized
	case "public":
		return cloudprovider.ImageTypeSystem
	case "shared":
		return cloudprovider.ImageTypeShared
	case "safe", "community":
		return cloudprovider.ImageTypeMarket
	default:
		return cloudprovider.ImageTypeCustomized
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.Status {
	case "accepted", "active":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "deactivated", "deactivating", "deleted", "deleting", "pending_delete", "rejected":
		return cloudprovider.IMAGE_STATUS_DELETED
	case "error", "killed":
		return cloudprovider.IMAGE_STATUS_KILLED
	case "importing", "queued", "reactivating", "saving", "syncing", "uploading", "waiting":
		return cloudprovider.IMAGE_STATUS_QUEUED
	}
	return cloudprovider.IMAGE_STATUS_KILLED
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(self.ImageName, self.Architecture, self.OsType, self.OsDistro, self.OsVersion)
		self.imgInfo = &imgInfo
	}
	return self.imgInfo
}

func (self *SImage) GetFullOsName() string {
	return self.getNormalizedImageInfo().GetFullOsName()
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
	return self.DiskSize
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetImageFormat() string {
	return self.DiskFormat
}

func (self *SImage) GetCreateTime() time.Time {
	return time.Time{}
}

func (self *SRegion) GetImages(imageType string) ([]SImage, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	switch imageType {
	case "private":
		params["visibility"] = 0
	case "public":
		params["visibility"] = 1
	case "shared":
		params["visibility"] = 2
	case "safe":
		params["visibility"] = 3
	case "community":
		params["visibility"] = 4
	}
	ret := []SImage{}
	for {
		resp, err := self.list(SERVICE_IMAGE, "/v4/image/list", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				Images []SImage
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.ReturnObj.Images...)
		if len(part.ReturnObj.Images) == 0 || len(ret) >= part.TotalCount {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	params := map[string]interface{}{
		"imageID": imageId,
	}
	resp, err := self.list(SERVICE_IMAGE, "/v4/image/detail", params)
	if err != nil {
		return nil, err
	}
	ret := []SImage{}
	err = resp.Unmarshal(&ret, "returnObj", "images")
	if err != nil {
		return nil, err
	}
	for i := range ret {
		if ret[i].ImageId == imageId {
			return &ret[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, imageId)
}
