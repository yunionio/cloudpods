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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SStoragecache struct {
	ZoneId string
	region *SRegion
}

func (scache *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (scache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s/%s", scache.region.client.providerID, scache.region.GetId(), scache.ZoneId)
}

func (scache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s/%s", scache.region.client.providerName, scache.region.GetId(), scache.ZoneId)
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

func (scache *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	images, err := scache.region.GetImages(scache.ZoneId, "")
	if err != nil {
		return nil, err
	}
	iImages := []cloudprovider.ICloudImage{}
	for i := 0; i < len(images); i++ {
		images[i].storageCache = scache
		iImages = append(iImages, &images[i])
	}
	return iImages, nil
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

func (scache *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(image.ExternalId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", image.ExternalId)

		img, err := scache.region.GetImage(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		status := img.GetStatus()
		if api.CACHED_IMAGE_STATUS_READY == status && !isForce {
			return image.ExternalId, nil
		}
		log.Debugf("image %s status %s", image.ExternalId, status)
	} else {
		log.Debugf("UploadImage: no external ID")
	}

	return scache.uploadImage(ctx, userCred, image, isForce)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")

	meta, reader, size, err := modules.Images.Download(s, image.ImageId, string(qemuimg.QCOW2), false)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)

	// size, _ := meta.Int("size")
	img, err := self.region.CreateImage(self.ZoneId, image.ImageName, string(qemuimg.QCOW2), image.OsType, "", reader, size)
	if err != nil {
		return "", err
	}
	img.storageCache = self
	err = cloudprovider.WaitStatus(img, api.CACHED_IMAGE_STATUS_READY, time.Second*5, time.Minute*20) //windows镜像转换比较慢，等待时间稍微设长一些
	if err != nil {
		log.Errorf("waitting for image %s(%s) status ready timeout", img.Name, img.UUID)
	}
	return img.UUID, err
}

func (scache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (scache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
