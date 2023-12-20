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

package huawei

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	HuaweiTags
	region *SRegion
}

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), imageId)
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Name, self.region.GetId())
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetGlobalId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	imagesSelf, err := self.region.GetImages("", "", "private", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}

	ret := []cloudprovider.ICloudImage{}
	for i := range imagesSelf {
		imagesSelf[i].storageCache = self
		ret = append(ret, &imagesSelf[i])
	}
	return ret, nil
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := self.region.GetImage(extId)
	if err != nil {
		return nil, errors.Wrap(err, "self.region.GetImage")
	}
	image.storageCache = self
	return image, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return self.uploadImage(ctx, image, callback)
}

func (self *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	bucketName := GetBucketName(self.region.GetId(), image.ImageId)

	exist, _ := self.region.IBucketExist(bucketName)
	if !exist {
		err := self.region.CreateIBucket(bucketName, "", "")
		if err != nil {
			return "", errors.Wrap(err, "CreateIBucket")
		}
	}
	defer self.region.DeleteIBucket(bucketName)

	reader, sizeByte, err := image.GetReader(image.ImageId, string(qemuimgfmt.VMDK))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	minDiskGB := int64(math.Ceil(float64(image.MinDiskMb) / 1024))
	// 在使用OBS桶的外部镜像文件制作镜像时生效且为必选字段。取值为40～1024GB。
	if minDiskGB < 40 {
		minDiskGB = 40
	} else if minDiskGB > 1024 {
		minDiskGB = 1024
	}

	bucket, err := self.region.GetIBucketByName(bucketName)
	if err != nil {
		return "", errors.Wrapf(err, "GetIBucketByName %s", bucketName)
	}

	body := multicloud.NewProgress(sizeByte, 95, reader, callback)
	err = cloudprovider.UploadObject(context.Background(), bucket, image.ImageId, 0, body, sizeByte, "", "", nil, false)
	if err != nil {
		return "", errors.Wrap(err, "cloudprovider.UploadObject")
	}

	defer bucket.DeleteObject(context.Background(), image.ImageId)

	// check image name, avoid name conflict
	imageBaseName := image.ImageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", image.ImageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	for {
		_, err = self.region.GetImageByName(imageName)
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

	jobId, err := self.region.ImportImageJob(imageName, image.OsDistribution, image.OsVersion, image.OsArch, bucketName, image.ImageId, int64(minDiskGB))

	if err != nil {
		log.Errorf("ImportImage error %s %s %s %s", jobId, image.ImageId, bucketName, err)
		return "", err
	}

	// timeout: 1hour = 3600 seconds
	err = self.region.waitTaskStatus(SERVICE_IMS_V1, jobId, TASK_SUCCESS, 15*time.Second, 3600*time.Second)
	if err != nil {
		log.Errorf("waitTaskStatus %s", err)
		return "", err
	}

	if callback != nil {
		callback(100)
	}

	return self.region.GetTaskEntityID(SERVICE_IMS, jobId, "image_id")
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) GetIStoragecacheById(idstr string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == idstr {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}
