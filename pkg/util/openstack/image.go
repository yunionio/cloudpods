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

package openstack

import (
	"context"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	QUEUED         = "queued"         //	The Image service reserved an image ID for the image in the catalog but did not yet upload any image data.
	SAVING         = "saving"         //	The Image service is in the process of saving the raw data for the image into the backing store.
	ACTIVE         = "active"         //	The image is active and ready for consumption in the Image service.
	KILLED         = "killed"         // An image data upload error occurred.
	DELETED        = "deleted"        //	The Image service retains information about the image but the image is no longer available for use.
	PENDING_DELETE = "pending_delete" // Similar to the deleted status. An image in this state is not recoverable.
	DEACTIVATED    = "deactivated"    //	The image data is not available for use.
	UPLOADING      = "uploading"      // Data has been staged as part of the interoperable image import process. It is not yet available for use. (Since Image API 2.6)
	IMPORTING      = "importing"      //	The image data is being processed as part of the interoperable image import process, but is not yet available for use. (Since Image API 2.6)
)

type SImage struct {
	storageCache *SStoragecache

	Status          string
	Name            string
	Tags            []string
	ContainerFormat string
	CreatedAt       time.Time
	DiskFormat      string
	UpdatedAt       time.Time
	Visibility      string
	Self            string
	MinDisk         int
	Protected       bool
	ID              string
	File            string
	Checksum        string
	OsHashAlgo      string
	OsHashValue     string
	OsHidden        bool
	Owner           string
	Size            int
	MinRAM          int
	Schema          string
	VirtualSize     int
	visibility      string
}

func (image *SImage) GetMinRamSizeMb() int {
	return image.MinRAM
}

func (region *SRegion) GetImages(name string, status string, imageId string) ([]SImage, error) {
	params := url.Values{}
	if utils.IsInStringArray(status, []string{QUEUED, SAVING, ACTIVE, KILLED, DELETED, PENDING_DELETE, DEACTIVATED, UPLOADING, IMPORTING}) {
		params.Add("status", status)
	}
	if len(name) > 0 {
		params.Add("name", name)
	}
	if len(imageId) > 0 {
		params.Add("id", imageId)
	}
	_, resp, err := region.List("image", "/v2/images?"+params.Encode(), "", nil)
	if err != nil {
		return nil, err
	}
	images := []SImage{}
	return images, resp.Unmarshal(&images, "images")
}

func (image *SImage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (image *SImage) GetId() string {
	return image.ID
}

func (image *SImage) GetName() string {
	return image.Name
}

func (image *SImage) IsEmulated() bool {
	return false
}

func (image *SImage) GetGlobalId() string {
	return image.ID
}

func (image *SImage) Delete(ctx context.Context) error {
	return image.storageCache.region.DeleteImage(image.ID)
}

func (image *SImage) GetStatus() string {
	switch image.Status {
	case QUEUED, SAVING, UPLOADING, IMPORTING:
		return api.CACHED_IMAGE_STATUS_CACHING
	case ACTIVE:
		return api.CACHED_IMAGE_STATUS_READY
	case DELETED, DEACTIVATED, PENDING_DELETE, KILLED:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (image *SImage) GetImageStatus() string {
	switch image.Status {
	case QUEUED, SAVING, UPLOADING, IMPORTING:
		return cloudprovider.IMAGE_STATUS_SAVING
	case ACTIVE:
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case DELETED, DEACTIVATED, PENDING_DELETE:
		return cloudprovider.IMAGE_STATUS_DELETED
	case KILLED:
		return cloudprovider.IMAGE_STATUS_KILLED
	default:
		return cloudprovider.IMAGE_STATUS_DELETED
	}
}

func (image *SImage) Refresh() error {
	new, err := image.storageCache.region.GetImage(image.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(image, new)
}

func (image *SImage) GetImageType() string {
	switch image.Visibility {
	case "public":
		return cloudprovider.CachedImageTypeSystem
	default:
		return cloudprovider.CachedImageTypeCustomized
	}
}

func (image *SImage) GetSize() int64 {
	return int64(image.Size)
}

func (image *SImage) GetOsType() string {
	return "Linux"
}

func (image *SImage) GetOsDist() string {
	return "Linux"
}

func (image *SImage) GetOsVersion() string {
	return ""
}

func (image *SImage) GetOsArch() string {
	return "x86_64"
}

func (image *SImage) GetMinOsDiskSizeGb() int {
	return image.MinDisk
}

func (image *SImage) GetImageFormat() string {
	return image.DiskFormat
}

func (image *SImage) GetCreateTime() time.Time {
	return image.CreatedAt
}

func (region *SRegion) GetImage(imageId string) (*SImage, error) {
	images, err := region.GetImages("", "", imageId)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(images) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &images[0], nil
}

func (image *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return image.storageCache
}

func (region *SRegion) DeleteImage(imageId string) error {
	_, err := region.Delete("image", "/v2/images/"+imageId, "")
	return err
}

func (region *SRegion) GetImageStatus(imageId string) (string, error) {
	image, err := region.GetImage(imageId)
	if err != nil {
		return "", err
	}
	return image.Status, nil
}

func (region *SRegion) GetImageByName(name string) (*SImage, error) {
	images, err := region.GetImages(name, "", "")
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &images[0], nil
}

func (region *SRegion) CreateImage(imageName string) (*SImage, error) {
	params := map[string]string{
		"container_format": "bare",
		"disk_format":      "vmdk",
		"name":             imageName,
	}

	_, resp, err := region.Post("image", "/v2/images", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	image := &SImage{}
	return image, resp.Unmarshal(image)
}
