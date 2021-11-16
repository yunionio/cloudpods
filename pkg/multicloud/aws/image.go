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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

	ImageId string `xml:"imageId"`
	TaskId  string `xml:"importTaskId"`
	Status  string `xml:"status"`
}

type RootDevice struct {
	SnapshotId string
	Size       int    // GB
	Category   string // VolumeType
}

type SImage struct {
	multicloud.SImageBase
	multicloud.AwsTags
	storageCache *SStoragecache

	Architecture       string               `xml:"architecture"`
	BlockDeviceMapping []BlockDeviceMapping `xml:"blockDeviceMapping"`
	BootMode           string               `xml:"bootMode"`
	CreationDate       time.Time            `xml:"creationDate"`
	DeprecationTime    string               `xml:"deprecationTime"`
	Description        string               `xml:"description"`
	EnaSupport         bool                 `xml:"enaSupport"`
	Hypervisor         string               `xml:"hypervisor"`
	ImageId            string               `xml:"imageId"`
	ImageLocation      string               `xml:"imageLocation"`
	ImageOwnerAlias    string               `xml:"imageOwnerAlias"`
	ImageOwnerId       string               `xml:"imageOwnerId"`
	ImageState         string               `xml:"imageState"`
	ImageType          string               `xml:"imageType"`
	IsPublic           bool                 `xml:"isPublic"`
	KernelId           string               `xml:"kernelId"`
	Name               string               `xml:"name"`
	Platform           string               `xml:"platform"`
	PlatformDetails    string               `xml:"platformDetails"`
	ProductCodes       []ProductCode        `xml:"productCodes"`
	RamdiskId          string               `xml:"ramdiskId"`
	RootDeviceName     string               `xml:"rootDeviceName"`
	RootDeviceType     string               `xml:"rootDeviceType"`
	SriovNetSupport    string               `xml:"sriovNetSupport"`
	StateReason        StateReason          `xml:"stateReason"`
	UsageOperation     string               `xml:"usageOperation"`
	VirtualizationType string               `xml:"virtualizationType"`

	// only use for import image
	Status       string `xml:"status"`
	ImportTaskId string `xml:"importTaskId"`
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return self.ImageId
}

func (self *SImage) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ImageId
}

func (self *SImage) GetGlobalId() string {
	return self.ImageId
}

// pending | available | invalid | deregistered | transient | failed | error
func (self *SImage) GetStatus() string {
	switch self.ImageState {
	case "pending":
		return api.CACHED_IMAGE_STATUS_CACHING
	case "available":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	case "failed", "error":
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.ImageState {
	case "pending":
		return cloudprovider.IMAGE_STATUS_QUEUED
	case "available":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "failed", "error":
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
	_, ok := awsImagePublishers[self.ImageOwnerId]
	if ok {
		return cloudprovider.ImageTypeSystem
	}
	if !self.IsPublic {
		return cloudprovider.ImageTypeCustomized
	}
	return cloudprovider.ImageTypeMarket
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.getRootDiskSizeGb()) * 1024 * 1024 * 1024
}

func (self *SImage) getRootDiskSizeGb() int {
	for _, block := range self.BlockDeviceMapping {
		if block.DeviceName == self.RootDeviceName {
			return block.Ebs.VolumeSize
		}
	}
	return 0
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	if strings.Contains(strings.ToLower(self.PlatformDetails), "linux") {
		return cloudprovider.OsTypeLinux
	}
	return cloudprovider.OsTypeWindows
}

func (self *SImage) GetOsArch() string {
	switch self.Architecture {
	case "x86_64", "x86_64_mac":
		return apis.OS_ARCH_X86_64
	case "i386":
		return apis.OS_ARCH_I386
	case "arm64":
		return apis.OS_ARCH_AARCH64
	default:
		return apis.OS_ARCH_X86_64
	}
}

func (self *SImage) GetOsDist() string {
	ownerInfo, ok := awsImagePublishers[self.ImageOwnerId]
	if ok {
		return ownerInfo.GetOSDist(*self)
	}
	return ""
}

func (self *SImage) GetOsVersion() string {
	ownerInfo, ok := awsImagePublishers[self.ImageOwnerId]
	if ok {
		return ownerInfo.GetOSVersion(*self)
	}
	return ""
}

func (self *SImage) GetOsBuildId() string {
	ownerInfo, ok := awsImagePublishers[self.ImageOwnerId]
	if ok {
		return ownerInfo.GetOSBuildID(*self)
	}
	return ""
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return self.getRootDiskSizeGb()
}

func (self *SImage) GetImageFormat() string {
	return "vhd"
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.CreationDate
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.ImageId)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) DescribeExportImageTasks(taskId string) (*ImageExportTask, error) {
	params := map[string]string{
		"ExportImageTaskId.1": taskId,
	}
	ret := struct {
		Tasks []ImageExportTask `xml:"exportImageTaskSet>item"`
	}{}
	err := self.ec2Request("DescribeExportImageTasks", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeExportImageTasks")
	}
	for i := range ret.Tasks {
		if ret.Tasks[i].TaskId == taskId {
			return &ret.Tasks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "with task id %s", taskId)
}

func (self *SRegion) DescribeImportImageTasks(taskId string) (*ImageImportTask, error) {
	params := map[string]string{
		"ImportImageTaskId.1": taskId,
	}
	ret := struct {
		Tasks []ImageImportTask `xml:"importImageTaskSet>item"`
	}{}
	err := self.ec2Request("DescribeImportImageTasks", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeImportImageTasks")
	}
	for i := range ret.Tasks {
		if ret.Tasks[i].TaskId == taskId {
			return &ret.Tasks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "with task id %s", taskId)
}

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ImportImage.html
func (self *SRegion) ImportImage(name string, osArch string, osType string, osDist string, diskFormat string, bucket string, key string) (*SImage, error) {
	params := map[string]string{
		"Architecture": osArch,
		"Hypervisor":   "xen",
		"Platform":     osType,
		// https://docs.aws.amazon.com/zh_cn/vm-import/latest/userguide/vmimport-image-import.html#import-vm-image
		"RoleName":                            "vmimport",
		"DiskContainer.1.Description":         fmt.Sprintf("vmimport %s - %s", name, osDist),
		"DiskContainer.1.Format":              diskFormat,
		"DiskContainer.1.DeviceName":          "/dev/sda",
		"DiskContainer.1.UserBucket.S3Bucket": bucket,
		"DiskContainer.1.UserBucket.S3Key":    key,
		"LicenseType":                         "BYOL",
		"TagSpecification.1.ResourceType":     "import-image-task",
		"TagSpecification.1.Tags.1.Key":       "Name",
		"TagSpecification.1.Tags.1.Value":     name,
	}
	image := &SImage{}
	return image, self.ec2Request("ImportImage", params, image)
}

type ImageExportTask struct {
	ImageId string
	TaskId  string `xml:"exportTaskId"`
}

func (self *SRegion) ExportImage(instanceId string, imageId string) (*ImageExportTask, error) {
	params := map[string]string{
		"InstanceId":                 instanceId,
		"Description":                fmt.Sprintf("image %s export from aws", imageId),
		"TargetEnvironment":          "vmware",
		"ExportToS3.ContainerFormat": "ova",
		"ExportToS3.DiskImageFormat": "RAW",
		"ExportToS3.S3Bucket":        "imgcache-onecloud",
	}
	ret := struct {
		ExportTask ImageExportTask `xml:"exportTask"`
	}{
		ExportTask: ImageExportTask{
			ImageId: imageId,
		},
	}
	return &ret.ExportTask, self.ec2Request("CreateInstanceExportTask", params, &ret)
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	if len(imageId) == 0 {
		return nil, fmt.Errorf("GetImage image id should not be empty")
	}

	images, err := self.getImages("", ImageOwnerAll, []string{imageId}, "", "", nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "getImages")
	}
	for i := range images {
		if images[i].ImageId == imageId {
			return &images[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetImage(%s)", imageId)
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

func getLatestImage(images []SImage) SImage {
	var latestBuild string
	latestBuildIdx := -1
	for i := range images {
		if latestBuildIdx < 0 || comapreImageBuildIds(latestBuild, images[i]) < 0 {
			latestBuild = images[i].GetOsBuildId()
			latestBuildIdx = i
		}
	}
	return images[latestBuildIdx]
}

func (self *SRegion) GetImages(status string, owners []TImageOwnerType, imageId []string, name string, virtualizationType string, ownerIds []string, volumeType string, latest bool) ([]SImage, error) {
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
		key := fmt.Sprintf("%s%s", images[i].GetOsDist(), images[i].GetOsVersion())
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

func (self *SRegion) getImages(status string, owners []TImageOwnerType, imageIds []string, name string, virtualizationType string, ownerIds []string, volumeType string) ([]SImage, error) {
	params := map[string]string{}
	idx := 1
	for k, v := range map[string]string{
		"state":                            status,
		"name":                             name,
		"virtualization-type":              virtualizationType,
		"block-device-mapping.volume-type": volumeType,
		"image-type":                       "machine",
	} {
		if len(v) > 0 {
			params[fmt.Sprintf("Filter.%d.Name", idx)] = k
			params[fmt.Sprintf("Filter.%d.Value", idx)] = v
			idx++
		}
	}
	for k, ids := range map[string][]string{
		"Owner":   ownerIds,
		"ImageId": imageIds,
	} {
		for i, id := range ids {
			params[fmt.Sprintf("%s.%d", k, i+1)] = id

		}
	}

	ret := struct {
		Images []SImage `xml:"imagesSet>item"`
	}{}
	err := self.ec2Request("DescribeImages", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeImages")
	}
	return ret.Images, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	params := map[string]string{
		"ImageId": imageId,
	}
	return self.ec2Request("DeregisterImage", params, nil)
}

func (self *SImage) UEFI() bool {
	return self.BootMode == "uefi"
}
