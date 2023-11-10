// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ImageStatusType string

const (
	ImageStatusCreating  ImageStatusType = "creating"
	ImageStatusAvailable ImageStatusType = "available"
	ImageStatusError     ImageStatusType = "error"
)

type ImageOwnerType string

const (
	ImageOwnerPrivate ImageOwnerType = "private"
	ImageOwnerShared  ImageOwnerType = "shared"
	ImageOwnerPublic  ImageOwnerType = "public"
)

type SImage struct {
	multicloud.SImageBase
	VolcEngineTags
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

	Architecture         string
	CreationTime         time.Time
	Description          string
	ImageId              string
	ImageName            string
	OSName               string
	OSType               string
	Visibility           string
	IsSupportCloudinit   bool
	IsSupportIoOptimized bool
	Platform             string
	Size                 int
	Status               ImageStatusType
	Usage                string
	BootMode             string
}

func (img *SImage) GetMinRamSizeMb() int {
	return 0
}

func (img *SImage) GetId() string {
	return img.ImageId
}

func (img *SImage) GetName() string {
	return img.ImageName
}

func (img *SImage) Delete(ctx context.Context) error {
	return img.storageCache.region.DeleteImage(img.ImageId)
}

func (img *SImage) GetGlobalId() string {
	return img.ImageId
}

func (img *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return img.storageCache
}

func (img *SImage) GetStatus() string {
	switch img.Status {
	case ImageStatusCreating:
		return api.CACHED_IMAGE_STATUS_SAVING
	case ImageStatusAvailable:
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case ImageStatusError:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (region *SRegion) ImportImage(name string, osArch string, osType string, platform, platformVersion string, bucket string, key string) (string, error) {
	params := map[string]string{
		"Architecture":    osArch,
		"OsType":          osType,
		"Platform":        platform,
		"PlatformVersion": platformVersion,
		"Tags.1.Key":      "Name",
		"Tags.2.Value":    name,
		"Url":             fmt.Sprintf("https://%s.%s/%s", bucket, region.getS3Endpoint(), key),
	}
	body, err := region.ecsRequest("ImportImage", params)
	if err != nil {
		return "", errors.Wrapf(err, "ImportImage")
	}
	imageId, err := body.GetString("ImageId")
	if err != nil {
		return "", errors.Wrap(err, "Unmarsh imageId failed")
	}
	return imageId, nil
}

func (region *SRegion) ExportImage(imageId, bucketName string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ImageId"] = imageId
	params["OssBucket"] = bucketName
	params["OssPrefix"] = fmt.Sprintf("%sexport", strings.Replace(imageId, "-", "", -1))

	body, err := region.ecsRequest("ExportImage", params)
	if err != nil {
		return "", errors.Wrapf(err, "ExportImage")
	}
	taskId, err := body.GetString("TaskId")
	if err != nil {
		return "", errors.Wrapf(err, "Unmarshal")
	}
	return taskId, nil
}

func (img *SImage) GetImageStatus() string {
	switch img.Status {
	case ImageStatusCreating:
		return cloudprovider.IMAGE_STATUS_QUEUED
	case ImageStatusAvailable:
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case ImageStatusError:
		return cloudprovider.IMAGE_STATUS_DELETED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (img *SImage) Refresh() error {
	image, err := img.storageCache.region.GetImage(img.ImageId)
	if err != nil {
		return err
	}
	return jsonutils.Update(img, image)
}

func (img *SImage) GetImageType() cloudprovider.TImageType {
	switch img.Visibility {
	case string(ImageOwnerPublic):
		return cloudprovider.ImageTypeSystem
	case string(ImageOwnerPrivate), string(ImageOwnerShared):
		return cloudprovider.ImageTypeCustomized
	default:
		return cloudprovider.ImageTypeCustomized
	}
}

func (img *SImage) GetSizeByte() int64 {
	return int64(img.Size) * 1024 * 1024 * 1024
}

func (img *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(img.getNormalizedImageInfo().OsType)
}

func (img *SImage) GetOsDist() string {
	return img.getNormalizedImageInfo().OsDistro
}

func (img *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if img.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(img.OSName, img.Architecture, img.OSType, img.Platform, "")
		img.imgInfo = &imgInfo
	}

	return img.imgInfo
}

func (img *SImage) GetFullOsName() string {
	return img.OSName
}

func (img *SImage) GetOsVersion() string {
	return img.getNormalizedImageInfo().OsVersion
}

func (img *SImage) GetOsLang() string {
	return img.getNormalizedImageInfo().OsLang
}

func (img *SImage) GetOsArch() string {
	if strings.Contains(img.Architecture, "arm") {
		return osprofile.OS_ARCH_ARM
	}
	return osprofile.OS_ARCH_X86_64
}

func (img *SImage) GetBios() cloudprovider.TBiosType {
	if img.BootMode == "UEFI" {
		return cloudprovider.UEFI
	}
	return cloudprovider.BIOS
}

func (img *SImage) GetMinOsDiskSizeGb() int {
	return 40
}

func (img *SImage) GetImageFormat() string {
	return "vhd"
}

func (img *SImage) GetCreatedAt() time.Time {
	return img.CreationTime
}

func (region *SRegion) GetImage(imageId string) (*SImage, error) {
	images, err := region.GetImages("", []string{imageId}, "")
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].ImageId == imageId {
			images[i].storageCache = region.getStoragecache()
			return &images[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, imageId)
}

func (region *SRegion) GetImageByName(name string) (*SImage, error) {
	images, err := region.GetImages("", nil, name)
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].ImageName == name {
			return &images[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (region *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	image, err := region.GetImage(imageId)
	if err != nil {
		return "", err
	}
	return image.Status, nil
}

func (region *SRegion) GetImages(visibility string, imageIds []string, name string) ([]SImage, error) {
	params := make(map[string]string)
	params["MaxResults"] = "100"
	for i, id := range imageIds {
		params[fmt.Sprintf("ImageIds.%d", i+1)] = id
	}
	if len(visibility) > 0 {
		params["Visibility"] = visibility
	}

	if len(name) > 0 {
		params["ImageName"] = name
	}

	ret := []SImage{}
	for {
		resp, err := region.ecsRequest("DescribeImages", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeImages")
		}
		part := struct {
			Images    []SImage
			NextToken string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Images...)
		if len(part.NextToken) == 0 || len(part.Images) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (region *SRegion) DeleteImage(imageId string) error {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ImageId"] = imageId
	params["Force"] = "true"

	_, err := region.ecsRequest("DeleteImage", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteImage fail")
	}
	return nil
}
