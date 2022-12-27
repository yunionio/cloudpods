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
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	multicloud.SImageBase
	storageCache *SStoragecache
	OpenStackTags

	// normalized image info
	imgInfo *imagetools.ImageInfo

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
	Id              string
	File            string
	Checksum        string
	OsHashAlgo      string
	OsHashValue     string
	OsHidden        bool
	OsDistro        string
	OsType          string
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
	query := url.Values{}
	if len(status) > 0 {
		query.Set("status", status)
	}
	if len(name) > 0 {
		query.Set("name", name)
	}
	if len(imageId) > 0 {
		query.Set("id", imageId)
	}
	images := []SImage{}
	resource := "/v2/images"
	marker := ""
	for {
		if len(marker) > 0 {
			query.Set("marker", marker)
		}
		resp, err := region.imageList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "imageList")
		}
		part := struct {
			Images []SImage
			Next   string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		images = append(images, part.Images...)
		if len(part.Next) == 0 {
			break
		}
		if len(part.Next) > 0 {
			href, err := url.Parse(part.Next)
			if err != nil {
				marker = ""
			} else {
				marker = href.Query().Get("marker")
			}
		}
		if len(marker) == 0 {
			break
		}
	}
	return images, nil
}

func (image *SImage) GetId() string {
	return image.Id
}

func (image *SImage) GetName() string {
	return image.Name
}

func (image *SImage) IsEmulated() bool {
	return false
}

func (image *SImage) GetGlobalId() string {
	return image.Id
}

func (image *SImage) Delete(ctx context.Context) error {
	return image.storageCache.region.DeleteImage(image.Id)
}

func (image *SImage) GetStatus() string {
	switch image.Status {
	case QUEUED, SAVING, UPLOADING, IMPORTING:
		return api.CACHED_IMAGE_STATUS_CACHING
	case ACTIVE:
		return api.CACHED_IMAGE_STATUS_ACTIVE
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
	_image, err := image.storageCache.region.GetImage(image.Id)
	if err != nil {
		return errors.Wrap(err, "GetImage")
	}
	return jsonutils.Update(image, _image)
}

func (image *SImage) GetImageType() cloudprovider.TImageType {
	return cloudprovider.ImageTypeSystem
}

func (image *SImage) GetPublicScope() rbacscope.TRbacScope {
	switch image.Visibility {
	case "private":
		return rbacscope.ScopeNone
	default:
		return rbacscope.ScopeSystem
	}
}

func (image *SImage) GetProjectId() string {
	return image.Owner
}

func (image *SImage) GetSizeByte() int64 {
	return int64(image.Size)
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(self.Name, "", self.OsType, "", "")
		self.imgInfo = &imgInfo
	}

	return self.imgInfo
}

func (image *SImage) GetFullOsName() string {
	return image.Name
}

func (image *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(image.getNormalizedImageInfo().OsType)
}

func (image *SImage) GetOsDist() string {
	return image.getNormalizedImageInfo().OsDistro
}

func (image *SImage) GetOsVersion() string {
	return image.getNormalizedImageInfo().OsVersion
}

func (image *SImage) GetOsArch() string {
	return image.getNormalizedImageInfo().OsArch
}

func (image *SImage) GetOsLang() string {
	return image.getNormalizedImageInfo().OsLang
}

func (image *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(image.getNormalizedImageInfo().OsBios)
}

func (image *SImage) GetMinOsDiskSizeGb() int {
	if image.MinDisk > 0 {
		return image.MinDisk
	}
	return 30
}

func (image *SImage) GetImageFormat() string {
	return image.DiskFormat
}

func (image *SImage) GetCreatedAt() time.Time {
	return image.CreatedAt
}

func (region *SRegion) GetImage(imageId string) (*SImage, error) {
	resource := "/v2/images/" + imageId
	resp, err := region.imageGet(resource)
	if err != nil {
		return nil, errors.Wrapf(err, "imageGet(%s)", resource)
	}
	image := &SImage{}
	err = resp.Unmarshal(image)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return image, nil
}

func (image *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return image.storageCache
}

func (region *SRegion) DeleteImage(imageId string) error {
	_, err := region.imageDelete("/v2/images/" + imageId)
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

func (region *SRegion) CreateImage(imageName string, osType string, osDist string, minDiskGb int, minRam int, size int64, body io.Reader, callback func(progress float32)) (*SImage, error) {
	params := map[string]interface{}{
		"container_format":    "bare",
		"disk_format":         string(qemuimgfmt.QCOW2),
		"name":                imageName,
		"min_disk":            minDiskGb,
		"min_ram":             minRam,
		"os_type":             strings.ToLower(osType),
		"os_distro":           osDist,
		"hw_qemu_guest_agent": "yes",
	}

	resp, err := region.imagePost("/v2/images", params)
	if err != nil {
		return nil, errors.Wrap(err, "imagePost")
	}
	image := &SImage{}
	err = resp.Unmarshal(image)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	url := fmt.Sprintf("/v2/images/%s/file", image.Id)
	err = region.imageUpload(url, size, body, callback)
	if err != nil {
		return nil, errors.Wrap(err, "imageUpload")
	}
	return image, nil
}
