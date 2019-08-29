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

package openstack

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
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (cache *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (cache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerID, cache.region.GetId())
}

func (cache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerName, cache.region.GetId())
}

func (cache *SStoragecache) GetStatus() string {
	return "available"
}

func (cache *SStoragecache) Refresh() error {
	return nil
}

func (cache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerID, cache.region.GetGlobalId())
}

func (cache *SStoragecache) IsEmulated() bool {
	return false
}

func (cache *SStoragecache) fetchImages() error {
	images, err := cache.region.GetImages("", ACTIVE, "")
	if err != nil {
		return err
	}
	cache.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i++ {
		images[i].storageCache = cache
		cache.iimages[i] = &images[i]
	}
	return nil
}

func (cache *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if cache.iimages == nil {
		if err := cache.fetchImages(); err != nil {
			return nil, err
		}
	}
	return cache.iimages, nil
}

func (cache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := cache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	image.storageCache = cache
	return image, nil
}

func (cache *SStoragecache) GetPath() string {
	return ""
}

func (cache *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(image.ExternalId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", image.ExternalId)

		statsu, err := cache.region.GetImageStatus(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if statsu == ACTIVE && !isForce {
			return image.ExternalId, nil
		}
	}
	log.Debugf("UploadImage: no external ID")
	return cache.uploadImage(ctx, userCred, image, isForce)
}

func (cache *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")

	meta, reader, _, err := modules.Images.Download(s, image.ImageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)

	imageBaseName := image.ImageName
	imageName := imageBaseName
	nameIdx := 1

	for {
		_, err = cache.region.GetImageByName(imageName)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				break
			} else {
				return "", err
			}
		}
		imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
		nameIdx++
	}

	img, err := cache.region.CreateImage(imageName)
	if err != nil {
		return "", err
	}

	img.storageCache = cache

	_, err = cache.region.client.StreamRequest(cache.region.Name, "image", "PUT", fmt.Sprintf("/v2/images/%s/file", img.ID), "", reader)
	if err != nil {
		return "", err
	}
	return img.ID, cloudprovider.WaitStatus(img, api.CACHED_IMAGE_STATUS_READY, 15*time.Second, 3600*time.Second)
}

func (cache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
