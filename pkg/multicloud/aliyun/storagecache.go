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

package aliyun

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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

func (self *SStoragecache) fetchImages() error {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages(ImageStatusType(""), "", nil, "", len(images), 50)
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

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {

	if len(image.ExternalId) > 0 {
		status, err := self.region.GetImageStatus(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		log.Debugf("UploadImage: Image external ID %s exists, status %s", image.ExternalId, status)
		if status == ImageStatusAvailable && !isForce {
			return image.ExternalId, nil
		}
		// 不能直接删除 ImageStatusCreating 状态的image ,需要先取消importImage Task
		if status == ImageStatusCreating {
			err := self.region.CancelImageImportTasks()
			if err != nil {
				log.Errorln(err)
			}
		}
		if len(status) > 0 {
			err = self.region.DeleteImage(image.ExternalId)
			if err != nil {
				log.Errorf("failed to delete image %s(%s) error: %v", image.ExternalId, status, err)
			}
		}
	} else {
		log.Debugf("UploadImage: no external ID")
	}

	return self.uploadImage(ctx, userCred, image, isForce)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	// first upload image to oss
	s := auth.GetAdminSession(ctx, options.Options.Region, "")

	meta, reader, sizeByte, err := modules.Images.Download(s, image.ImageId, string(qemuimg.QCOW2), false)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)

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
	err = cloudprovider.UploadObject(context.Background(), bucket, image.ImageId, 0, reader, sizeByte, "", "", "", false)
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
			if err == cloudprovider.ErrNotFound {
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

func (self *SRegion) CheckBucket(bucketName string) (*oss.Bucket, error) {
	return self.checkBucket(bucketName)
}

func (self *SRegion) checkBucket(bucketName string) (*oss.Bucket, error) {
	oss, err := self.GetOssClient()
	if err != nil {
		log.Errorf("GetOssClient err %s", err)
		return nil, err
	}
	if exist, err := oss.IsBucketExist(bucketName); err != nil {
		log.Errorf("IsBucketExist err %s", err)
		return nil, err
	} else if !exist {
		log.Debugf("Bucket %s not exists, to create ...", bucketName)
		if err := oss.CreateBucket(bucketName); err != nil {
			log.Errorf("Create bucket error %s", err)
			return nil, err
		}
	}
	log.Debugf("Bucket %s exists", bucketName)
	if bucket, err := oss.Bucket(bucketName); err != nil {
		log.Errorf("Bucket error %s %s", bucketName, err)
		return nil, err
	} else {
		return bucket, nil
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

	if _, err := self.checkBucket(params["OssBucket"]); err != nil {
		return "", err
	}

	if body, err := self.ecsRequest("CreateImage", params); err != nil {
		log.Errorf("CreateImage fail %s", err)
		return "", err
	} else {
		log.Infof("%s", body)
		return body.GetString("ImageId")
	}
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId, path)
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

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	err := self.region.GetClient().EnableImageExport()
	if err != nil {
		log.Errorf("fail to enable export privileges: %s", err)
		return nil, err
	}

	tmpImageFile, err := ioutil.TempFile(path, extId)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpImageFile.Name())
	bucketName := strings.ToLower(fmt.Sprintf("imgcache-%s", self.region.GetId()))
	if bucket, err := self.region.checkBucket(bucketName); err != nil {
		return nil, err
	} else if _, err := self.region.GetImage(extId); err != nil {
		return nil, err
	} else if task, err := self.region.ExportImage(extId, bucket); err != nil {
		return nil, err
	} else if err := self.region.waitTaskStatus(ExportImageTask, task.TaskId, TaskStatusFinished, 15*time.Second, 3600*time.Second); err != nil {
		return nil, err
	} else if imageList, err := bucket.ListObjects(oss.Prefix(fmt.Sprintf("%sexport", strings.Replace(extId, "-", "", -1)))); err != nil {
		return nil, err
	} else if len(imageList.Objects) != 1 {
		return nil, fmt.Errorf("exported image not find")
	} else if err := bucket.DownloadFile(imageList.Objects[0].Key, tmpImageFile.Name(), 12*1024*1024, oss.Routines(3), oss.Progress(&OssProgressListener{})); err != nil {
		return nil, err
	} else {
		s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
		params := jsonutils.Marshal(map[string]string{"image_id": imageId, "disk-format": "raw"})
		if file, err := os.Open(tmpImageFile.Name()); err != nil {
			return nil, err
		} else if result, err := modules.Images.Upload(s, params, file, imageList.Objects[0].Size); err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}
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
