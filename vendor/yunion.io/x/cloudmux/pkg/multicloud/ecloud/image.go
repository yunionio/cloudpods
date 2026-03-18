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
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SImageBase
	EcloudTags
	SZoneRegionBase

	storageCache *SStoragecache
	imgInfo      *imagetools.ImageInfo

	// 字段与 IMS OpenAPI listImageRespV2/getImageRespV2 对齐
	ImageId         string    `json:"imageId,omitempty"`
	ServerId        string    `json:"serverId,omitempty"`
	ImageAlias      string    `json:"imageAlias,omitempty"`
	Name            string    `json:"name,omitempty"`
	Url             string    `json:"url,omitempty"`
	SrourceImageId  string    `json:"sourceImageId,omitempty"`
	Status          string    `json:"status,omitempty"`
	SizeMb          int       `json:"size"` // IMS 返回 size，按 MB 处理
	IsPublic        int       `json:"isPublic,omitempty"`
	ImageSource     string    `json:"imageSource,omitempty"` // 通过 imageSource 判断是否公共镜像
	CreateTime      time.Time `json:"-"`                     // 旧接口可能直接反序列化为 time.Time
	CreateTimeStr   string    `json:"createTime,omitempty"`
	Note            string    `json:"note,omitempty"`
	OsType          string    `json:"osType,omitempty"`
	MinDiskGB       int       `json:"minDisk,omitempty"`
	ImageType       string    `json:"imageType,omitempty"`
	PublicImageType string    `json:"publicImageType,omitempty"`
	BackupType      string    `json:"backupType,omitempty"`
	BackupWay       string    `json:"backupWay,omitempty"`
	SnapshotId      string    `json:"snapshotId,omitempty"`
	OsName          string    `json:"osName,omitempty"`
}

func (r *SRegion) GetImage(imageId string) (*SImage, error) {
	// 使用 IMS OpenAPI 镜像详情：GET /api/openapi-ims/user/v5/image/{imageId}
	req := NewOpenApiEbsRequest(r.RegionId, fmt.Sprintf("/api/openapi-ims/user/v5/image/%s", imageId), nil, nil)
	var image SImage
	err := r.client.doGet(context.Background(), req.Base(), &image)
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (r *SRegion) GetImages(isPublic bool) ([]SImage, error) {
	// IMS 接口要求 region（资源池 ID），否则报 CSLOPENSTACK_COMPUTE_IMAGE_PARAM_INVALID
	poolId := regionIdToPoolId[r.RegionId]
	if poolId == "" {
		poolId = r.RegionId
	}
	// IMS OpenAPI ListImageRespV2
	// GET /api/openapi-ims/user/v5/image?region=CIDC-RP-xx&imageSource=PUBLIC|PRIVATE
	query := map[string]string{
		"region": poolId,
	}
	if isPublic {
		query["imageSource"] = "PUBLIC"
	} else {
		query["imageSource"] = "PRIVATE"
	}
	req := NewOpenApiEbsRequest(r.RegionId, "/api/openapi-ims/user/v5/image", query, nil)
	images := make([]SImage, 0, 20)
	if err := r.client.doList(context.Background(), req.Base(), &images); err != nil {
		return nil, err
	}
	return images, nil
}

func (i *SImage) GetCreatedAt() time.Time {
	if len(i.CreateTimeStr) > 0 {
		// IMS 接口时间格式一般为 "2006-01-02 15:04:05"
		if t, err := time.Parse("2006-01-02 15:04:05", i.CreateTimeStr); err == nil {
			return t
		}
		// 兼容 RFC3339
		if t, err := time.Parse(time.RFC3339, i.CreateTimeStr); err == nil {
			return t
		}
	}
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

func (i *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if i.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(i.OsName, "", i.OsType, "", "")
		i.imgInfo = &imgInfo
	}
	return i.imgInfo
}

func (i *SImage) GetFullOsName() string {
	return i.OsName
}

func (i *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.getNormalizedImageInfo().OsType)
}

func (i *SImage) GetOsArch() string {
	return i.getNormalizedImageInfo().OsArch
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

func (i *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(i.getNormalizedImageInfo().OsBios)
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
