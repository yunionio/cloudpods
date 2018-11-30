package aws

import (
	"fmt"
	"strings"

	"context"
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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
	ImageOwnerSystem      ImageOwnerType = "amazon"
	ImageOwnerSelf        ImageOwnerType = "self"
	ImageOwnerOthers      ImageOwnerType = "microsoft"
	ImageOwnerMarketplace ImageOwnerType = "aws-marketplace"
)

type ImageImportTask struct {
	ImageId  string
	RegionId string
	TaskId   string
}

type RootDevice struct {
	SnapshotId string
	Size       int    // GB
	Category   string // VolumeType
}

type SImage struct {
	storageCache *SStoragecache

	Architecture         string
	CreationTime         string
	Description          string
	ImageId              string
	ImageName            string
	OSName               string
	OSType               string
	ImageType            string
	IsSupportCloudinit   bool
	IsSupportIoOptimized bool
	Platform             string
	Size                 int
	Status               ImageStatusType
	Usage                string
	RootDevice           RootDevice
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

func (self *SImage) Delete(ctx context.Context) error {
	// todo: implement me
	return self.storageCache.region.DeleteImage(self.ImageId)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) ImportImage(name string, osArch string, osType string, osDist string, diskFormat string, bucket string, key string) (*ImageImportTask, error) {
	params := &ec2.ImportImageInput{}
	params.SetArchitecture(osArch)
	params.SetHypervisor("xen") // todo: osType?
	params.SetPlatform(osType)  // Linux|Windows
	// https://docs.aws.amazon.com/zh_cn/vm-import/latest/userguide/vmimport-image-import.html#import-vm-image
	params.SetRoleName("vmimport")
	container := &ec2.ImageDiskContainer{}
	container.SetDescription(fmt.Sprintf("vmimport %s - %s", name, osDist))
	container.SetFormat(diskFormat)
	container.SetDeviceName("/dev/sda") // default /dev/sda
	bkt := &ec2.UserBucket{S3Bucket: &bucket, S3Key: &key}
	container.SetUserBucket(bkt)
	params.SetDiskContainers([]*ec2.ImageDiskContainer{container})
	params.SetLicenseType("BYOL") // todo: AWS?
	ret, err := self.ec2Client.ImportImage(params)
	if err != nil {
		return nil, err
	}
	log.Debugf("ImportImage task: %s", ret.String())
	return &ImageImportTask{ImageId: StrVal(ret.ImageId), RegionId: self.RegionId, TaskId: *ret.ImportTaskId}, nil
}

type ImageExportTask struct {
	ImageId  string
	RegionId string
	TaskId   string
}

func (self *SRegion) ExportImage(instanceId string, imageId string) (*ImageExportTask, error) {
	params := &ec2.CreateInstanceExportTaskInput{}
	params.SetInstanceId(instanceId)
	params.SetDescription(fmt.Sprintf("image %s export from aws", imageId))
	params.SetTargetEnvironment("vmware")
	spec := &ec2.ExportToS3TaskSpecification{}
	spec.SetContainerFormat("ova")
	spec.SetDiskImageFormat("RAW")
	spec.SetS3Bucket("imgcache-onecloud")
	params.SetExportToS3Task(spec)
	ret, err := self.ec2Client.CreateInstanceExportTask(params)
	if err != nil {
		return nil, err
	}

	return &ImageExportTask{ImageId: imageId, RegionId: self.RegionId, TaskId: *ret.ExportTask.ExportTaskId}, nil
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

	log.Debugf("%d image found match name %", len(images), name)
	return &images[0], nil
}

func (self *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	image, err := self.GetImage(imageId)
	if err != nil {
		return "", err
	}
	return image.Status, nil
}

func getRootDiskSize(image *ec2.Image) (int, error) {
	rootDeivce := *image.RootDeviceName
	for _, volume := range image.BlockDeviceMappings {
		if len(rootDeivce) > 0 && *volume.DeviceName == rootDeivce {
			return int(*volume.Ebs.VolumeSize), nil
		}
	}

	return 0, fmt.Errorf("image size not found: %s", image.String())
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

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	ret, err := self.ec2Client.DescribeImages(params)
	if err != nil {
		if strings.Contains(err.Error(), ".NotFound") {
			return nil, 0, cloudprovider.ErrNotFound
		}
		return nil, 0, err
	}

	images := []SImage{}
	for _, image := range ret.Images {
		if err := FillZero(image); err != nil {
			return nil, 0, err
		}

		tagspec := TagSpec{}
		tagspec.LoadingEc2Tags(image.Tags)

		size, err := getRootDiskSize(image)
		if err != nil {
			log.Debugf(err.Error())
		}

		var rootDevice RootDevice
		for _, block := range image.BlockDeviceMappings {
			if len(*image.RootDeviceName) > 0 && *block.DeviceName == *image.RootDeviceName {
				rootDevice.SnapshotId = *block.Ebs.SnapshotId
				rootDevice.Category = *block.Ebs.VolumeType
				rootDevice.Size = int(*block.Ebs.VolumeSize)
			}
		}

		osType := ""
		if StrVal(image.Platform) != "windows" {
			osType = "Linux"
		} else {
			osType = "Windows"
		}

		images = append(images, SImage{
			storageCache:         self.getStoragecache(),
			Architecture:         *image.Architecture,
			Description:          *image.Description,
			ImageId:              *image.ImageId,
			ImageName:            tagspec.GetNameTag(),
			ImageType:            *image.ImageType,
			IsSupportIoOptimized: *image.EnaSupport,
			Platform:             *image.Platform,
			Status:               ImageStatusType(*image.State),
			CreationTime:         *image.CreationDate,
			Size:                 size,
			RootDevice:           rootDevice,
			OSType:               osType,
			// Usage:                "",
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
