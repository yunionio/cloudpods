package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "pending"
	ImageStatusAvailable    ImageStatusType = "available"
	ImageStatusCreateFailed ImageStatusType = "failed"
)

type ImageOwnerType string

const (
	ImageOwnerSystem      ImageOwnerType = "amazon"
	ImageOwnerSelf        ImageOwnerType = "self"
	ImageOwnerOthers      ImageOwnerType = "microsoft"
	ImageOwnerMarketplace ImageOwnerType = "aws-marketplace"
)

type ImageImportTask struct {
	ImageId  string
	RegionId string
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
	// todo: implement me
	return self.storageCache.region.DeleteImage(self.ImageId)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) ImportImage(name string, osArch string, osType string, osDist string, bucket string, key string) (*ImageImportTask, error) {
	params := &ec2.ImportImageInput{}
	params.SetArchitecture(osArch)
	params.SetHypervisor(osType) // todo: osType?
	params.SetPlatform(osDist)
	// https://docs.aws.amazon.com/zh_cn/vm-import/latest/userguide/vmimport-image-import.html#import-vm-image
	params.SetRoleName("vmimport")
	container := &ec2.ImageDiskContainer{}
	container.SetDescription(fmt.Sprintf("vmimport %s", name))
	container.SetFormat(osType)
	container.SetDeviceName("/dev/sda") // default /dev/sda
	bkt := &ec2.UserBucket{S3Bucket: &bucket, S3Key: &key}
	container.SetUserBucket(bkt)
	params.SetDiskContainers([]*ec2.ImageDiskContainer{container})
	params.SetLicenseType("AWS")  // todo: AWS?
	ret, err := self.ec2Client.ImportImage(params)
	if err != nil {
		return nil, err
	}

	return &ImageImportTask{ImageId: *ret.ImageId, RegionId: self.RegionId, TaskId: *ret.ImportTaskId}, nil
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	images, _, err := self.GetImages("", ImageOwnerSelf, []string{imageId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("image %s not found", imageId)
	}
	return &images[0], nil
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	images, _, err := self.GetImages("", ImageOwnerSelf, nil, name, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &images[0], nil
}

func (self *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	image, err := self.GetImage(imageId)
	if err != nil {
		return "", err
	}
	return image.Status, nil
}

func (self *SRegion) GetImages(status ImageStatusType, owner ImageOwnerType, imageId []string, name string, offset int, limit int) ([]SImage, int, error) {
	params := &ec2.DescribeImagesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(status) > 0 {
		filters = AppendSingleValueFilter(filters, "state", string(status))
	}

	if len(name) > 0 {
		filters = AppendSingleValueFilter(filters, "name", name)
	}

	if len(owner) > 0 {
		own := string(owner)
		params.SetOwners([]*string{&own})
	}

	if len(imageId) > 0 {
		params.SetImageIds(ConvertedList(imageId))
	}

	ret, err := self.ec2Client.DescribeImages(params)
	if err != nil {
		return nil, 0, err
	}

	images := make([]SImage, len(ret.Images))
	for _, image := range ret.Images {
		images = append(images, SImage{
			storageCache:         self.getStoragecache(),
			Architecture:         *image.Architecture,
			Description:          *image.Description,
			ImageId:              *image.ImageId,
			ImageName:            *image.ImageId,
			// OSName:               *image.Platform,
			OSType:               *image.ImageType,
			IsSupportIoOptimized: *image.EnaSupport,
			// Platform:             *image.Platform,
			Status:               ImageStatusCreating, // *image.State,
			// Usage:                "",
			// Size:                 .,
			// CreationTime:         *image.CreationDate,
		})
	}

	return images, len(images), nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	params := &ec2.DeregisterImageInput{}
	params.SetImageId(imageId)
	_, err := self.ec2Client.DeregisterImage(params)
	return err
}