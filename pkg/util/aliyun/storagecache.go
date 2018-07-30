package aliyun

import (
	"fmt"
	"strings"
	"time"

	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/auth"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/options"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
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

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SStoragecache) fetchImages() error {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages(ImageStatusType(""), ImageOwnerSelf, nil, "", len(images), 50)
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

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		status, _ := self.region.GetImageStatus(extId)
		if status == ImageStatusAvailable && !isForce {
			return extId, nil
		}
	}
	return self.uploadImage(userCred, imageId, isForce)
}

func (self *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, isForce bool) (string, error) {
	// first upload image to oss
	s := auth.GetAdminSession(options.Options.Region, "")

	meta, reader, err := modules.Images.Download(s, imageId)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)
	oss, err := self.region.GetOssClient()
	if err != nil {
		log.Errorf("GetOssClient err %s", err)
		return "", err
	}
	bucketName := strings.ToLower(fmt.Sprintf("imgcache-%s", self.region.GetId()))
	exist, err := oss.IsBucketExist(bucketName)
	if err != nil {
		log.Errorf("IsBucketExist err %s", err)
		return "", err
	}
	if !exist {
		log.Debugf("Bucket %s not exists, to create ...", bucketName)
		err = oss.CreateBucket(bucketName)
		if err != nil {
			log.Errorf("Create bucket error %s", err)
			return "", err
		}
	} else {
		log.Debugf("Bucket %s exists", bucketName)
	}
	bucket, err := oss.Bucket(bucketName)
	if err != nil {
		log.Errorf("Bucket error %s %s", bucketName, err)
		return "", err
	}
	log.Debugf("To upload image to bucket %s ...", bucketName)
	err = bucket.PutObject(imageId, reader)
	if err != nil {
		log.Errorf("PutObject error %s %s", imageId, err)
		return "", err
	}

	imageBaseName := imageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", imageId)
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

	task, err := self.region.ImportImage(imageName, bucketName, imageId)

	if err != nil {
		log.Errorf("ImportImage error %s %s %s", imageId, bucketName, err)
		return "", err
	}

	// timeout: 1hour = 3600 seconds
	err = self.region.waitTaskStatus(ImportImageTask, task.TaskId, "Finished", 15*time.Second, 3600*time.Second)
	if err != nil {
		log.Errorf("waitTaskStatus %s", err)
		return task.ImageId, err
	}

	return task.ImageId, nil
}
