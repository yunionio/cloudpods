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

package apsara

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	ApsaraTags
	region *SRegion

	iimages []cloudprovider.ICloudImage
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

func (self *SStoragecache) getImages(owner ImageOwnerType) ([]SImage, error) {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages(ImageStatusType(""), owner, nil, "", len(images), 50)
		if err != nil {
			return nil, err
		}
		images = append(images, parts...)
		if len(images) >= total {
			break
		}
	}
	return images, nil
}

func (self *SStoragecache) fetchImages() error {
	images := make([]SImage, 0)
	for _, owner := range []ImageOwnerType{ImageOwnerSelf, ImageOwnerSystem} {
		_images, err := self.getImages(owner)
		if err != nil {
			return errors.Wrapf(err, "GetImage(%s)", owner)
		}
		images = append(images, _images...)
	}
	self.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		self.iimages[i] = &images[i]
	}
	return nil
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	if self.iimages == nil {
		err := self.fetchImages()
		if err != nil {
			return nil, err
		}
	}
	return self.iimages, nil
}

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	img, err := self.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	img.storageCache = self
	return img, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {

	if len(image.ExternalId) > 0 {
		status, err := self.region.GetImageStatus(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		log.Debugf("UploadImage: Image external ID %s exists, status %s", image.ExternalId, status)
		// 不能直接删除 ImageStatusCreating 状态的image ,需要先取消importImage Task
		if status == ImageStatusCreating {
			err := self.region.CancelImageImportTasks()
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	return self.uploadImage(ctx, image, callback)
}

func (self *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	reader, sizeByte, err := image.GetReader(image.ImageId, string(qemuimgfmt.QCOW2))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	bucketName := strings.ToLower(fmt.Sprintf("imgcache-%s-%s", self.region.GetId(), image.ImageId))
	exist, err := self.region.IBucketExist(bucketName)
	if err != nil {
		log.Errorf("IsBucketExist err %s", err)
		return "", err
	}
	if !exist {
		log.Debugf("Bucket %s not exists, to create ...", bucketName)
		err = self.region.CreateIBucket(bucketName, "", "")
		if err != nil {
			log.Errorf("Create bucket error %s", err)
			return "", err
		}
	} else {
		log.Debugf("Bucket %s exists", bucketName)
	}

	defer self.region.DeleteIBucket(bucketName) // remove bucket

	bucket, err := self.region.GetIBucketByName(bucketName)
	if err != nil {
		log.Errorf("Bucket error %s %s", bucketName, err)
		return "", err
	}
	log.Debugf("To upload image to bucket %s ...", bucketName)
	body := multicloud.NewProgress(sizeByte, 80, reader, callback)
	err = cloudprovider.UploadObject(context.Background(), bucket, image.ImageId, 0, body, sizeByte, "", "", nil, false)
	// err = bucket.PutObject(image.ImageId, reader)
	if err != nil {
		log.Errorf("PutObject error %s %s", image.ImageId, err)
		return "", err
	}

	defer bucket.DeleteObject(context.Background(), image.ImageId) // remove object

	imageBaseName := image.ImageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", image.ImageId)
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
		nameIdx += 1
	}

	log.Debugf("Import image %s", imageName)

	// ensure privileges
	err = self.region.GetClient().EnableImageImport()
	if err != nil {
		log.Errorf("fail to enable import privileges: %s", err)
		return "", err
	}

	task, err := self.region.ImportImage(imageName, image.OsArch, image.OsType, image.OsDistribution, bucketName, image.ImageId)

	if err != nil {
		log.Errorf("ImportImage error %s %s %s", image.ImageId, bucketName, err)
		return "", err
	}

	// timeout: 1hour = 3600 seconds
	err = self.region.waitTaskStatus(ImportImageTask, task.TaskId, TaskStatusFinished, 15*time.Second, 3600*time.Second)
	if err != nil {
		log.Errorf("waitTaskStatus %s", err)
		return task.ImageId, err
	}

	if callback != nil {
		callback(100)
	}

	return task.ImageId, nil
}

func (self *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	if imageId, err := self.region.createIImage(snapshoutId, imageName, imageDesc); err != nil {
		return nil, err
	} else if image, err := self.region.GetImage(imageId); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		iimage := make([]cloudprovider.ICloudImage, 1)
		iimage[0] = image
		if err := cloudprovider.WaitStatus(iimage[0], cloudprovider.IMAGE_STATUS_ACTIVE, 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return iimage[0], nil
	}
}

func (self *SRegion) CreateImage(snapshoutId, imageName, imageDesc string) (string, error) {
	return self.createIImage(snapshoutId, imageName, imageDesc)
}

func (self *SRegion) createIImage(snapshoutId, imageName, imageDesc string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["OssBucket"] = strings.ToLower(fmt.Sprintf("imgcache-%s", self.GetId()))
	params["SnapshotId"] = snapshoutId
	params["ImageName"] = imageName
	params["Description"] = imageDesc

	if body, err := self.ecsRequest("CreateImage", params); err != nil {
		log.Errorf("CreateImage fail %s", err)
		return "", err
	} else {
		log.Infof("%s", body)
		return body.GetString("ImageId")
	}
}

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(imageId, extId, path)
}

// 定义进度条监听器。
type OssProgressListener struct {
}

// 定义进度变更事件处理函数。
func (listener *OssProgressListener) ProgressChanged(event *oss.ProgressEvent) {
	switch event.EventType {
	case oss.TransferStartedEvent:
		log.Debugf("Transfer Started, ConsumedBytes: %d, TotalBytes %d.\n",
			event.ConsumedBytes, event.TotalBytes)
	case oss.TransferDataEvent:
		log.Debugf("\rTransfer Data, ConsumedBytes: %d, TotalBytes %d, %d%%.",
			event.ConsumedBytes, event.TotalBytes, event.ConsumedBytes*100/event.TotalBytes)
	case oss.TransferCompletedEvent:
		log.Debugf("\nTransfer Completed, ConsumedBytes: %d, TotalBytes %d.\n",
			event.ConsumedBytes, event.TotalBytes)
	case oss.TransferFailedEvent:
		log.Debugf("\nTransfer Failed, ConsumedBytes: %d, TotalBytes %d.\n",
			event.ConsumedBytes, event.TotalBytes)
	default:
	}
}

func (self *SStoragecache) downloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
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
