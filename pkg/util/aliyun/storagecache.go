package aliyun

import (
	"fmt"
	"os"
	"strings"
	"time"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	compute "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, extId string, isForce bool) (string, error) {

	if len(extId) > 0 {
		status, err := self.region.GetImageStatus(extId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if status == ImageStatusAvailable && !isForce {
			return extId, nil
		}
	}

	return self.uploadImage(userCred, imageId, osArch, osType, osDist, isForce)
}

func (self *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, isForce bool) (string, error) {
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
	bucketName := strings.ToLower(fmt.Sprintf("imgcache-%s-%s", self.region.GetId(), self.region.client.providerId))
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

	task, err := self.region.ImportImage(imageName, osArch, osType, osDist, bucketName, imageId)

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

func (self *SStoragecache) CreateIImage(snapshoutId, imageName, imageDesc string) (cloudprovider.ICloudImage, error) {
	if imageId, err := self.region.createIImage(snapshoutId, imageName, imageDesc); err != nil {
		return nil, err
	} else if image, err := self.region.GetImage(imageId); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		iimage := make([]cloudprovider.ICloudImage, 1)
		iimage[0] = image
		if err := cloudprovider.WaitStatus(iimage[0], compute.IMAGE_STATUS_ACTIVE, 15*time.Second, 3600*time.Second); err != nil {
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

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId)
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

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	tmpImageFile := fmt.Sprintf("/tmp/%s", extId)
	bucketName := strings.ToLower(fmt.Sprintf("imgcache-%s", self.region.GetId()))
	if bucket, err := self.region.checkBucket(bucketName); err != nil {
		return nil, err
	} else if _, err := self.region.GetImage(extId); err != nil {
		return nil, err
	} else if task, err := self.region.ExportImage(extId, bucket); err != nil {
		return nil, err
	} else if err := self.region.waitTaskStatus(ExportImageTask, task.TaskId, "Finished", 15*time.Second, 3600*time.Second); err != nil {
		return nil, err
	} else if imageList, err := bucket.ListObjects(oss.Prefix(fmt.Sprintf("%sexport", strings.Replace(extId, "-", "", -1)))); err != nil {
		return nil, err
	} else if len(imageList.Objects) != 1 {
		return nil, httperrors.NewResourceNotFoundError("exported image not find")
	} else if err := bucket.DownloadFile(imageList.Objects[0].Key, tmpImageFile, 12*1024*1024, oss.Routines(3), oss.Progress(&OssProgressListener{})); err != nil {
		return nil, err
	} else {
		s := auth.GetAdminSession(options.Options.Region, "")
		params := jsonutils.Marshal(map[string]string{"image_id": imageId, "disk-format": "raw"})
		if file, err := os.Open(tmpImageFile); err != nil {
			return nil, err
		} else if result, err := modules.Images.Upload(s, params, file, imageList.Objects[0].Size); err != nil {
			return nil, err
		} else {
			os.Remove(tmpImageFile)
			return result, nil
		}
	}
}
