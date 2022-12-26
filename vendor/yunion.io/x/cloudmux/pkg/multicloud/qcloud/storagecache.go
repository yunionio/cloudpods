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

package qcloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	QcloudTags
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
	return false
}

func (self *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	// if imageId, err := self.region.createIImage(snapshoutId, imageName, imageDesc); err != nil {
	// 	return nil, err
	// } else if image, err := self.region.GetImage(imageId); err != nil {
	// 	return nil, err
	// } else {
	// 	image.storageCache = self
	// 	iimage := make([]cloudprovider.ICloudImage, 1)
	// 	iimage[0] = image
	// 	if err := cloudprovider.WaitStatus(iimage[0], compute.IMAGE_STATUS_ACTIVE, 15*time.Second, 3600*time.Second); err != nil {
	// 		return nil, err
	// 	}
	// 	return iimage[0], nil
	// }
	return nil, nil
}

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	//return self.downloadImage(userCred, imageId, extId, path)
	return nil, nil
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages("", "PRIVATE_IMAGE", nil, "", len(images), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "GetImages")
		}
		images = append(images, parts...)
		if len(images) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudImage{}
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	parts, _, err := self.region.GetImages("", "", []string{extId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	parts[0].storageCache = self
	return &parts[0], nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return self.uploadImage(ctx, image, callback)
}

func (self *SRegion) getCosUrl(bucket, object string) string {
	//signature := cosauth.NewSignature(self.client.AppID, bucket, self.client.secretId, time.Now().Add(time.Minute*30).String(), time.Now().String(), "yunion", object).SignOnce(self.client.secretKey)
	return fmt.Sprintf("http://%s-%s.cos.%s.myqcloud.com/%s", bucket, self.client.appId, self.Region, object)
}

func (self *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	reader, sizeBytes, err := image.GetReader(image.ImageId, string(qemuimgfmt.QCOW2))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	bucketName := strings.Replace(strings.ToLower(self.region.GetId()+image.ImageId), "-", "", -1)
	if len(bucketName) > 40 {
		bucketName = bucketName[:40]
	}
	exists, _ := self.region.IBucketExist(bucketName)
	if !exists {
		log.Debugf("Bucket %s not exists, to create ...", bucketName)
		err := self.region.CreateIBucket(bucketName, "", "public-read")
		if err != nil {
			log.Errorf("Create bucket error %s", err)
			return "", err
		}
	} else {
		log.Debugf("Bucket %s exists", bucketName)
	}
	defer self.region.DeleteIBucket(bucketName)
	log.Debugf("To upload image to bucket %s ...", bucketName)
	bucket, err := self.region.GetIBucketById(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	body := multicloud.NewProgress(sizeBytes, 80, reader, callback)
	err = cloudprovider.UploadObject(context.Background(), bucket, image.ImageId, 0, body, sizeBytes, "", "", nil, false)
	// err = bucket.PutObject(context.Background(), image.ImageId, reader, sizeBytes, "", "", "")
	if err != nil {
		log.Errorf("UploadObject error %s %s", image.ImageId, err)
		return "", errors.Wrap(err, "bucket.PutObject")
	}

	defer bucket.DeleteObject(context.Background(), image.ImageId)

	// 腾讯云镜像名称需要小于20个字符
	imageBaseName := image.ImageId[:10]
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", image.ImageId[:10])
	}
	imageName := imageBaseName
	nameIdx := 1

	// check image name, avoid name conflict
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
		nameIdx++
	}

	log.Debugf("Import image %s", imageName)
	img, err := self.region.ImportImage(imageName, image.OsArch, image.OsDistribution, image.OsVersion, self.region.getCosUrl(bucketName, image.ImageId))
	if err != nil {
		return "", err
	}
	err = cloudprovider.WaitStatus(img, api.CACHED_IMAGE_STATUS_ACTIVE, 15*time.Second, 3600*time.Second)
	if err != nil {
		return "", err
	}
	if callback != nil {
		callback(100)
	}
	return img.ImageId, nil
}
