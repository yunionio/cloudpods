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

package bingocloud

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SResourceBase
	BingoTags
	cache *SStoragecache

	imageInfo *imagetools.ImageInfo

	Architecture       string `json:"architecture"`
	BlockDeviceMapping string `json:"blockDeviceMapping"`
	Bootloader         string `json:"bootloader"`
	Clonemode          string `json:"clonemode"`
	ClusterId          string `json:"clusterId"`
	Contentmode        string `json:"contentmode"`
	DefaultStorageId   string `json:"defaultStorageId"`
	Description        string `json:"description"`
	DiskBus            string `json:"diskBus"`
	ExtendDisk         string `json:"extendDisk"`
	Features           string `json:"features"`
	Hypervisor         string `json:"hypervisor"`
	ImageId            string `json:"imageId"`
	ImageLocation      string `json:"imageLocation"`
	ImageOwnerId       string `json:"imageOwnerId"`
	ImagePath          string `json:"imagePath"`
	ImageSize          int64  `json:"imageSize"`
	ImageState         string `json:"imageState"`
	ImageType          string `json:"imageType"`
	IsBareMetal        string `json:"isBareMetal"`
	IsPublic           bool   `json:"isPublic"`
	KernelId           string `json:"kernelId"`
	Name               string `json:"name"`
	OsId               string `json:"osId"`
	OSName             string `json:"osName"`
	Platform           string `json:"platform"`
	RamdiskId          string `json:"ramdiskId"`
	RootDeviceName     string `json:"rootDeviceName"`
	RootDeviceType     string `json:"rootDeviceType"`
	ScheduleTags       string `json:"scheduleTags"`
	Shared             string `json:"shared"`
	Sharemode          string `json:"sharemode"`
	StateReason        string `json:"stateReason"`
	StorageId          string `json:"storageId"`
}

func (self *SImage) GetId() string {
	return self.ImageId
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetSizeByte() int64 {
	return self.ImageSize
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	if self.IsPublic {
		return cloudprovider.ImageTypeSystem
	}
	return cloudprovider.ImageTypeCustomized
}

func (self *SImage) GetImageStatus() string {
	return ""
}

func (i *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if i.imageInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(i.OSName, i.Architecture, i.Platform, "", "")
		i.imageInfo = &imgInfo
	}
	return i.imageInfo
}

func (i *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.getNormalizedImageInfo().OsType)
}

func (i *SImage) GetFullOsName() string {
	return i.OSName
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

func (self *SImage) GetMinOsDiskSizeGb() int {
	return 0
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetImageFormat() string {
	return "raw"
}

func (self *SImage) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SImage) UEFI() bool {
	return self.Bootloader == "uefi"
}

func (self *SImage) GetPublicScope() rbacscope.TRbacScope {
	if self.Shared == "true" {
		return rbacscope.ScopeSystem
	}
	return rbacscope.ScopeDomain
}

func (self *SImage) GetSubImages() []cloudprovider.SSubImage {
	return []cloudprovider.SSubImage{}
}

func (self *SImage) GetProjectId() string {
	return ""
}

func (self *SImage) GetStatus() string {
	switch self.ImageState {
	case "available":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	default:
		return self.ImageState
	}
}

func (self *SRegion) GetImages(id, nextToken string) ([]SImage, string, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["imageId"] = id
	}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	resp, err := self.invoke("DescribeImages", params)
	if err != nil {
		return nil, "", err
	}
	ret := struct {
		NextToken string
		ImagesSet []SImage
	}{}
	resp.Unmarshal(&ret)
	return ret.ImagesSet, ret.NextToken, nil
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	part, nextToken, err := self.region.GetImages("", "")
	if err != nil {
		return nil, err
	}
	images := []SImage{}
	images = append(images, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.region.GetImages("", nextToken)
		if err != nil {
			return nil, err
		}
		images = append(images, part...)
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		if images[i].StorageId == self.storageId {
			images[i].cache = self
			ret = append(ret, &images[i])
		}
	}
	return ret, nil
}

func (self *SStoragecache) GetIImageById(id string) (cloudprovider.ICloudImage, error) {
	images, _, err := self.region.GetImages(id, "")
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].GetGlobalId() == id && images[i].StorageId == self.storageId {
			images[i].cache = self
			return &images[i], nil
		}
	}

	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
