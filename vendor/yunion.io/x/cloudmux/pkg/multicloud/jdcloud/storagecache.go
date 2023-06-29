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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	JdcloudTags
	region *SRegion
}

func (sc *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", sc.region.cpcfg.Id, sc.region.GetId())
}

func (sc *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", sc.region.cpcfg.Name, sc.region.GetName())
}

func (sc *SStoragecache) GetStatus() string {
	return "available"
}

func (sc *SStoragecache) Refresh() error {
	return nil
}

func (sc *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", sc.region.cpcfg.Id, sc.region.GetGlobalId())
}

func (sc *SStoragecache) IsEmulated() bool {
	return false
}

func (sc *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (sc *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images := make([]SImage, 0)
	n := 1
	for {
		parts, total, err := sc.region.GetImages([]string{}, "private", n, 100)
		if err != nil {
			return nil, err
		}
		images = append(images, parts...)
		if len(images) >= total {
			break
		}
		n++
	}
	ret := make([]cloudprovider.ICloudImage, len(images))
	for i := range ret {
		images[i].storageCache = sc
		ret[i] = &images[i]
	}
	return ret, nil
}

func (sc *SStoragecache) GetIImageById(exId string) (cloudprovider.ICloudImage, error) {
	img, err := sc.region.GetImage(exId)
	if err != nil {
		return nil, err
	}
	img.storageCache = sc
	return img, nil
}

func (sc *SStoragecache) GetPath() string {
	return ""
}

func (sc *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return "", cloudprovider.ErrNotSupported
}
