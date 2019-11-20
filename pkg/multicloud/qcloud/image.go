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

package qcloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type ImageStatusType string

const (
	ImageStatusCreating  ImageStatusType = "CREATING"
	ImageStatusNormal    ImageStatusType = "NORMAL"
	ImageStatusSycing    ImageStatusType = "SYNCING"
	ImageStatusImporting ImageStatusType = "IMPORTING"
	ImageStatusUsing     ImageStatusType = "USING"
	ImageStatusDeleting  ImageStatusType = "DELETING"
)

type SImage struct {
	storageCache *SStoragecache

	ImageId            string          //	镜像ID
	OsName             string          //	镜像操作系统
	ImageType          string          //	镜像类型
	CreatedTime        time.Time       //	镜像创建时间
	ImageName          string          //	镜像名称
	ImageDescription   string          //	镜像描述
	ImageSize          int             //	镜像大小
	Architecture       string          //	镜像架构
	ImageState         ImageStatusType //	镜像状态
	Platform           string          //	镜像来源平台
	ImageCreator       string          //	镜像创建者
	ImageSource        string          //	镜像来源
	SyncPercent        int             //	同步百分比
	IsSupportCloudinit bool            //	镜像是否支持cloud-init
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SRegion) GetImages(status string, owner string, imageIds []string, name string, offset int, limit int) ([]SImage, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	filter := 0
	if len(status) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "image-state"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = status
		filter++
	}
	if imageIds != nil && len(imageIds) > 0 {
		for index, imageId := range imageIds {
			params[fmt.Sprintf("ImageIds.%d", index)] = imageId
		}
	}
	if len(owner) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "image-type"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = owner
		filter++
	}

	if len(name) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "image-name"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = name
		filter++
	}

	images := make([]SImage, 0)
	body, err := self.cvmRequest("DescribeImages", params, true)
	if err != nil {
		return nil, 0, err
	}
	err = body.Unmarshal(&images, "ImageSet")
	if err != nil {
		return nil, 0, err
	}
	for i := 0; i < len(images); i++ {
		images[i].storageCache = self.getStoragecache()
	}
	total, _ := body.Float("TotalCount")
	return images, int(total), nil
}

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SImage) GetId() string {
	return self.ImageId
}

func (self *SImage) GetName() string {
	return self.ImageName
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) GetGlobalId() string {
	return self.ImageId
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.ImageId)
}

func (self *SImage) GetStatus() string {
	switch self.ImageState {
	case ImageStatusCreating, ImageStatusSycing, ImageStatusImporting:
		return api.CACHED_IMAGE_STATUS_CACHING
	case ImageStatusNormal, ImageStatusUsing:
		return api.CACHED_IMAGE_STATUS_READY
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.ImageState {
	case ImageStatusCreating, ImageStatusSycing, ImageStatusImporting:
		return cloudprovider.IMAGE_STATUS_SAVING
	case ImageStatusNormal, ImageStatusUsing:
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case ImageStatusDeleting:
		return cloudprovider.IMAGE_STATUS_DELETED
	default:
		return cloudprovider.IMAGE_STATUS_DELETED
	}
}

func (self *SImage) Refresh() error {
	new, err := self.storageCache.region.GetImage(self.ImageId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SImage) GetImageType() string {
	switch self.ImageType {
	case "PUBLIC_IMAGE":
		return cloudprovider.CachedImageTypeSystem
	case "PRIVATE_IMAGE":
		return cloudprovider.CachedImageTypeCustomized
	default:
		return cloudprovider.CachedImageTypeCustomized
	}
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.ImageSize) * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() string {
	switch self.Platform {
	case "Windows", "FreeBSD":
		return self.Platform
	default:
		return "Linux"
	}
}

func (self *SImage) GetOsDist() string {
	return self.Platform
}

func (self *SImage) GetOsVersion() string {
	return imagetools.NormalizeImageInfo(self.OsName, "", "", "", "").OsVersion
}

func (self *SImage) GetOsArch() string {
	return self.Architecture
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return 50
}

func (self *SImage) GetImageFormat() string {
	return "qcow2"
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.CreatedTime
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	images, _, err := self.GetImages("", "", []string{imageId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("image %s not found", imageId)
	}
	return &images[0], nil
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) DeleteImage(imageId string) error {
	params := make(map[string]string)
	params["ImageIds.0"] = imageId

	_, err := self.cvmRequest("DeleteImages", params, true)
	return err
}

func (self *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	image, err := self.GetImage(imageId)
	if err != nil {
		return "", err
	}
	return image.ImageState, nil
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	images, _, err := self.GetImages("", "", nil, name, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &images[0], nil
}

type ImportImageOsListSupported struct {
	Linux   []string //"CentOS|Ubuntu|Debian|OpenSUSE|SUSE|CoreOS|FreeBSD|Other Linux"
	Windows []string //"Windows Server 2008|Windows Server 2012|Windows Server 2016"
}

type ImportImageOsVersionSet struct {
	Architecture []string
	OsName       string //"CentOS|Ubuntu|Debian|OpenSUSE|SUSE|CoreOS|FreeBSD|Other Linux|Windows Server 2008|Windows Server 2012|Windows Server 2016"
	OsVersions   []string
}

type SupportImageSet struct {
	ImportImageOsListSupported ImportImageOsListSupported
	ImportImageOsVersionSet    []ImportImageOsVersionSet
}

func (self *SRegion) GetSupportImageSet() (*SupportImageSet, error) {
	body, err := self.cvmRequest("DescribeImportImageOs", map[string]string{}, true)
	if err != nil {
		return nil, err
	}
	imageSet := SupportImageSet{}
	return &imageSet, body.Unmarshal(&imageSet)
}

func (self *SRegion) GetImportImageParams(name string, osArch, osDist, osVersion string, imageUrl string) (map[string]string, error) {
	params := map[string]string{}
	imageSet, err := self.GetSupportImageSet()
	if err != nil {
		return nil, err
	}
	osType := ""
	for _, _imageSet := range imageSet.ImportImageOsVersionSet {
		if strings.ToLower(osDist) == strings.ToLower(_imageSet.OsName) { //Linux一般可正常匹配
			osType = _imageSet.OsName
		} else if strings.Contains(strings.ToLower(_imageSet.OsName), "windows") && strings.Contains(strings.ToLower(osDist), "windows") {
			info := strings.Split(_imageSet.OsName, " ")
			_osVersion := "2008"
			for _, version := range info {
				if _, err := strconv.Atoi(version); err == nil {
					_osVersion = version
					break
				}
			}
			if strings.Contains(osDist+osVersion, _osVersion) {
				osType = _imageSet.OsName
			}
		}
		if len(osType) == 0 {
			continue
		}
		if !utils.IsInStringArray(osArch, _imageSet.Architecture) {
			osArch = "x86_64"
		}
		for _, _osVersion := range _imageSet.OsVersions {
			if strings.HasPrefix(osVersion, _osVersion) {
				osVersion = _osVersion
				break
			}
		}
		if !utils.IsInStringArray(osVersion, _imageSet.OsVersions) {
			osVersion = "-"
			if len(_imageSet.OsVersions) > 0 {
				osVersion = _imageSet.OsVersions[0]
			}
		}
		break
	}
	if len(osType) == 0 {
		osType = "Other Linux"
		osArch = "x86_64"
		osVersion = "-"
	}

	params["ImageName"] = name
	params["OsType"] = osType
	params["OsVersion"] = osVersion
	params["Architecture"] = osArch // "x86_64|i386"
	params["ImageUrl"] = imageUrl
	params["Force"] = "true"
	return params, nil
}

func (self *SRegion) ImportImage(name string, osArch, osDist, osVersion string, imageUrl string) (*SImage, error) {
	params, err := self.GetImportImageParams(name, osArch, osDist, osVersion, imageUrl)
	if err != nil {
		return nil, err
	}

	log.Debugf("Upload image with params %#v", params)

	if _, err := self.cvmRequest("ImportImage", params, true); err != nil {
		return nil, err
	}
	for i := 0; i < 8; i++ {
		image, err := self.GetImageByName(name)
		if err == nil {
			return image, nil
		}
		time.Sleep(time.Minute * time.Duration(i))
	}
	return nil, cloudprovider.ErrNotFound
}
