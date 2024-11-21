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

package baidu

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	SBaiduTag
	region *SRegion
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Name, self.region.GetId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetGlobalId())
}

func (self *SStoragecache) IsEmulated() bool {
	return true
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images := []SImage{}
	for _, imageType := range []string{"Custom", "Sharing"} {
		part, err := self.region.GetImages(imageType)
		if err != nil {
			return nil, err
		}
		images = append(images, part...)
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		images[i].cache = self
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := self.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	image.cache = self
	return image, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
