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
	"io"
	"os"
	"strings"
	"time"

	coslib "github.com/nelsonken/cos-go-sdk-v5/cos"

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

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
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

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	//return self.downloadImage(userCred, imageId, extId, path)
	return nil, nil
}

func (self *SStoragecache) fetchImages() error {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages("", "", nil, "", len(images), 50)
		if err != nil {
			return err
		}
		images = append(images, parts...)
		if len(images) >= total {
			break
		}
	}
	self.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		self.iimages[i] = &images[i]
	}
	return nil
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if self.iimages == nil {
		err := self.fetchImages()
		if err != nil {
			return nil, err
		}
	}
	return self.iimages, nil
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

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(image.ExternalId) > 0 {
		status, err := self.region.GetImageStatus(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		log.Debugf("ctx %p UploadImage: Image external ID %s exists, status %s", ctx, image.ExternalId, status)
		if (status == ImageStatusNormal || status == ImageStatusUsing) && !isForce {
			return image.ExternalId, nil
		}
		log.Debugf("image status: %s isForce: %v", status, isForce)
	} else {
		log.Debugf("ctx %s UploadImage: no external ID", ctx)
	}
	return self.uploadImage(ctx, userCred, image, isForce)
}

func (self *SRegion) getCosUrl(bucket, object string) string {
	//signature := cosauth.NewSignature(self.client.AppID, bucket, self.client.SecretID, time.Now().Add(time.Minute*30).String(), time.Now().String(), "yunion", object).SignOnce(self.client.SecretKey)
	return fmt.Sprintf("http://%s-%s.cos.%s.myqcloud.com/%s", bucket, self.client.AppID, self.Region, object)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	// first upload image to oss
	s := auth.GetAdminSession(ctx, options.Options.Region, "")

	meta, reader, err := modules.Images.Download(s, image.ImageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", err
	}

	tmpFile := fmt.Sprintf("%s/%s", options.Options.TempPath, image.ImageId)
	defer os.Remove(tmpFile)
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		return "", err
	}

	log.Infof("meta data %s", meta)
	cos, err := self.region.GetCosClient()
	if err != nil {
		log.Errorf("GetOssClient err %s", err)
		return "", err
	}
	bucketName := strings.ReplaceAll(strings.ToLower(self.region.GetId()+image.ImageId), "-", "")
	if len(bucketName) > 40 {
		bucketName = bucketName[:40]
	}
	err = cos.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Debugf("Bucket %s not exists, to create ...", bucketName)
		err := cos.CreateBucket(context.Background(), bucketName, &coslib.AccessControl{ACL: "public-read"})
		if err != nil {
			log.Errorf("Create bucket error %s", err)
			return "", err
		}
	} else {
		log.Debugf("Bucket %s exists", bucketName)
	}
	log.Debugf("To upload image to bucket %s ...", bucketName)
	err = cos.Bucket(bucketName).UploadObjectBySlice(context.Background(), image.ImageId, tmpFile, 3, map[string]string{})
	if err != nil {
		log.Errorf("UploadObject error %s %s", image.ImageId, err)
		return "", err
	}

	defer cos.Bucket(bucketName).DeleteObject(context.Background(), image.ImageId)

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
			if err == cloudprovider.ErrNotFound {
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
	err = cloudprovider.WaitStatus(img, api.CACHED_IMAGE_STATUS_READY, 15*time.Second, 3600*time.Second)
	if err != nil {
		return "", err
	}
	return img.ImageId, nil
}
