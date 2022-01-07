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

package nutanix

import (
	"context"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type SImage struct {
	multicloud.STagBase
	multicloud.SImageBase

	cache *SStoragecache

	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Deleted              bool   `json:"deleted"`
	StorageContainerID   int    `json:"storage_container_id"`
	StorageContainerUUID string `json:"storage_container_uuid"`
	LogicalTimestamp     int    `json:"logical_timestamp"`
	ImageType            string `json:"image_type"`
	VMDiskID             string `json:"vm_disk_id"`
	ImageState           string `json:"image_state"`
	CreatedTimeInUsecs   int64  `json:"created_time_in_usecs"`
	UpdatedTimeInUsecs   int64  `json:"updated_time_in_usecs"`
	VMDiskSize           int64  `json:"vm_disk_size"`
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) GetId() string {
	return self.UUID
}

func (self *SImage) GetGlobalId() string {
	return self.UUID
}

func (self *SImage) Refresh() error {
	image, err := self.cache.region.GetImage(self.GetGlobalId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetImageFormat() string {
	if self.ImageType == "ISO_IMAGE" {
		return "iso"
	}
	return "raw"
}

func (self *SImage) GetStatus() string {
	switch self.ImageState {
	case "ACTIVE":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "INACTIVE":
		return api.CACHED_IMAGE_STATUS_SAVING
	}
	return self.ImageState
}

func (self *SImage) GetImageStatus() string {
	switch self.ImageState {
	case "ACTIVE":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "INACTIVE":
		return cloudprovider.IMAGE_STATUS_QUEUED
	}
	return cloudprovider.IMAGE_STATUS_KILLED
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	return cloudprovider.ImageTypeSystem
}

func (self *SImage) GetCreatedAt() time.Time {
	return time.Unix(self.CreatedTimeInUsecs/1000, self.CreatedTimeInUsecs%1000)
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.VMDiskSize / 1024 / 1024 / 1024)
}

func (self *SImage) GetSizeByte() int64 {
	return self.VMDiskSize
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(imagetools.NormalizeImageInfo(self.Name, "x86_64", "", "", "").OsType)
}

func (self *SImage) GetOsDist() string {
	return imagetools.NormalizeImageInfo(self.Name, "x86_64", "", "", "").OsDistro
}

func (self *SImage) GetOsVersion() string {
	return imagetools.NormalizeImageInfo(self.Name, "x86_64", "", "", "").OsVersion
}

func (self *SImage) UEFI() bool {
	return false
}

func (self *SImage) GetOsArch() string {
	return "x86_64"
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SRegion) GetImages() ([]SImage, error) {
	images := []SImage{}
	params := url.Values{}
	params.Set("include_vm_disk_sizes", "true")
	params.Set("include_vm_disk_paths", "true")
	return images, self.listAll("images", nil, &images)
}

func (self *SRegion) GetImage(id string) (*SImage, error) {
	image := &SImage{}
	params := url.Values{}
	params.Set("include_vm_disk_sizes", "true")
	params.Set("include_vm_disk_paths", "true")
	return image, self.get("images", id, params, image)
}
