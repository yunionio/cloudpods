package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "pending"
	ImageStatusAvailable    ImageStatusType = "available"
	ImageStatusCreateFailed ImageStatusType = "failed"
)

type ImageOwnerType string

const (
	ImageOwnerSystem      ImageOwnerType = "system"
	ImageOwnerSelf        ImageOwnerType = "self"
	ImageOwnerOthers      ImageOwnerType = "others"
	ImageOwnerMarketplace ImageOwnerType = "marketplace"
)

type ImageImportTask struct {
	ImageId  string
	RegionId string
	// RequestId string
	TaskId string
}

type SImage struct {
	storageCache *SStoragecache

	Architecture         string
	CreationTime         time.Time
	Description          string
	ImageId              string
	ImageName            string
	OSName               string
	OSType               string
	IsSupportCloudinit   bool
	IsSupportIoOptimized bool
	Platform             string
	Size                 int
	Status               ImageStatusType
	Usage                string
}

func (self *SImage) GetId() string {
	return self.ImageId
}

func (self *SImage) GetName() string {
	return self.ImageName
}

func (self *SImage) GetGlobalId() string {
	return self.ImageId
}

func (self *SImage) GetStatus() string {
	switch self.Status {
	case ImageStatusCreating:
		return models.IMAGE_STATUS_QUEUED
	case ImageStatusAvailable:
		return models.IMAGE_STATUS_ACTIVE
	case ImageStatusCreateFailed:
		return models.IMAGE_STATUS_KILLED
	default:
		return models.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	// todo: GetImage
	new, err := self.storageCache.region.GetImage(self.ImageId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	if len(self.Architecture) > 0 {
		data.Add(jsonutils.NewString(self.Architecture), "os_arch")
	}
	if len(self.OSType) > 0 {
		data.Add(jsonutils.NewString(self.OSType), "os_name")
	}
	if len(self.Platform) > 0 {
		data.Add(jsonutils.NewString(self.Platform), "os_distribution")
	}
	if len(self.OSName) > 0 {
		data.Add(jsonutils.NewString(self.OSName), "os_version")
	}
	return data
}

func (self *SImage) Delete() error {
	// todo: implement meßß
	return self.storageCache.region.DeleteImage(self.ImageId)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) ImportImage(name string, osArch string, osType string, osDist string, bucket string, key string) (*ImageImportTask, error) {
	return nil, nil
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	return nil, nil
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	return nil, nil
}

func (self *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	return "",nil
}

func (self *SRegion) GetImages(status ImageStatusType, owner ImageOwnerType, imageId []string, name string, offset int, limit int) ([]SImage, int, error) {
	return nil,0, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	return nil
}