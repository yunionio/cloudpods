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

package oracle

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
	SOracleTag
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

	DisplayName   string
	Id            string
	LaunchMode    string
	LaunchOptions struct {
		BootVolumeType                  string
		Firmware                        string
		NetworkType                     string
		RemoteDataVolumeType            string
		IsPvEncryptionInTransitEnabled  bool
		IsConsistentVolumeNamingEnabled string
	}
	LifecycleState         string
	OperatingSystem        string
	OperatingSystemVersion string
	SizeInMBs              int
	BillableSizeInGBs      int
	TimeCreated            time.Time
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return self.Id
}

func (self *SImage) GetName() string {
	return self.DisplayName
}

func (self *SImage) GetGlobalId() string {
	return self.Id
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SImage) GetStatus() string {
	// AVAILABLE, DELETED, DISABLED, EXPORTING, IMPORTING, PROVISIONING
	switch self.LifecycleState {
	case "PROVISIONING", "IMPORTING":
		return api.CACHED_IMAGE_STATUS_CACHING
	case "AVAILABLE", "EXPORTING":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.LifecycleState {
	case "PROVISIONING", "IMPORTING":
		return cloudprovider.IMAGE_STATUS_SAVING
	case "AVAILABLE", "EXPORTING":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "DELETED", "DISABLED":
		return cloudprovider.IMAGE_STATUS_DELETED
	default:
		return cloudprovider.IMAGE_STATUS_DELETED
	}
}

func (self *SImage) Refresh() error {
	image, err := self.storageCache.region.GetImage(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	return cloudprovider.ImageTypeSystem
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.SizeInMBs) * 1024 * 1024
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(self.DisplayName, "", self.OperatingSystem, "", self.OperatingSystemVersion)
		self.imgInfo = &imgInfo
	}
	return self.imgInfo
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.getNormalizedImageInfo().OsType)
}

func (self *SImage) GetOsDist() string {
	return self.getNormalizedImageInfo().OsDistro
}

func (self *SImage) GetOsVersion() string {
	return self.getNormalizedImageInfo().OsVersion
}

func (self *SImage) GetOsArch() string {
	return self.getNormalizedImageInfo().OsArch
}

func (img *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(img.getNormalizedImageInfo().OsBios)
}

func (img *SImage) GetFullOsName() string {
	return fmt.Sprintf("%s %s", img.OperatingSystem, img.OperatingSystemVersion)
}

func (img *SImage) GetOsLang() string {
	return img.getNormalizedImageInfo().OsLang
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return self.SizeInMBs / 1024
}

func (self *SImage) GetImageFormat() string {
	return "qcow2"
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.TimeCreated
}

func (self *SRegion) GetImage(id string) (*SImage, error) {
	resp, err := self.get(SERVICE_IAAS, "images", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SImage{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) GetImages() ([]SImage, error) {
	resp, err := self.list(SERVICE_IAAS, "images", nil)
	if err != nil {
		return nil, err
	}
	ret := []SImage{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
