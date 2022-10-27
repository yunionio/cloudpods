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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	EcloudTags
	region *SRegion
}

func (sc *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (sc *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (sc *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (sc *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", sc.region.client.cpcfg.Id, sc.region.GetId())
}

func (sc *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", sc.region.client.cpcfg.Name, sc.region.GetId())
}

func (sc *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", sc.region.client.cpcfg.Id, sc.region.GetGlobalId())
}

func (sc *SStoragecache) GetStatus() string {
	return "available"
}

func (sc *SStoragecache) Refresh() error {
	return nil
}

func (sc *SStoragecache) IsEmulated() bool {
	return false
}

func (sc *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (sc *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	imagesSelf, err := sc.region.GetImages(false)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range imagesSelf {
		imagesSelf[i].storageCache = sc
		ret = append(ret, &imagesSelf[i])
	}
	return ret, nil
}

func (sc *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := sc.region.GetImage(extId)
	if err != nil {
		return nil, errors.Wrap(err, "SStoragecache.GetIImageById.GetImage")
	}

	image.storageCache = sc
	return image, err
}

func (sc *SStoragecache) GetPath() string {
	return ""
}
