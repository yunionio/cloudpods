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

package huawei

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type TImageOwnerType string

const (
	ImageOwnerPublic TImageOwnerType = "gold"    // 公共镜像：gold
	ImageOwnerSelf   TImageOwnerType = "private" // 私有镜像：private
	ImageOwnerShared TImageOwnerType = "shared"  // 共享镜像：shared

	EnvFusionCompute = "FusionCompute"
	EnvIronic        = "Ironic"
)

const (
	ImageStatusQueued  = "queued"  // queued：表示镜像元数据已经创建成功，等待上传镜像文件。
	ImageStatusSaving  = "saving"  // saving：表示镜像正在上传文件到后端存储。
	ImageStatusDeleted = "deleted" // deleted：表示镜像已经删除。
	ImageStatusKilled  = "killed"  // killed：表示镜像上传错误。
	ImageStatusActive  = "active"  // active：表示镜像可以正常使用
)

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0020091565.html
type SImage struct {
	storageCache *SStoragecache

	Schema                 string    `json:"schema"`
	MinDiskGB              int64     `json:"min_disk"`
	CreatedAt              time.Time `json:"created_at"`
	ImageSourceType        string    `json:"__image_source_type"`
	ContainerFormat        string    `json:"container_format"`
	File                   string    `json:"file"`
	UpdatedAt              time.Time `json:"updated_at"`
	Protected              bool      `json:"protected"`
	Checksum               string    `json:"checksum"`
	ID                     string    `json:"id"`
	Isregistered           string    `json:"__isregistered"`
	MinRamMB               int       `json:"min_ram"`
	Lazyloading            string    `json:"__lazyloading"`
	Owner                  string    `json:"owner"`
	OSType                 string    `json:"__os_type"`
	Imagetype              string    `json:"__imagetype"`
	Visibility             string    `json:"visibility"`
	VirtualEnvType         string    `json:"virtual_env_type"`
	Platform               string    `json:"__platform"`
	SizeGB                 int       `json:"size"`
	ImageSize              int64     `json:"__image_size"`
	OSBit                  string    `json:"__os_bit"`
	OSVersion              string    `json:"__os_version"`
	Name                   string    `json:"name"`
	Self                   string    `json:"self"`
	DiskFormat             string    `json:"disk_format"`
	Status                 string    `json:"status"`
	SupportKVMFPGAType     string    `json:"__support_kvm_fpga_type"`
	SupportKVMNVMEHIGHIO   string    `json:"__support_nvme_highio"`
	SupportLargeMemory     string    `json:"__support_largememory"`
	SupportDiskIntensive   string    `json:"__support_diskintensive"`
	SupportHighPerformance string    `json:"__support_highperformance"`
	SupportXENGPUType      string    `json:"__support_xen_gpu_type"`
	SupportKVMGPUType      string    `json:"__support_kvm_gpu_type"`
	SupportGPUT4           string    `json:"__support_gpu_t4"`
	SupportKVMAscend310    string    `json:"__support_kvm_ascend_310"`
	SupportArm             string    `json:"__support_arm"`
}

func (self *SImage) GetMinRamSizeMb() int {
	return self.MinRamMB
}

func (self *SImage) GetId() string {
	return self.ID
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) GetGlobalId() string {
	return self.ID
}

func (self *SImage) GetStatus() string {
	switch self.Status {
	case ImageStatusQueued:
		return api.CACHED_IMAGE_STATUS_CACHING
	case ImageStatusActive:
		return api.CACHED_IMAGE_STATUS_READY
	case ImageStatusKilled:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	default:
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.Status {
	case ImageStatusQueued:
		return cloudprovider.IMAGE_STATUS_QUEUED
	case ImageStatusActive:
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case ImageStatusKilled:
		return cloudprovider.IMAGE_STATUS_KILLED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	new, err := self.storageCache.region.GetImage(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SImage) GetImageType() string {
	switch self.Imagetype {
	case "gold":
		return cloudprovider.CachedImageTypeSystem
	case "private":
		return cloudprovider.CachedImageTypeCustomized
	case "shared":
		return cloudprovider.CachedImageTypeShared
	default:
		return cloudprovider.CachedImageTypeCustomized
	}
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.MinDiskGB) * 1024 * 1024 * 1024
}

func (self *SImage) GetOsType() string {
	return self.OSType
}

func (self *SImage) GetOsDist() string {
	return self.Platform
}

func (self *SImage) GetOsVersion() string {
	return self.OSVersion
}

func (self *SImage) GetOsArch() string {
	if self.OSType == "32" {
		return "x86"
	} else {
		return "x86_64"
	}
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.MinDiskGB)
}

func (self *SImage) GetImageFormat() string {
	return self.DiskFormat
}

func (self *SImage) GetCreateTime() time.Time {
	return self.CreatedAt
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	if len(self.OSBit) > 0 {
		data.Add(jsonutils.NewString(self.GetOsArch()), "os_arch")
	}
	if len(self.OSType) > 0 {
		data.Add(jsonutils.NewString(self.GetOsType()), "os_name")
	}
	if len(self.Platform) > 0 {
		data.Add(jsonutils.NewString(self.GetOsDist()), "os_distribution")
	}
	if len(self.OSVersion) > 0 {
		data.Add(jsonutils.NewString(self.GetOsVersion()), "os_version")
	}
	return data
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.GetId())
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) GetImage(imageId string) (SImage, error) {
	image := SImage{}
	err := DoGet(self.ecsClient.Images.Get, imageId, nil, &image)
	return image, err
}

func excludeImage(image SImage) bool {
	if image.VirtualEnvType == "Ironic" {
		return true
	}

	if len(image.SupportDiskIntensive) > 0 {
		return true
	}

	if len(image.SupportKVMFPGAType) > 0 || len(image.SupportKVMAscend310) > 0 {
		return true
	}

	if len(image.SupportKVMGPUType) > 0 {
		return true
	}

	if len(image.SupportKVMNVMEHIGHIO) > 0 {
		return true
	}

	if len(image.SupportGPUT4) > 0 {
		return true
	}

	if len(image.SupportXENGPUType) > 0 {
		return true
	}

	if len(image.SupportHighPerformance) > 0 {
		return true
	}

	if len(image.SupportArm) > 0 {
		return true
	}

	return false
}

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0060804959.html
func (self *SRegion) GetImages(status string, imagetype TImageOwnerType, name string, envType string) ([]SImage, error) {
	queries := map[string]string{}
	if len(status) > 0 {
		queries["status"] = status
	}

	if len(imagetype) > 0 {
		queries["__imagetype"] = string(imagetype)
		if imagetype == ImageOwnerPublic {
			queries["protected"] = "True"
		}
	}
	if len(envType) > 0 {
		queries["virtual_env_type"] = envType
	}

	if len(name) > 0 {
		queries["name"] = name
	}

	images := make([]SImage, 0)
	err := doListAllWithMarker(self.ecsClient.Images.List, queries, &images)

	// 排除掉需要特定镜像才能创建的实例类型
	// https://support.huaweicloud.com/eu-west-0-api-ims/zh-cn_topic_0031617666.html#ZH-CN_TOPIC_0031617666__table48545918250
	// https://support.huaweicloud.com/productdesc-ecs/zh-cn_topic_0088142947.html
	filtedImages := make([]SImage, 0)
	for i := range images {
		if !excludeImage(images[i]) {
			filtedImages = append(filtedImages, images[i])
		}
	}

	return filtedImages, err
}

func (self *SRegion) DeleteImage(imageId string) error {
	return DoDelete(self.ecsClient.OpenStackImages.Delete, imageId, nil, nil)
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("image name should not be empty")
	}

	images, err := self.GetImages("", TImageOwnerType(""), name, "")
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	log.Debugf("%d image found match name %s", len(images), name)
	return &images[0], nil
}

/* https://support.huaweicloud.com/api-ims/zh-cn_topic_0020092109.html
   os version 取值范围： https://support.huaweicloud.com/api-ims/zh-cn_topic_0031617666.html
   用于创建私有镜像的源云服务器系统盘大小大于等于40GB且不超过1024GB。
   目前支持vhd，zvhd、raw，qcow2
   todo: 考虑使用镜像快速导入。 https://support.huaweicloud.com/api-ims/zh-cn_topic_0133188204.html
   使用OBS文件创建镜像

   * openstack原生接口支持的格式：https://support.huaweicloud.com/api-ims/zh-cn_topic_0031615566.html
*/
func (self *SRegion) ImportImageJob(name string, osDist string, osVersion string, osArch string, bucket string, key string, minDiskGB int64) (string, error) {
	os_version, err := stdVersion(osDist, osVersion, osArch)
	log.Debugf("%s %s %s: %s.min_disk %d GB", osDist, osVersion, osArch, os_version, minDiskGB)
	if err != nil {
		log.Debugln(err)
	}

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(name), "name")
	image_url := fmt.Sprintf("%s:%s", bucket, key)
	params.Add(jsonutils.NewString(image_url), "image_url")
	if len(os_version) > 0 {
		params.Add(jsonutils.NewString(os_version), "os_version")
	}
	params.Add(jsonutils.NewBool(true), "is_config_init")
	params.Add(jsonutils.NewBool(true), "is_config")
	params.Add(jsonutils.NewInt(minDiskGB), "min_disk")

	ret, err := self.ecsClient.Images.PerformAction2("action", "", params, "")
	if err != nil {
		return "", err
	}

	return ret.GetString("job_id")
}

func formatVersion(osDist string, osVersion string) (string, error) {
	err := fmt.Errorf("unsupport version %s.reference: https://support.huaweicloud.com/api-ims/zh-cn_topic_0031617666.html", osVersion)
	dist := strings.ToLower(osDist)
	if dist == "ubuntu" || dist == "redhat" || dist == "centos" || dist == "oracle" || dist == "euleros" {
		parts := strings.Split(osVersion, ".")
		if len(parts) < 2 {
			return "", err
		}

		return parts[0] + "." + parts[1], nil
	}

	if dist == "debian" {
		parts := strings.Split(osVersion, ".")
		if len(parts) < 3 {
			return "", err
		}

		return parts[0] + "." + parts[1] + "." + parts[2], nil
	}

	if dist == "fedora" || dist == "windows" || dist == "suse" {
		parts := strings.Split(osVersion, ".")
		if len(parts) < 1 {
			return "", err
		}

		return parts[0], nil
	}

	if dist == "opensuse" {
		parts := strings.Split(osVersion, ".")
		if len(parts) == 0 {
			return "", err
		}

		if len(parts) == 1 {
			return parts[0], nil
		}

		if len(parts) >= 2 {
			return parts[0] + "." + parts[1], nil
		}
	}

	return "", err
}

// todo: 如何保持同步更新
// https://support.huaweicloud.com/api-ims/zh-cn_topic_0031617666.html
func stdVersion(osDist string, osVersion string, osArch string) (string, error) {
	// 架构
	arch := ""
	switch osArch {
	case "64", "x86_64":
		arch = "64bit"
	case "32", "x86_32":
		arch = "32bit"
	default:
		return "", fmt.Errorf("unsupported arch %s.reference: https://support.huaweicloud.com/api-ims/zh-cn_topic_0031617666.html", osArch)
	}

	_dist := strings.Split(strings.TrimSpace(osDist), " ")[0]
	_dist = strings.ToLower(_dist)
	// 版本
	ver, err := formatVersion(_dist, osVersion)
	if err != nil {
		return "", err
	}

	//  操作系统
	dist := ""

	switch _dist {
	case "ubuntu":
		return fmt.Sprintf("Ubuntu %s server %s", ver, arch), nil
	case "redhat":
		dist = "Redhat Linux Enterprise"
	case "centos":
		dist = "CentOS"
	case "fedora":
		dist = "Fedora"
	case "debian":
		dist = "Debian GNU/Linux"
	case "windows":
		dist = "Windows Server"
	case "oracle":
		dist = "Oracle Linux Server release"
	case "suse":
		dist = "SUSE Linux Enterprise Server"
	case "opensuse":
		dist = "OpenSUSE"
	case "euleros":
		dist = "EulerOS"
	default:
		return "", fmt.Errorf("unsupported os %s. reference: https://support.huaweicloud.com/api-ims/zh-cn_topic_0031617666.html", dist)
	}

	return fmt.Sprintf("%s %s %s", dist, ver, arch), nil
}
