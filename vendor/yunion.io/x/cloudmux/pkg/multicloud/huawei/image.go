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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	ImageStatusQueued  = "queued"  // queued：表示镜像元数据已经创建成功，等待上传镜像文件。
	ImageStatusSaving  = "saving"  // saving：表示镜像正在上传文件到后端存储。
	ImageStatusDeleted = "deleted" // deleted：表示镜像已经删除。
	ImageStatusKilled  = "killed"  // killed：表示镜像上传错误。
	ImageStatusActive  = "active"  // active：表示镜像可以正常使用
)

type SImage struct {
	multicloud.SImageBase
	HuaweiTags
	storageCache *SStoragecache

	// normalized image info
	imgInfo *imagetools.ImageInfo

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
		return api.CACHED_IMAGE_STATUS_ACTIVE
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
	image, err := self.storageCache.region.GetImage(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	switch self.Imagetype {
	case "gold":
		return cloudprovider.ImageTypeSystem
	case "private":
		return cloudprovider.ImageTypeCustomized
	case "shared":
		return cloudprovider.ImageTypeShared
	default:
		return cloudprovider.ImageTypeCustomized
	}
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.MinDiskGB) * 1024 * 1024 * 1024
}

func (self *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if self.imgInfo == nil {
		arch := "x86"
		if strings.ToLower(self.SupportArm) == "true" {
			arch = "arm"
		}
		imgInfo := imagetools.NormalizeImageInfo(self.ImageSourceType, arch, self.OSType, self.Platform, "")
		self.imgInfo = &imgInfo
	}

	return self.imgInfo
}

func (self *SImage) GetFullOsName() string {
	return self.ImageSourceType
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(self.getNormalizedImageInfo().OsType)
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

func (self *SImage) GetOsArch() string {
	return self.getNormalizedImageInfo().OsArch
}

func (self *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.getNormalizedImageInfo().OsBios)
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.MinDiskGB)
}

func (self *SImage) GetImageFormat() string {
	return self.DiskFormat
}

func (self *SImage) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.GetId())
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	images, err := self.GetImages(imageId, "", "", "")
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].ID == imageId {
			return &images[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, imageId)
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IMS/doc?api=ListImages
func (self *SRegion) GetImages(id, status string, imagetype string, name string) ([]SImage, error) {
	query := url.Values{}
	query.Set("virtual_env_type", "FusionCompute")
	if len(status) > 0 {
		query.Set("status", status)
	}

	if len(id) > 0 {
		query.Set("id", id)
	}

	if len(imagetype) > 0 {
		query.Set("__imagetype", string(imagetype))
		if imagetype == "gold" {
			query.Set("protected", "True")
		}
	}

	if len(name) > 0 {
		query.Set("name", name)
	}

	ret := []SImage{}
	for {
		resp, err := self.list(SERVICE_IMS, "cloudimages", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Images []SImage
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.Images...)
		if len(part.Images) == 0 {
			break
		}
		query.Set("marker", part.Images[len(part.Images)-1].ID)
	}

	return ret, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	images, err := self.GetImages("", "", "", name)
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].Name == name {
			return &images[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (self *SRegion) ImportImageJob(name string, osDist string, osVersion string, osArch string, bucket string, key string, minDiskGB int64) (string, error) {
	osVersion, err := stdVersion(osDist, osVersion, osArch)
	log.Debugf("%s %s %s: %s.min_disk %d GB", osDist, osVersion, osArch, osVersion, minDiskGB)

	imageUrl := fmt.Sprintf("%s:%s", bucket, key)
	params := map[string]interface{}{
		"name":           name,
		"image_url":      imageUrl,
		"is_config_init": true,
		"is_config":      true,
		"min_disk":       minDiskGB,
	}
	if len(osVersion) > 0 {
		params["os_version"] = osVersion
	}
	resp, err := self.post(SERVICE_IMS, "cloudimages/quickimport/action", params)
	if err != nil {
		return "", errors.Wrapf(err, "import image")
	}
	return resp.GetString("job_id")
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
	case "64", apis.OS_ARCH_X86_64:
		arch = "64bit"
	case "32", apis.OS_ARCH_X86_32:
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
