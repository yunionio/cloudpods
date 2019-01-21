package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"time"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/timeutils"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "pending"
	ImageStatusAvailable    ImageStatusType = "available"
	ImageStatusCreateFailed ImageStatusType = "failed"
)

var (
	ImageOwnerAmazone     = "amazon"
	ImageOwnerSelf        = "self"
	ImageOwnerMicrosoft   = "microsoft"
	ImageOwnerMarketplace = "aws-marketplace"
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

	Architecture string
	CreationTime time.Time
	Description  string
	ImageId      string
	ImageName    string
	OSName       string
	OSType       string
	ImageType    string
	// IsSupportCloudinit   bool
	IsSupportIoOptimized bool
	Platform             string
	SizeGB               int
	Status               ImageStatusType
	OwnerType            string
	// Usage                string
	RootDevice RootDevice
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
		return models.CACHED_IMAGE_STATUS_CACHING
	case ImageStatusAvailable:
		return models.CACHED_IMAGE_STATUS_READY
	case ImageStatusCreateFailed:
		return models.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return models.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.Status {
	case ImageStatusCreating:
		return cloudprovider.IMAGE_STATUS_QUEUED
	case ImageStatusAvailable:
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case ImageStatusCreateFailed:
		return cloudprovider.IMAGE_STATUS_KILLED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	new, err := self.storageCache.region.GetImage(self.ImageId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SImage) GetImageType() string {
	switch self.OwnerType {
	case ImageOwnerSelf:
		return cloudprovider.CachedImageTypeCustomized
	case ImageOwnerAmazone:
		return cloudprovider.CachedImageTypeSystem
	case ImageOwnerMicrosoft:
		return cloudprovider.CachedImageTypeShared
	case ImageOwnerMarketplace:
		return cloudprovider.CachedImageTypeMarket
	default:
		return cloudprovider.CachedImageTypeShared
	}
}

func (self *SImage) GetSize() int64 {
	return int64(self.SizeGB) * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() string {
	return self.OSType
}

func (self *SImage) GetOsArch() string {
	return self.Architecture
}

func (self *SImage) GetOsDist() string {
	return self.Platform
}

func (self *SImage) GetOsVersion() string {
	return self.OSName
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return 10
}

func (self *SImage) GetImageFormat() string {
	return "vhd"
}

func (self *SImage) GetCreateTime() time.Time {
	return self.CreationTime
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
	if len(imageId) == 0 {
		return nil, fmt.Errorf("GetImage image id should not be empty")
	}

	images, err := self.GetImages("", nil, []string{imageId}, "")
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("image %s not found", imageId)
	}
	return &images[0], nil
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("image name should not be empty")
	}

	images, err := self.GetImages("", nil, nil, name)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	log.Debugf("%d image found match name %s", len(images), name)
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

func (self *SRegion) GetImages(status ImageStatusType, owners []*string, imageId []string, name string) ([]SImage, error) {
	params := &ec2.DescribeImagesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(status) > 0 {
		filters = AppendSingleValueFilter(filters, "state", string(status))
	}

	if len(name) > 0 {
		filters = AppendSingleValueFilter(filters, "name", name)
	}

	if len(owners) > 0 {
		params.SetOwners(owners)
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
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	}

	images := []SImage{}
	for _, image := range ret.Images {
		if err := FillZero(image); err != nil {
			return nil, err
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

		createTime, _ := timeutils.ParseTimeStr(*image.CreationDate)

		images = append(images, SImage{
			storageCache:         self.getStoragecache(),
			Architecture:         *image.Architecture,
			Description:          *image.Description,
			ImageId:              *image.ImageId,
			ImageName:            tagspec.GetNameTag(),
			ImageType:            *image.ImageType,
			OwnerType:            *image.ImageOwnerAlias,
			IsSupportIoOptimized: *image.EnaSupport,
			Platform:             *image.Platform,
			Status:               ImageStatusType(*image.State),
			CreationTime:         createTime,
			SizeGB:               size,
			RootDevice:           rootDevice,
			OSType:               osType,
		})
	}

	return images, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	params := &ec2.DeregisterImageInput{}
	params.SetImageId(imageId)
	_, err := self.ec2Client.DeregisterImage(params)
	return err
}
