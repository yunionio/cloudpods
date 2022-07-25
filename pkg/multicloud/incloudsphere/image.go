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

package incloudsphere

import (
	"context"
	"fmt"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type SImage struct {
	multicloud.SImageBase
	multicloud.InCloudSphereTags
	cache *SStoragecache

	OsType              string
	OsDist              string
	Model               string
	SocketLimit         int
	SupportCpuHotPlug   bool
	SupportMemHotPlug   bool
	SupportDiskHotPlug  bool
	SupportUefiBootMode bool
}

func (self *SRegion) GetImages() ([]SImage, error) {
	ret := []SImage{}
	return ret, self.list("/vms/guestos", nil, &ret)
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return self.Model
}

func (self *SImage) GetName() string {
	return self.Model
}

func (self *SImage) IsEmulated() bool {
	return true
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SImage) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", self.OsType, self.OsDist, self.Model)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetStatus() string {
	return api.CACHED_IMAGE_STATUS_ACTIVE
}

func (self *SImage) GetImageStatus() string {
	return cloudprovider.IMAGE_STATUS_ACTIVE
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	return cloudprovider.ImageTypeSystem
}

func (self *SImage) GetSizeByte() int64 {
	return 20 * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	if strings.Contains(strings.ToLower(self.OsType), "windows") {
		return cloudprovider.OsTypeWindows
	}
	return cloudprovider.OsTypeLinux
}

func (self *SImage) GetOsDist() string {
	return self.getNormalizedImageInfo().OsDistro
}

func (self *SImage) getNormalizedImageInfo() imagetools.ImageInfo {
	return imagetools.NormalizeImageInfo(self.Model, "", "", "", "")
}

func (self *SImage) GetOsVersion() string {
	return self.getNormalizedImageInfo().OsVersion
}

func (self *SImage) GetOsArch() string {
	return self.getNormalizedImageInfo().OsArch
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	if self.GetOsType() == "windows" {
		return 40
	}
	return 30
}

func (self *SImage) GetImageFormat() string {
	return "vhd"
}

func (self *SImage) UEFI() bool {
	return false
}

type SImageTree struct {
	Id       string
	Name     string
	Children []struct {
		Id       string
		Name     string
		Children []struct {
			Id     string
			Name   string
			Object SImage
		}
	}
}

func (self *SImageTree) ToList() []SImage {
	ret := []SImage{}
	for i := range self.Children {
		for j := range self.Children[i].Children {
			self.Children[i].Children[j].Object.OsType = self.Name
			self.Children[i].Children[j].Object.OsDist = self.Children[i].Name
			ret = append(ret, self.Children[i].Children[j].Object)
		}
	}
	return ret
}

func (self *SRegion) GetImageTrees() ([]SImageTree, error) {
	ret := []SImageTree{}
	return ret, self.get("/vms/ostrees", nil, &ret)
}
