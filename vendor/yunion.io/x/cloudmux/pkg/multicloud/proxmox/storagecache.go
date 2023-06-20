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
	"io"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
)

type SStoragecache struct {
	multicloud.SResourceBase
	ProxmoxTags

	region  *SRegion
	Node    string
	isShare bool
}

func (self *SStoragecache) GetGlobalId() string {
	if self.isShare {
		return fmt.Sprintf("%s-share", self.region.GetGlobalId())
	}
	return fmt.Sprintf("%s-%s", self.region.GetGlobalId(), self.Node)
}

func (self *SStoragecache) GetId() string {
	return self.region.GetId()
}

func (self *SStoragecache) GetName() string {
	if self.isShare {
		return fmt.Sprintf("%s-share", self.region.GetName())
	}
	return fmt.Sprintf("%s-%s", self.region.GetName(), self.Node)
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	ret := []cloudprovider.ICloudImage{}
	storages, err := self.region.GetStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if !strings.Contains(storages[i].Content, "iso") {
			continue
		}
		if (self.isShare && storages[i].Shared != 1) || (!self.isShare && storages[i].Node != self.Node) {
			continue
		}
		images, err := self.region.GetImages(storages[i].Node, storages[i].Storage)
		if err != nil {
			return nil, err
		}
		for i := range images {
			images[i].cache = self
			ret = append(ret, &images[i])
		}
	}
	return ret, nil
}

func (seflf *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) GetIImageById(id string) (cloudprovider.ICloudImage, error) {
	images, err := self.GetICloudImages()
	if err != nil {
		return nil, err
	}
	for i := range images {
		if images[i].GetGlobalId() == id {
			return images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) UploadImage(ctx context.Context, opts *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	reader, sizeByte, err := opts.GetReader(opts.ImageId, string(qemuimgfmt.ISO))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}
	storages, err := self.region.GetStorages()
	if err != nil {
		return "", err
	}
	for i := range storages {
		if (self.isShare && storages[i].Shared == 0) || (!self.isShare && storages[i].Shared == 1) {
			continue
		}
		if !strings.Contains(storages[i].Content, "iso") {
			continue
		}
		if storages[i].MaxDisk-storages[i].Disk < sizeByte {
			continue
		}
		log.Debugf("upload image %s for %s %s", opts.ImageName, storages[i].Node, storages[i].Storage)
		image, err := self.region.UploadImage(storages[i].Node, storages[i].Storage, opts.ImageName, reader)
		if err != nil {
			return "", errors.Wrapf(err, "UploadImage")
		}
		return image.GetGlobalId(), nil
	}
	return "", fmt.Errorf("no valid shared storage for upload")
}

func (self *SRegion) UploadImage(node, storage, filename string, reader io.Reader) (*SImage, error) {
	return self.client.upload(node, storage, filename, reader)
}
