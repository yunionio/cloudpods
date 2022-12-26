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

package zstack

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/qemuimgfmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	ZStackTags
	ZoneId string
	region *SRegion
}

func (scache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s/%s", scache.region.client.cpcfg.Id, scache.region.GetId(), scache.ZoneId)
}

func (scache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s/%s", scache.region.client.cpcfg.Name, scache.region.GetId(), scache.ZoneId)
}

func (scache *SStoragecache) GetStatus() string {
	return "available"
}

func (scache *SStoragecache) Refresh() error {
	return nil
}

func (scache *SStoragecache) GetGlobalId() string {
	return scache.GetId()
}

func (scache *SStoragecache) IsEmulated() bool {
	return false
}

func (scache *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (scache *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := scache.region.GetImages(scache.ZoneId, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudImage{}
	for i := 0; i < len(images); i++ {
		images[i].storageCache = scache
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (scache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	images, err := scache.region.GetImages(scache.ZoneId, extId)
	if err != nil {
		return nil, err
	}
	if len(images) == 1 {
		images[0].storageCache = scache
		return &images[0], nil
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (scache *SStoragecache) GetPath() string {
	return ""
}

func (scache *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return scache.uploadImage(ctx, image, callback)
}

func (self *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	reader, size, err := image.GetReader(image.ImageId, string(qemuimgfmt.QCOW2))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	// size, _ := meta.Int("size")
	img, err := self.region.CreateImage(self.ZoneId, image.ImageName, string(qemuimgfmt.QCOW2), image.OsType, "", reader, size, callback)
	if err != nil {
		return "", err
	}
	img.storageCache = self
	err = cloudprovider.WaitStatus(img, api.CACHED_IMAGE_STATUS_ACTIVE, time.Second*5, time.Minute*20) //windows镜像转换比较慢，等待时间稍微设长一些
	if err != nil {
		log.Errorf("waitting for image %s(%s) status ready timeout", img.Name, img.UUID)
	}
	if callback != nil {
		callback(100)
	}
	return img.UUID, err
}

func (scache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (scache *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
