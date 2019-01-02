package huawei

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ImageStatusType string
type ImageOwnerType string

const ImageOwnerSelf ImageOwnerType = "private"

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0020091565.html
type SImage struct {
	storageCache *SStoragecache

	Schema             string `json:"schema"`
	MinDisk            int64  `json:"min_disk"`
	CreatedAt          string `json:"created_at"`
	ImageSourceType    string `json:"__image_source_type"`
	ContainerFormat    string `json:"container_format"`
	File               string `json:"file"`
	UpdatedAt          string `json:"updated_at"`
	Protected          bool   `json:"protected"`
	Checksum           string `json:"checksum"`
	SupportKVMFPGAType string `json:"__support_kvm_fpga_type"`
	ID                 string `json:"id"`
	Isregistered       string `json:"__isregistered"`
	MinRAM             int64  `json:"min_ram"`
	Lazyloading        string `json:"__lazyloading"`
	Owner              string `json:"owner"`
	OSType             string `json:"__os_type"`
	Imagetype          string `json:"__imagetype"`
	Visibility         string `json:"visibility"`
	VirtualEnvType     string `json:"virtual_env_type"`
	Platform           string `json:"__platform"`
	Size               int64  `json:"size"`
	OSBit              string `json:"__os_bit"`
	OSVersion          string `json:"__os_version"`
	Name               string `json:"name"`
	Self               string `json:"self"`
	DiskFormat         string `json:"disk_format"`
	Status             string `json:"status"`
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
	case "queued":
		return models.IMAGE_STATUS_QUEUED
	case "active":
		return models.IMAGE_STATUS_ACTIVE
	case "killed":
		return models.IMAGE_STATUS_KILLED
	default:
		return models.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	new, err := self.storageCache.region.GetImage(self.GetId())
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
	if len(self.OSBit) > 0 {
		data.Add(jsonutils.NewString(self.OSBit), "os_arch")
	}
	if len(self.OSType) > 0 {
		data.Add(jsonutils.NewString(self.OSType), "os_name")
	}
	if len(self.Platform) > 0 {
		data.Add(jsonutils.NewString(self.Platform), "os_distribution")
	}
	if len(self.OSVersion) > 0 {
		data.Add(jsonutils.NewString(self.OSVersion), "os_version")
	}
	return data
}

func (self *SImage) Delete(ctx context.Context) error {
	// todo: implement me
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

func (self *SRegion) GetImages(status string, imagetype ImageOwnerType, name string, limit int, marker string) ([]SImage, int, error) {
	querys := map[string]string{}
	if len(status) > 0 {
		querys["status"] = status
	}

	if len(imagetype) > 0 {
		querys["__imagetype"] = string(imagetype)
	}

	if len(name) > 0 {
		querys["name"] = name
	}

	if len(marker) > 0 {
		querys["marker"] = marker
	}

	images := make([]SImage, 0)
	err := DoList(self.ecsClient.Images.List, querys, &images)
	return images, len(images), err
}

func (self *SRegion) DeleteImage(imageId string) error {
	// todo: implement me
	return nil
}
