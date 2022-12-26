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

	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SImageBase
	InCloudSphereTags
	cache *SStoragecache

	imageInfo *imagetools.ImageInfo

	Name          string `json:"name"`
	FileSize      int64  `json:"fileSize"`
	RealSize      int64  `json:"realSize"`
	SourceType    string `json:"sourceType"`
	Format        string `json:"format"`
	FileType      string `json:"fileType"`
	Size          int64  `json:"size"`
	Date          string `json:"date"`
	Path          string `json:"path"`
	FTPServer     string `json:"ftpServer"`
	DataStoreID   string `json:"dataStoreId"`
	DataStoreName string `json:"dataStoreName"`
	ServerID      string `json:"serverId"`
	Md5           string `json:"md5"`
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return fmt.Sprintf("%s/%s/%s", self.cache.zone.Id, self.Path, self.Name)
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
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
	return self.FileSize
}

func (i *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.getNormalizedImageInfo().OsType)
}

func (i *SImage) GetOsDist() string {
	return i.getNormalizedImageInfo().OsDistro
}

func (i *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if i.imageInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(i.Name, "", "", "", "")
		i.imageInfo = &imgInfo
	}
	return i.imageInfo
}

func (i *SImage) GetFullOsName() string {
	return i.Name
}

func (i *SImage) GetOsVersion() string {
	return i.getNormalizedImageInfo().OsVersion
}

func (i *SImage) GetOsArch() string {
	return i.getNormalizedImageInfo().OsArch
}

func (i *SImage) GetOsLang() string {
	return i.getNormalizedImageInfo().OsLang
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	if self.GetOsType() == "windows" {
		return 40
	}
	return 30
}

func (self *SImage) GetImageFormat() string {
	return self.Format
}

func (i *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(i.getNormalizedImageInfo().OsBios)
}

type SImageInfo struct {
	OsType              string
	OsDist              string
	Model               string
	SocketLimit         int
	SupportCpuHotPlug   bool
	SupportMemHotPlug   bool
	SupportDiskHotPlug  bool
	SupportUefiBootMode bool
}

func (self *SImageInfo) IsEquals(name string) bool {
	model := imagetools.NormalizeImageInfo(self.Model, "", "", "", "")
	image := imagetools.NormalizeImageInfo(name, "", "", "", "")
	return model.OsDistro == image.OsDistro && model.OsVersion == image.OsVersion
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
			Object SImageInfo
		}
	}
}

func (self *SImageTree) ToList() []SImageInfo {
	ret := []SImageInfo{}
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
