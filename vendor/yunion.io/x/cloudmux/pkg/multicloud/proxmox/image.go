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

package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SImage struct {
	multicloud.SImageBase
	ProxmoxTags
	cache *SStoragecache

	imageInfo *imagetools.ImageInfo

	VmId   int
	Node   string
	Name   string
	Format string
	SizeGB float64
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetId() string {
	return fmt.Sprintf("%d", self.VmId)
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetStatus() string {
	return api.CACHED_IMAGE_STATUS_ACTIVE
}

func (self *SImage) GetImageStatus() string {
	return cloudprovider.IMAGE_STATUS_ACTIVE
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	return cloudprovider.ImageTypeSystem
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.SizeGB * 1024 * 1024)
}

func (img *SImage) getNormalizedImageInfo() *imagetools.ImageInfo {
	if img.imageInfo == nil {
		imgInfo := imagetools.NormalizeImageInfo(img.Name, "", "", "", "")
		img.imageInfo = &imgInfo
	}
	return img.imageInfo
}

func (img *SImage) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(img.getNormalizedImageInfo().OsType)
}

func (img *SImage) GetOsDist() string {
	return img.getNormalizedImageInfo().OsDistro
}

func (img *SImage) GetOsVersion() string {
	return img.getNormalizedImageInfo().OsVersion
}

func (img *SImage) GetOsArch() string {
	return img.getNormalizedImageInfo().OsArch
}

func (img *SImage) GetOsLang() string {
	return img.getNormalizedImageInfo().OsLang
}

func (img *SImage) GetFullOsName() string {
	return img.Name
}

func (img *SImage) GetBios() cloudprovider.TBiosType {
	return cloudprovider.BIOS
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	if self.GetOsType() == "windows" {
		return 40
	}
	return 30
}

func (self *SImage) GetImageFormat() string {
	return "raw"
}

func (self *SRegion) GetImageList() ([]SImage, error) {
	ret := []SImage{}
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return nil, err
	}
	for _, vm := range resources {
		if vm.Template == true {
			image := SImage{
				VmId: vm.VmId,
				Name: vm.Name,
				Node: vm.Node,
			}

			res := fmt.Sprintf("/nodes/%s/qemu/%d/config", image.Node, image.VmId)
			vmConfig := map[string]interface{}{}
			err := self.get(res, url.Values{}, &vmConfig)
			if err != nil {
				return nil, err
			}

			diskNames := []string{}
			for k := range vmConfig {
				if diskName := regexp.MustCompile(`(virtio|scsi|sata)\d+`).FindStringSubmatch(k); len(diskName) > 0 {
					diskNames = append(diskNames, diskName[0])
				}
			}

			for _, diskName := range diskNames {
				diskConfStr := vmConfig[diskName].(string)
				diskConfMap := ParsePMConf(diskConfStr, "volume")

				if diskConfMap["volume"].(string) == "none" {
					continue
				}
				if diskConfMap["media"] != nil {
					continue
				}

				storageName, fileName := ParseSubConf(diskConfMap["volume"].(string), ":")
				diskConfMap["storage"] = storageName
				diskConfMap["file"] = fileName

				// cloud-init disks not always have the size sent by the API, which results in a crash
				if diskConfMap["size"] == nil && strings.Contains(fileName.(string), "cloudinit") {
					diskConfMap["size"] = "4M" // default cloud-init disk size
				}

				image.SizeGB += DiskSizeGB(diskConfMap["size"])

			}
			ret = append(ret, image)
		}
	}

	return ret, nil
}

func (self *SRegion) GetImage(id string) (*SImage, error) {
	image := &SImage{}
	vmId, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return nil, err
	}
	if resources[vmId].Template == false {
		return nil, errors.Errorf("self.GetDisk")
	}
	image.VmId = resources[vmId].VmId
	image.Name = resources[vmId].Name
	image.Node = resources[vmId].Node
	res := fmt.Sprintf("/nodes/%s/qemu/%d/config", image.Node, image.VmId)
	vmConfig := map[string]interface{}{}
	err = self.get(res, url.Values{}, &vmConfig)
	if err != nil {
		return nil, err
	}

	diskNames := []string{}
	for k := range vmConfig {
		if diskName := regexp.MustCompile(`(virtio|scsi|sata)\d+`).FindStringSubmatch(k); len(diskName) > 0 {
			diskNames = append(diskNames, diskName[0])
		}
	}

	for _, diskName := range diskNames {
		diskConfStr := vmConfig[diskName].(string)
		diskConfMap := ParsePMConf(diskConfStr, "volume")

		if diskConfMap["volume"].(string) == "none" || diskConfMap["media"].(string) == "cdrom" {
			continue
		}

		storageName, fileName := ParseSubConf(diskConfMap["volume"].(string), ":")
		diskConfMap["storage"] = storageName
		diskConfMap["file"] = fileName

		// cloud-init disks not always have the size sent by the API, which results in a crash
		if diskConfMap["size"] == nil && strings.Contains(fileName.(string), "cloudinit") {
			diskConfMap["size"] = "4M" // default cloud-init disk size
		}

		var sizeInTerabytes = regexp.MustCompile(`[0-9]+T`)
		// Convert to gigabytes if disk size was received in terabytes
		matched := sizeInTerabytes.MatchString(diskConfMap["size"].(string))
		if matched {
			image.SizeGB += DiskSizeGB(diskConfMap["size"])
		}

	}

	return image, nil
}
