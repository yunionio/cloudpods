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

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "pending"
	ImageStatusAvailable    ImageStatusType = "available"
	ImageStatusCreateFailed ImageStatusType = "failed"

	ImageImportStatusCompleted   = "completed"
	ImageImportStatusUncompleted = "uncompleted"
	ImageImportStatusError       = "error"
)

type TImageOwnerType string

const (
	ImageOwnerTypeSystem = TImageOwnerType("system")
	ImageOwnerTypeSelf   = TImageOwnerType("self")
	ImageOwnerTypeOther  = TImageOwnerType("other")
)

var (
	ImageOwnerAll        = []TImageOwnerType(nil)
	ImageOwnerSelf       = []TImageOwnerType{ImageOwnerTypeSelf}
	ImageOwnerSystem     = []TImageOwnerType{ImageOwnerTypeSystem}
	ImageOwnerSelfSystem = []TImageOwnerType{ImageOwnerTypeSystem, ImageOwnerTypeSelf}
)

type ImageImportTask struct {
	region *SRegion

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
	OSType       string
	ImageType    string
	// IsSupportCloudinit   bool
	EnaSupport bool
	Platform   string
	SizeGB     int
	Status     ImageStatusType
	OwnerType  string
	// Usage                string
	RootDevice RootDevice

	Public             bool
	Hypervisor         string
	VirtualizationType string
	OwnerId            string

	ProductCodes []*ec2.ProductCode

	OSVersion string
	OSDist    string
	OSBuildId string
}

func (self *ImageImportTask) GetId() string {
	return self.TaskId
}

func (self *ImageImportTask) GetName() string {
	return self.GetId()
}

func (self *ImageImportTask) GetGlobalId() string {
	return self.GetId()
}

func (self *ImageImportTask) Refresh() error {
	return nil
}

func (self *ImageImportTask) IsEmulated() bool {
	return true
}

func (self *ImageImportTask) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *ImageImportTask) GetStatus() string {
	ret, err := self.region.ec2Client.DescribeImportImageTasks(&ec2.DescribeImportImageTasksInput{ImportTaskIds: []*string{&self.TaskId}})
	if err != nil {
		log.Errorf("DescribeImportImageTasks %s", err)
		return ImageImportStatusError
	}

	err = FillZero(ret)
	if err != nil {
		log.Errorf("DescribeImportImageTask.FillZero %s", err)
		return ImageImportStatusError
	}

	// 打印上传进度
	log.Debugf("DescribeImportImage Task %s", ret.String())
	for _, item := range ret.ImportImageTasks {
		if *item.Status == "completed" {
			return ImageImportStatusCompleted
		} else {
			return ImageImportStatusUncompleted
		}
	}

	return ImageImportStatusUncompleted
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return self.ImageId
}

func (self *SImage) GetName() string {
	if len(self.ImageName) > 0 {
		return self.ImageName
	}

	return self.GetId()
}

func (self *SImage) GetGlobalId() string {
	return self.ImageId
}

func (self *SImage) GetStatus() string {
	switch self.Status {
	case ImageStatusCreating:
		return api.CACHED_IMAGE_STATUS_CACHING
	case ImageStatusAvailable:
		return api.CACHED_IMAGE_STATUS_READY
	case ImageStatusCreateFailed:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
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
	return self.ImageType
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.SizeGB) * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() string {
	return self.OSType
}

func (self *SImage) GetOsArch() string {
	return self.Architecture
}

func (self *SImage) GetOsDist() string {
	return self.OSDist
}

func (self *SImage) GetOsVersion() string {
	return self.OSVersion
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return self.SizeGB
}

func (self *SImage) GetImageFormat() string {
	return "vhd"
}

func (self *SImage) GetCreatedAt() time.Time {
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
	if len(self.OSDist) > 0 {
		data.Add(jsonutils.NewString(self.OSDist), "os_distribution")
	}
	if len(self.OSVersion) > 0 {
		data.Add(jsonutils.NewString(self.OSVersion), "os_version")
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

	images, err := self.getImages("", ImageOwnerAll, []string{imageId}, "", "", nil, "")
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, ErrorNotFound()
	}
	return &images[0], nil
}

func (self *SRegion) GetImageByName(name string, owners []TImageOwnerType) (*SImage, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("image name should not be empty")
	}

	images, err := self.getImages("", owners, nil, name, "hvm", nil, "")
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, ErrorNotFound()
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
		if len(rootDeivce) > 0 && *volume.DeviceName == rootDeivce && volume.Ebs != nil && volume.Ebs.VolumeSize != nil {
			return int(*volume.Ebs.VolumeSize), nil
		}
	}
	return 0, fmt.Errorf("image size not found: %s", image.String())
}

func getLatestImage(images []SImage) SImage {
	var latestBuild string
	latestBuildIdx := -1
	for i := range images {
		if latestBuildIdx < 0 || comapreImageBuildIds(latestBuild, images[i]) < 0 {
			latestBuild = images[i].OSBuildId
			latestBuildIdx = i
		}
	}
	return images[latestBuildIdx]
}

func (self *SRegion) GetImages(status ImageStatusType, owners []TImageOwnerType, imageId []string, name string, virtualizationType string, ownerIds []string, volumeType string, latest bool) ([]SImage, error) {
	images, err := self.getImages(status, owners, imageId, name, virtualizationType, ownerIds, volumeType)
	if err != nil {
		return nil, err
	}
	if !latest {
		return images, err
	}
	noVersionImages := make([]SImage, 0)
	versionedImages := make(map[string][]SImage)
	for i := range images {
		key := fmt.Sprintf("%s%s", images[i].OSDist, images[i].OSVersion)
		if len(key) == 0 {
			noVersionImages = append(noVersionImages, images[i])
			continue
		}
		if _, ok := versionedImages[key]; !ok {
			versionedImages[key] = make([]SImage, 0)
		}
		versionedImages[key] = append(versionedImages[key], images[i])
	}
	for key := range versionedImages {
		noVersionImages = append(noVersionImages, getLatestImage(versionedImages[key]))
	}
	return noVersionImages, nil
}

func (self *SRegion) getImages(status ImageStatusType, owners []TImageOwnerType, imageId []string, name string, virtualizationType string, ownerIds []string, volumeType string) ([]SImage, error) {
	params := &ec2.DescribeImagesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(status) > 0 {
		filters = AppendSingleValueFilter(filters, "state", string(status))
	}

	if len(name) > 0 {
		filters = AppendSingleValueFilter(filters, "name", name)
	}

	if len(virtualizationType) > 0 {
		filters = AppendSingleValueFilter(filters, "virtualization-type", virtualizationType)
	}

	if len(volumeType) > 0 {
		filters = AppendSingleValueFilter(filters, "block-device-mapping.volume-type", volumeType)
	}

	filters = AppendSingleValueFilter(filters, "image-type", "machine")

	if len(owners) > 0 || len(ownerIds) > 0 {
		params.SetOwners(imageOwnerTypes2Strings(owners, ownerIds))
	}

	if len(imageId) > 0 {
		params.SetImageIds(ConvertedList(imageId))
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	ret, err := self.ec2Client.DescribeImages(params)
	err = parseNotFoundError(err)
	if err != nil {
		return nil, err
	}

	images := []SImage{}
	for i := range ret.Images {
		image := ret.Images[i]

		if err := FillZero(image); err != nil {
			return nil, err
		}

		tagspec := TagSpec{}
		tagspec.LoadingEc2Tags(image.Tags)

		size, err := getRootDiskSize(image)
		if err != nil {
			// fail to get disk size, ignore the image
			/// log.Debugln(err)
			continue
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

		name := tagspec.GetNameTag()
		if len(name) == 0 && image.Name != nil {
			name = *image.Name
		}

		sImage := SImage{
			storageCache: self.getStoragecache(),
			Architecture: *image.Architecture,
			Description:  *image.Description,
			ImageId:      *image.ImageId,
			Public:       *image.Public,
			ImageName:    name,
			OSType:       osType,
			// ImageType:          *image.ImageType,
			OwnerType:          *image.ImageOwnerAlias,
			EnaSupport:         *image.EnaSupport,
			Platform:           *image.Platform,
			Status:             ImageStatusType(*image.State),
			CreationTime:       createTime,
			SizeGB:             size,
			RootDevice:         rootDevice,
			VirtualizationType: *image.VirtualizationType,
			Hypervisor:         *image.Hypervisor,
			ProductCodes:       image.ProductCodes,
			OwnerId:            *image.OwnerId,
		}
		sImage.ImageType = getImageType(sImage)
		sImage.OSType = getImageOSType(sImage)
		sImage.OSDist = getImageOSDist(sImage)
		sImage.OSVersion = getImageOSVersion(sImage)
		sImage.OSBuildId = getImageOSBuildID(sImage)
		images = append(images, sImage)
	}

	return images, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	params := &ec2.DeregisterImageInput{}
	params.SetImageId(imageId)
	_, err := self.ec2Client.DeregisterImage(params)
	return err
}

func (self *SRegion) addTags(resId string, key string, value string) error {
	input := &ec2.CreateTagsInput{}
	input.SetResources([]*string{&resId})
	tag := ec2.Tag{}
	tag.Key = &key
	tag.Value = &value
	input.SetTags([]*ec2.Tag{&tag})
	_, err := self.ec2Client.CreateTags(input)
	if err != nil {
		return err
	}
	return nil
}
