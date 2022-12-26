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

package jdcloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/models"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SImageBase
	JdcloudTags

	storageCache *SStoragecache

	imageInfo *imagetools.ImageInfo

	models.Image
}

func (i *SImage) GetCreatedAt() time.Time {
	return parseTime(i.CreateTime)
}

func (i *SImage) GetId() string {
	return i.ImageId
}

func (i *SImage) GetName() string {
	return i.Name
}

func (i *SImage) GetDescription() string {
	return i.Desc
}

func (i *SImage) GetGlobalId() string {
	return i.GetId()
}

func (i *SImage) GetStatus() string {
	switch i.Status {
	case "pending":
		return api.CACHED_IMAGE_STATUS_SAVING
	case "ready":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "deleting":
		return api.CACHED_IMAGE_STATUS_DELETING
	case "error":
		return api.CACHED_IMAGE_STATUS_UNKNOWN
	default:
		return api.CACHED_IMAGE_STATUS_UNKNOWN
	}
}

func (i *SImage) Refresh() error {
	return nil
}

func (i *SImage) IsEmulated() bool {
	return false
}

func (i *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if i.imageInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(i.Name, i.Architecture, i.OsType, i.Platform, i.OsVersion)
		i.imageInfo = &imgInfo
	}
	return i.imageInfo
}

func (i *SImage) GetFullOsName() string {
	return i.Name
}

func (i *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.getNormalizedImageInfo().OsType)
}

func (i *SImage) GetOsDist() string {
	return i.getNormalizedImageInfo().OsDistro
}

func (i *SImage) GetOsVersion() string {
	return i.getNormalizedImageInfo().OsVersion
}

func (i *SImage) GetOsLang() string {
	return i.getNormalizedImageInfo().OsLang
}

func (i *SImage) GetOsArch() string {
	return i.getNormalizedImageInfo().OsArch
}

func (i *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(i.getNormalizedImageInfo().OsBios)
}

func (i *SImage) GetMinOsDiskSizeGb() int {
	return i.SystemDiskSizeGB
}

func (i *SImage) GetMinRamSizeMb() int {
	return 0
}

func (i *SImage) GetImageFormat() string {
	return ""
}

func (i *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (i *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (i *SImage) GetSizeByte() int64 {
	return int64(i.SizeMB) * 1024 * 1024
}

func (i *SImage) GetImageType() cloudprovider.TImageType {
	switch i.ImageSource {
	case "jcloud":
		return cloudprovider.ImageTypeSystem
	case "marketplace":
		return cloudprovider.ImageTypeMarket
	case "self":
		return cloudprovider.ImageTypeCustomized
	case "shared":
		return cloudprovider.ImageTypeShared
	default:
		return cloudprovider.TImageType("")
	}
}

func (i *SImage) GetImageStatus() string {
	switch i.Status {
	case "pending":
		return cloudprovider.IMAGE_STATUS_QUEUED
	default:
		return cloudprovider.IMAGE_STATUS_ACTIVE
	}
}

func (r *SRegion) GetImage(imageId string) (*SImage, error) {
	req := apis.NewDescribeImageRequest(r.ID, imageId)
	client := client.NewVmClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeImage(req)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code >= 400 {
		err = fmt.Errorf(resp.Error.Message)
		return nil, err
	}
	return &SImage{
		Image: resp.Result.Image,
	}, nil
}

func (r *SRegion) GetImages(imageIds []string, imageSource string, pageNumber, pageSize int) ([]SImage, int, error) {
	req := apis.NewDescribeImagesRequestWithAllParams(r.ID, &imageSource, nil, nil, nil, imageIds, nil, nil, nil, nil, &pageNumber, &pageSize)
	client := client.NewVmClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeImages(req)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, 0, err
	}
	if resp.Error.Code >= 400 {
		if strings.Contains(resp.Error.Message, "secret key is nul") || strings.Contains(resp.Error.Message, "sign result is not same") {
			return nil, 0, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, resp.Error.Message)
		}
		err = fmt.Errorf(resp.Error.Message)
		return nil, 0, err
	}
	images := make([]SImage, len(resp.Result.Images))
	for i := range resp.Result.Images {
		images[i] = SImage{
			Image: resp.Result.Images[i],
		}
	}
	return images, resp.Result.TotalCount, nil
}
