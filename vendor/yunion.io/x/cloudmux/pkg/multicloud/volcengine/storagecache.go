// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	VolcEngineTags
	region *SRegion
}

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), imageId)
}

func (scache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", scache.region.client.cpcfg.Id, scache.region.GetId())
}

func (scache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", scache.region.client.cpcfg.Name, scache.region.GetId())
}

func (scache *SStoragecache) GetStatus() string {
	return "available"
}

func (scache *SStoragecache) Refresh() error {
	return nil
}

func (scache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", scache.region.client.cpcfg.Id, scache.region.GetGlobalId())
}

func (scache *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (scache *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := scache.region.GetImages("private", nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		images[i].storageCache = scache
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (scache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	img, err := scache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	img.storageCache = scache
	return img, nil
}

func (scache *SStoragecache) GetPath() string {
	return ""
}

func (scache *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return scache.uploadImage(ctx, image, callback)
}

func (scache *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	bucketName := GetBucketName(scache.region.GetId(), image.ImageId)
	exist, err := scache.region.IBucketExist(bucketName)
	if err != nil {
		return "", errors.Wrapf(err, "IBucketExist")
	}
	if !exist {
		err = scache.region.CreateIBucket(bucketName, "", "")
		if err != nil {
			return "", errors.Wrapf(err, "CreateIBucket")
		}
	}
	defer scache.region.DeleteIBucket(bucketName)

	reader, sizeBytes, err := image.GetReader(image.ImageId, string(qemuimgfmt.VMDK))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}
	bucket, err := scache.region.GetIBucketByName(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	body := multicloud.NewProgress(sizeBytes, 80, reader, callback)
	err = cloudprovider.UploadObject(ctx, bucket, image.ImageId, 0, body, sizeBytes, "", "", nil, false)
	if err != nil {
		return "", errors.Wrap(err, "cloudprovider.UploadObject")
	}
	defer bucket.DeleteObject(ctx, image.ImageId)

	imageBaseName := image.ImageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", image.ImageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	for {
		_, err = scache.region.GetImageByName(imageName)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				break
			} else {
				return "", err
			}
		}

		imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
		nameIdx += 1
		log.Debugf("uploadImage Match remote name %s", imageName)
	}

	log.Debugf("Import image %s", imageName)
	imageId, err := scache.region.ImportImage(imageName, image.OsArch, image.OsType, image.OsDistribution, image.OsVersion, bucketName, image.ImageId)

	if err != nil {
		return "", errors.Wrapf(err, "ImportImage %s %s", image.ImageId, bucketName)
	}
	return imageId, nil
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	if id == storageCache.GetGlobalId() {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}
