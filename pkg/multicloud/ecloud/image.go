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

package ecloud

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type SImage struct {
	multicloud.SImageBase
	multicloud.EcloudTags
	SZoneRegionBase

	storageCache *SStoragecache
	imgInfo      *imagetools.ImageInfo

	ImageId         string
	ServerId        string
	ImageAlias      string
	Name            string
	Url             string
	SrourceImageId  string
	Status          string
	SizeMb          int `json:"size"`
	IsPublic        int
	CreateTime      time.Time
	Note            string
	OsType          string
	MinDiskGB       int `json:"minDisk"`
	ImageType       string
	PublicImageType string
	BackupType      string
	BackupWay       string
	SnapshotId      string
	OsName          string
}

func (r *SRegion) GetImage(imageId string) (*SImage, error) {
	request := NewNovaRequest(NewApiRequest(r.ID, fmt.Sprintf("/api/v2/image/%s", imageId), nil, nil))
	var image SImage
	err := r.client.doGet(context.Background(), request, &image)
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (r *SRegion) GetImages(isPublic bool) ([]SImage, error) {
	if isPublic {
		return nil, cloudprovider.ErrNotImplemented
	}
	request := NewNovaRequest(NewApiRequest(r.ID, "/api/v2/image", nil, nil))
	images := make([]SImage, 0, 5)
	err := r.client.doList(context.Background(), request, &images)
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (i *SImage) GetCreatedAt() time.Time {
	return i.CreateTime
}

func (i *SImage) GetId() string {
	return i.ImageId
}

func (i *SImage) GetName() string {
	return i.Name
}

func (i *SImage) GetGlobalId() string {
	return i.ImageId
}

func (i *SImage) GetStatus() string {
	switch i.Status {
	case "active":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "queued":
		return api.CACHED_IMAGE_STATUS_INIT
	case "saving":
		return api.CACHED_IMAGE_STATUS_SAVING
	case "caching":
		return api.CACHED_IMAGE_STATUS_CACHING
	case "pending_delete":
		return api.CACHED_IMAGE_STATUS_DELETING
	default:
		return api.CACHED_IMAGE_STATUS_UNKNOWN
	}
}

func (i *SImage) Refresh() error {
	new, err := i.storageCache.region.GetImage(i.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(i, new)
}

func (i *SImage) IsEmulated() bool {
	return false
}

func (i *SImage) GetSysTags() map[string]string {
	data := map[string]string{}

	if osType := i.GetOsType(); len(osType) > 0 {
		data["os_type"] = osType
	}
	if len(i.OsName) > 0 {
		data["os_name"] = i.OsName
	}
	if osDis := i.GetOsDist(); len(osDis) > 0 {
		data["os_distribution"] = osDis
	}
	return data
}

func (i *SImage) GetOsType() string {
	return i.OsType
}

func (i *SImage) GetOsDist() string {
	return i.OsName
}

func (i *SImage) GetOsVersion() string {
	return ""
}

func (i *SImage) GetOsArch() string {
	return ""
}

func (i *SImage) GetMinOsDiskSizeGb() int {
	return i.MinDiskGB
}

func (i *SImage) GetMinRamSizeMb() int {
	return 0
}

func (i *SImage) GetImageFormat() string {
	return ""
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.SizeMb) * 1024 * 1024
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	if self.IsPublic == 1 {
		return cloudprovider.ImageTypeSystem
	}
	return cloudprovider.ImageTypeCustomized
}

func (self *SImage) GetImageStatus() string {
	return cloudprovider.IMAGE_STATUS_ACTIVE
}

func (i *SImage) UEFI() bool {
	return false
}
