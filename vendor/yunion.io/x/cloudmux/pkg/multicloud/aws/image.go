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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "pending"
	ImageStatusAvailable    ImageStatusType = "available"
	ImageStatusCreateFailed ImageStatusType = "failed"

	ImageImportStatusCompleted   = "completed"
	ImageImportStatusUncompleted = "uncompleted"
	ImageImportStatusError       = "error"
	ImageImportStatusDeleted     = "deleted"
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
	multicloud.SResourceBase
	region *SRegion

	ImageId  string `xml:"imageId"`
	TaskId   string `xml:"importTaskId"`
	Progress string `xml:"progress"`
	Status   string `xml:"status"`
}

type RootDevice struct {
	SnapshotId string
	Size       int    // GB
	Category   string // VolumeType
}

type SImage struct {
	multicloud.SImageBase
	AwsTags
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

	Architecture       string          `xml:"architecture"`
	CreationTime       time.Time       `xml:"creationDate"`
	Description        string          `xml:"description"`
	ImageId            string          `xml:"imageId"`
	ImageName          string          `xml:"name"`
	ImageType          string          `xml:"imageType"`
	EnaSupport         bool            `xml:"enaSupport"`
	Platform           string          `xml:"platformDetails"`
	Status             ImageStatusType `xml:"imageState"`
	OwnerType          string          `xml:"imageOwnerAlias"`
	RootDeviceName     string          `xml:"rootDeviceName"`
	BlockDeviceMapping []struct {
		DeviceName string `xml:"deviceName"`
		Ebs        struct {
			SnapshotId          string `xml:"snapshotId"`
			VolumeSize          int    `xml:"volumeSize"`
			DeleteOnTermination bool   `xml:"deleteOnTermination"`
			VolumeType          string `xml:"volumeType"`
			Iops                int    `xml:"iops"`
			Encrypted           bool   `xml:"encrypted"`
		} `xml:"ebs"`
	} `xml:"blockDeviceMapping>item"`

	Public             bool   `xml:"isPublic"`
	Hypervisor         string `xml:"hypervisor"`
	VirtualizationType string `xml:"virtualizationType"`
	OwnerId            string `xml:"imageOwnerId"`
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
	task, err := self.region.GetImportImageTask(self.TaskId)
	if err != nil {
		return errors.Wrapf(err, "GetImportImageTask")
	}
	log.Debugf("DescribeImportImage Task %s", jsonutils.Marshal(task))
	return jsonutils.Update(self, task)
}

func (self *SRegion) GetImportImageTasks(ids []string) ([]ImageImportTask, error) {
	params := map[string]string{}
	for i, id := range ids {
		params[fmt.Sprintf("ImportTaskId.%d", i+1)] = id
	}
	ret := struct {
		ImportImageTaskSet []ImageImportTask `xml:"importImageTaskSet>item"`
	}{}
	err := self.ec2Request("DescribeImportImageTasks", params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeImportImageTasks")
	}
	return ret.ImportImageTaskSet, nil
}

func (self *SRegion) GetImportImageTask(id string) (*ImageImportTask, error) {
	tasks, err := self.GetImportImageTasks([]string{id})
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		if tasks[i].TaskId == id {
			return &tasks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *ImageImportTask) IsEmulated() bool {
	return true
}

func (self *ImageImportTask) GetStatus() string {
	self.Refresh()
	if self.Status == "completed" {
		return ImageImportStatusCompleted
	} else if self.Status == "deleted" {
		return ImageImportStatusDeleted
	} else {
		return ImageImportStatusUncompleted
	}
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
		return api.CACHED_IMAGE_STATUS_ACTIVE
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

func (self *SImage) GetImageType() cloudprovider.TImageType {
	return getImageType(self)
}

func (self *SImage) GetBlockDeviceNames() []string {
	ret := []string{}
	for _, dev := range self.BlockDeviceMapping {
		ret = append(ret, dev.DeviceName)
	}
	return ret
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.GetMinOsDiskSizeGb()) * 1024 * 1024 * 1024
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo("", self.Architecture, getImageOSType(*self), getImageOSDist(*self), getImageOSVersion(*self))
		self.imgInfo = &imgInfo
	}
	return self.imgInfo
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.getNormalizedImageInfo().OsType)
}

func (self *SImage) GetOsArch() string {
	return self.getNormalizedImageInfo().OsArch
}

func (self *SImage) GetOsDist() string {
	return self.getNormalizedImageInfo().OsDistro
}

func (self *SImage) GetOsVersion() string {
	return self.getNormalizedImageInfo().OsVersion
}

func (self *SImage) GetOsLang() string {
	return self.getNormalizedImageInfo().OsLang
}

func (self *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.getNormalizedImageInfo().OsBios)
}

func (self *SImage) GetFullOsName() string {
	return self.getNormalizedImageInfo().GetFullOsName()
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	for _, dev := range self.BlockDeviceMapping {
		if dev.DeviceName == self.RootDeviceName {
			return dev.Ebs.VolumeSize
		}
	}
	return 0
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

func (self *SImage) Delete(ctx context.Context) error {
	// todo: implement me
	return self.storageCache.region.DeleteImage(self.ImageId)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) ImportImage(name string, osArch string, osType string, osDist string, diskFormat string, bucket string, key string) (*ImageImportTask, error) {
	params := map[string]string{
		"Architecture":                    osArch,
		"Hypervisor":                      "xen",
		"Platform":                        osType,
		"RoleName":                        "vmimport",
		"TagSpecification.1.ResourceType": "import-image-task",
		"TagSpecification.1.Tag.1.Key":    "Name",
		"TagSpecification.1.Tag.1.Value":  name,
		"Description":                     fmt.Sprintf("vmimport %s - %s", name, osDist),
		"DiskContainer.1.Format":          strings.ToUpper(diskFormat),
		"DiskContainer.1.DeviceName":      "/dev/sda",
		"DiskContainer.1.Url":             fmt.Sprintf("s3://%s/%s", bucket, key),
		"LicenseType":                     "BYOL",
	}
	ret := &ImageImportTask{region: self}
	err := self.ec2Request("ImportImage", params, ret)
	if err != nil {
		return nil, errors.Wrapf(err, "ImportImage")
	}
	return ret, nil
}

type ImageExportTask struct {
	ImageId  string
	RegionId string
	TaskId   string `xml:"exportTaskId"`
}

func (self *SRegion) ExportImage(instanceId string, imageId string) (*ImageExportTask, error) {
	params := map[string]string{
		"InstanceId":                 instanceId,
		"TargetEnvironment":          "vmware",
		"Description":                fmt.Sprintf("image %s export from aws", imageId),
		"ExportToS3.ContainerFormat": "ova",
		"ExportToS3.DiskImageFormat": "RAW",
		"ExportToS3.S3Bucket":        "imgcache-onecloud",
	}
	ret := struct {
		ExportTask ImageExportTask `xml:"exportTask"`
	}{}
	err := self.ec2Request("CreateInstanceExportTask", params, ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateInstanceExportTask")
	}
	ret.ExportTask.RegionId = self.RegionId
	ret.ExportTask.ImageId = imageId
	return &ret.ExportTask, nil
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	if len(imageId) == 0 {
		return nil, fmt.Errorf("GetImage image id should not be empty")
	}

	images, err := self.getImages("", ImageOwnerAll, []string{imageId}, "", "", nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "getImages")
	}
	if len(images) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "getImages")
	}
	return &images[0], nil
}

func (self *SRegion) GetImageByName(name string, owners []TImageOwnerType) (*SImage, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("image name should not be empty")
	}

	images, err := self.getImages("", owners, nil, name, "hvm", nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "getImages")
	}
	if len(images) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "getImages")
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

func getLatestImage(images []SImage) SImage {
	var latestBuild string
	latestBuildIdx := -1
	for i := range images {
		if latestBuildIdx < 0 || comapreImageBuildIds(latestBuild, images[i]) < 0 {
			latestBuild = getImageOSBuildID(images[i])
			latestBuildIdx = i
		}
	}
	return images[latestBuildIdx]
}

func (self *SRegion) GetImages(status ImageStatusType, owners []TImageOwnerType, imageId []string, name string, virtualizationType string, ownerIds []string, volumeType string, latest bool) ([]SImage, error) {
	images, err := self.getImages(status, owners, imageId, name, virtualizationType, ownerIds, volumeType)
	if err != nil {
		return nil, errors.Wrap(err, "getImages")
	}
	if !latest {
		return images, err
	}
	noVersionImages := make([]SImage, 0)
	versionedImages := make(map[string][]SImage)
	for i := range images {
		key := fmt.Sprintf("%s%s", getImageOSDist(images[i]), getImageOSVersion(images[i]))
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
	params := map[string]string{}
	idx := 1

	if len(status) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "state"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = string(status)
		idx++
	}

	if len(name) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "name"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = name
		idx++
	}

	if len(virtualizationType) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "virtualization-type"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = virtualizationType
		idx++
	}

	if len(volumeType) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "block-device-mapping.volume-type"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = volumeType
		idx++
	}

	if len(owners) > 0 || len(ownerIds) > 0 {
		for i, owner := range imageOwnerTypes2Strings(owners, ownerIds) {
			params[fmt.Sprintf("Owner.%d", i+1)] = string(owner)
		}
	}

	for i, id := range imageId {
		params[fmt.Sprintf("ImageId.%d", i+1)] = id
	}

	params[fmt.Sprintf("Filter.%d.Name", idx)] = "image-type"
	params[fmt.Sprintf("Filter.%d.Value.1", idx)] = "machine"
	idx++

	ret := []SImage{}
	for {
		part := struct {
			ImagesSet []SImage `xml:"imagesSet>item"`
			NextToken string   `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeImages", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeImages")
		}
		ret = append(ret, part.ImagesSet...)

		if len(part.ImagesSet) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	params := map[string]string{"ImageId": imageId}
	ret := struct{}{}
	return self.ec2Request("DeregisterImage", params, &ret)
}
