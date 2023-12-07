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

package cloudpods

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/image"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

type SImage struct {
	multicloud.SImageBase
	CloudpodsTags
	cache *SStoragecache

	api.ImageDetails
}

func (self *SImage) GetProjectId() string {
	return self.TenantId
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) GetId() string {
	return self.Id
}

func (self *SImage) GetGlobalId() string {
	return self.Id
}

func (self *SImage) GetStatus() string {
	return self.Status
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.cache.region.cli.delete(&modules.Images, self.Id)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetSizeByte() int64 {
	return self.Size
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	if self.IsStandard != nil && *self.IsStandard {
		return cloudprovider.ImageTypeSystem
	}
	return cloudprovider.ImageTypeCustomized
}

func (self *SImage) GetImageStatus() string {
	return self.Status
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	osType, ok := self.Properties["os_type"]
	if ok {
		return cloudprovider.TOsType(osType)
	}
	return cloudprovider.OsTypeLinux
}

func (self *SImage) GetOsDist() string {
	osDist, ok := self.Properties["os_distribution"]
	if ok {
		return osDist
	}
	return ""
}

func (self *SImage) GetOsVersion() string {
	osVer, ok := self.Properties["os_version"]
	if ok {
		return osVer
	}
	return ""
}

func (self *SImage) GetOsArch() string {
	osArch, ok := self.Properties["os_arch"]
	if ok {
		return osArch
	}
	return ""
}

func (self *SImage) GetOsLang() string {
	osLang, ok := self.Properties["os_language"]
	if ok {
		return osLang
	}
	return ""
}

func (self *SImage) GetBios() cloudprovider.TBiosType {
	uefi, ok := self.Properties["uefi_support"]
	if ok && utils.ToBool(uefi) {
		return cloudprovider.UEFI
	}
	return cloudprovider.BIOS
}

func (img *SImage) GetFullOsName() string {
	imgInfo := imagetools.NormalizeImageInfo("", img.GetOsArch(), string(img.GetOsType()), img.GetOsDist(), img.GetOsVersion())
	return imgInfo.GetFullOsName()
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.MinDiskMB / 1024)
}

func (self *SImage) GetMinRamSizeMb() int {
	return int(self.MinRamMB)
}

func (self *SImage) GetImageFormat() string {
	return self.DiskFormat
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SImage) Refresh() error {
	image, err := self.cache.region.GetImage(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := self.region.GetImages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		images[i].cache = self
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (self *SRegion) GetImages() ([]SImage, error) {
	params := map[string]interface{}{
		"is_guest_image": false,
	}
	images := []SImage{}
	return images, self.list(&modules.Images, params, &images)
}

func (self *SRegion) GetImage(id string) (*SImage, error) {
	image := &SImage{}
	resp, err := modules.Images.GetById(self.cli.s, id, nil)
	if err != nil {
		return nil, err
	}
	return image, resp.Unmarshal(image)
}

func (self *SRegion) UploadImage(ctx context.Context, opts *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	reader, sizeByte, err := opts.GetReader(opts.ImageId, string(qemuimgfmt.QCOW2))
	if err != nil {
		return "", err
	}

	params := map[string]interface{}{
		"generate_name": opts.ImageName,
		"properties": map[string]string{
			"os_type":         opts.OsType,
			"os_distribution": opts.OsDistribution,
			"os_arch":         opts.OsArch,
			"os_version":      opts.OsVersion,
		},
	}

	body := multicloud.NewProgress(sizeByte, 90, reader, callback)
	resp, err := modules.Images.Upload(self.cli.s, jsonutils.Marshal(params), body, sizeByte)
	if err != nil {
		return "", err
	}
	return resp.GetString("id")
}
