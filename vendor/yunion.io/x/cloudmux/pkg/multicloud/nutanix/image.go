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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.STagBase
	multicloud.SImageBase

	cache *SStoragecache

	imageInfo *imagetools.ImageInfo

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

func (img *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if img.imageInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(img.Name, "", "", "", "")
		img.imageInfo = &imgInfo
	}
	return img.imageInfo
}

func (img *SImage) GetFullOsName() string {
	return img.Name
}

func (ins *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(ins.getNormalizedImageInfo().OsType)
}

func (ins *SImage) GetOsDist() string {
	return ins.getNormalizedImageInfo().OsDistro
}

func (ins *SImage) GetOsVersion() string {
	return ins.getNormalizedImageInfo().OsVersion
}

func (ins *SImage) GetOsLang() string {
	return ins.getNormalizedImageInfo().OsLang
}

func (ins *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(ins.getNormalizedImageInfo().OsBios)
}

func (ins *SImage) GetOsArch() string {
	return ins.getNormalizedImageInfo().OsArch
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

func (self *SRegion) CreateImage(storageId string, opts *cloudprovider.SImageCreateOption, sizeBytes int64, body io.Reader, callback func(float32)) (*SImage, error) {
	params := map[string]interface{}{
		"image_type": "DISK_IMAGE",
		"name":       opts.ImageName,
		"annotation": opts.OsDistribution,
	}
	ret := struct {
		TaskUUID string
	}{}
	err := self.post("images", jsonutils.Marshal(params), &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "create image")
	}
	imageId := ""
	err = cloudprovider.Wait(time.Second*5, time.Minute*3, func() (bool, error) {
		task, err := self.GetTask(ret.TaskUUID)
		if err != nil {
			return false, err
		}
		for _, entity := range task.EntityList {
			imageId = entity.EntityID
		}
		log.Debugf("task %s %s status: %s", task.OperationType, task.UUID, task.ProgressStatus)
		if task.ProgressStatus == "Succeeded" {
			for _, entity := range task.EntityList {
				imageId = entity.EntityID
			}
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	header.Set("X-Nutanix-Destination-Container", storageId)
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Length", fmt.Sprintf("%d", sizeBytes))
	reader := multicloud.NewProgress(sizeBytes, 90, body, callback)
	resp, err := self.upload("images", fmt.Sprintf("%s/upload", imageId), header, reader)
	if err != nil {
		return nil, errors.Wrapf(err, "upload")
	}
	resp.Unmarshal(&ret)
	err = cloudprovider.Wait(time.Second*10, time.Hour*2, func() (bool, error) {
		task, err := self.GetTask(ret.TaskUUID)
		if err != nil {
			return false, err
		}
		if callback != nil {
			callback(90 + float32(task.PercentageComplete)*0.1)
		}
		log.Debugf("task %s %s status: %s", task.OperationType, task.UUID, task.ProgressStatus)
		if task.ProgressStatus == "Succeeded" {
			for _, entity := range task.EntityList {
				imageId = entity.EntityID
			}
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "wait image ready")
	}
	return self.GetImage(imageId)
}
