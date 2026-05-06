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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	ProxmoxTags

	client *SProxmoxClient
	Node   string
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.client.GetGlobalId(), self.Node)
}

func (self *SStoragecache) GetId() string {
	return self.client.GetId()
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.client.GetName(), self.Node)
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SProxmoxClient) GetTemplateImages(node string) ([]SImage, error) {
	ret := []SImage{}
	vms, err := self.GetClusterVmResources()
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if !vms[i].Template {
			continue
		}
		if len(node) > 0 && vms[i].Node != node {
			continue
		}
		image := SImage{
			Name:  vms[i].Name,
			Volid: vms[i].Id,
			Size:  vms[i].Size,
		}
		ret = append(ret, image)
	}
	return ret, nil
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	ret := []cloudprovider.ICloudImage{}
	images, err := self.client.GetTemplateImages(self.Node)
	if err != nil {
		return nil, err
	}
	for i := range images {
		images[i].cache = self
		ret = append(ret, &images[i])
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

func (self *SStoragecache) UploadImage(ctx context.Context, opts *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	return "", cloudprovider.ErrNotSupported
}
