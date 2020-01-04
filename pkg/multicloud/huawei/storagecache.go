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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

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

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), imageId)
}

func (self *SStoragecache) fetchImages() error {
	imagesGold, err := self.region.GetImages("", ImageOwnerPublic, "", EnvFusionCompute)
	if err != nil {
		return err
	}

	imagesSelf, err := self.region.GetImages("", ImageOwnerSelf, "", EnvFusionCompute)
	if err != nil {
		return err
	}

	self.iimages = make([]cloudprovider.ICloudImage, len(imagesGold)+len(imagesSelf))
	for i := range imagesGold {
		imagesGold[i].storageCache = self
		self.iimages[i] = &imagesGold[i]
	}
	for i := range imagesSelf {
		imagesSelf[i].storageCache = self
		self.iimages[i+len(imagesGold)] = &imagesSelf[i]
	}
	return nil
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
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

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
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
	image, err := self.region.GetImage(extId)
	image.storageCache = self
	return &image, err
}

func (self *SStoragecache) GetPath() string {
	return ""
}

// 目前支持使用vhd、zvhd、vmdk、qcow2、raw、zvhd2、vhdx、qcow、vdi或qed格式镜像文件创建私有镜像。
// 快速通道功能可快速完成镜像制作，但镜像文件需转换为raw或zvhd2格式并完成镜像优化。
// https://support.huaweicloud.com/api-ims/zh-cn_topic_0083905788.html
func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	if imageId, err := self.region.createIImage(snapshotId, imageName, imageDesc); err != nil {
		return nil, err
	} else if image, err := self.region.GetImage(imageId); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		iimage := make([]cloudprovider.ICloudImage, 1)
		iimage[0] = &image
		if err := cloudprovider.WaitStatus(iimage[0], "avaliable", 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return iimage[0], nil
	}
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId)
}

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(image.ExternalId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", image.ExternalId)

		img, err := self.region.GetImage(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if img.Status == ImageStatusActive && !isForce {
			return image.ExternalId, nil
		}
	} else {
		log.Debugf("UploadImage: no external ID")
	}

	return self.uploadImage(ctx, userCred, image, isForce)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	bucketName := GetBucketName(self.region.GetId(), image.ImageId)

	exist, _ := self.region.IBucketExist(bucketName)
	if !exist {
		err := self.region.CreateIBucket(bucketName, "", "")
		if err != nil {
			return "", errors.Wrap(err, "CreateIBucket")
		}
	}
	defer self.region.DeleteIBucket(bucketName)

	// upload to huawei cloud
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	meta, reader, sizeByte, err := modules.Images.Download(s, image.ImageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", errors.Wrap(err, "Images.Download")
	}
	log.Debugf("Images meta data %s", meta)

	minDiskMB, _ := meta.Int("min_disk")
	minDiskGB := int64(math.Ceil(float64(minDiskMB) / 1024))
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

	err = cloudprovider.UploadObject(context.Background(), bucket, image.ImageId, 0, reader, sizeByte, "", "", nil, false)
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
			if err == cloudprovider.ErrNotFound {
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
	serviceType := self.region.ecsClient.Images.ServiceType()
	err = self.region.waitTaskStatus(serviceType, jobId, TASK_SUCCESS, 15*time.Second, 3600*time.Second)
	if err != nil {
		log.Errorf("waitTaskStatus %s", err)
		return "", err
	}

	// https://support.huaweicloud.com/api-ims/zh-cn_topic_0022473688.html
	return self.region.GetTaskEntityID(serviceType, jobId, "image_id")
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

type SJob struct {
	Status     string            `json:"status"`
	Entities   map[string]string `json:"entities"`
	JobID      string            `json:"job_id"`
	JobType    string            `json:"job_type"`
	BeginTime  string            `json:"begin_time"`
	EndTime    string            `json:"end_time"`
	ErrorCode  string            `json:"error_code"`
	FailReason string            `json:"fail_reason"`
}

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0020092109.html
func (self *SRegion) createIImage(snapshotId, imageName, imageDesc string) (string, error) {
	snapshot, err := self.GetSnapshotById(snapshotId)
	if err != nil {
		return "", err
	}

	disk, err := self.GetDisk(snapshot.VolumeID)
	if err != nil {
		return "", err
	}

	if disk.GetDiskType() != api.DISK_TYPE_SYS {
		return "", fmt.Errorf("disk type err, expected disk type %s", api.DISK_TYPE_SYS)
	}

	if len(disk.Attachments) == 0 {
		return "", fmt.Errorf("disk is not attached.")
	}

	imageObj := jsonutils.NewDict()
	imageObj.Add(jsonutils.NewString(disk.Attachments[0].ServerID), "instance_id")
	imageObj.Add(jsonutils.NewString(imageName), "name")
	imageObj.Add(jsonutils.NewString(imageDesc), "description")

	ret, err := self.ecsClient.Images.PerformAction2("action", "", imageObj, "")
	if err != nil {
		return "", err
	}

	job := SJob{}
	jobId, err := ret.GetString("job_id")
	querys := map[string]string{"service_type": self.ecsClient.Images.ServiceType()}
	err = DoGet(self.ecsClient.Jobs.Get, jobId, querys, &job)
	if err != nil {
		return "", err
	}

	if job.Status == "SUCCESS" {
		imageId, exists := job.Entities["image_id"]
		if exists {
			return imageId, nil
		} else {
			return "", fmt.Errorf("image id not found in create image job %s", job.JobID)
		}
	} else {
		return "", fmt.Errorf("create image failed, %s", job.FailReason)
	}

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
