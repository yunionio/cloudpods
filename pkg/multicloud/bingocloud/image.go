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
package bingocloud

import (
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SImage struct {
	Architecture       string `json:"architecture"`
	BlockDeviceMapping string `json:"blockDeviceMapping"`
	Bootloader         string `json:"bootloader"`
	Clonemode          string `json:"clonemode"`
	ClusterId          string `json:"clusterId"`
	Contentmode        string `json:"contentmode"`
	DefaultStorageId   string `json:"defaultStorageId"`
	Description        string `json:"description"`
	DiskBus            string `json:"diskBus"`
	ExtendDisk         string `json:"extendDisk"`
	Features           string `json:"features"`
	Hypervisor         string `json:"hypervisor"`
	ImageId            string `json:"imageId"`
	ImageLocation      string `json:"imageLocation"`
	ImageOwnerId       string `json:"imageOwnerId"`
	ImagePath          string `json:"imagePath"`
	ImageSize          string `json:"imageSize"`
	ImageState         string `json:"imageState"`
	ImageType          string `json:"imageType"`
	IsBareMetal        string `json:"isBareMetal"`
	IsPublic           string `json:"isPublic"`
	KernelId           string `json:"kernelId"`
	Name               string `json:"name"`
	OsId               string `json:"osId"`
	OsName             string `json:"osName"`
	Latform            string `json:"platform"`
	RamdiskId          string `json:"ramdiskId"`
	RootDeviceName     string `json:"rootDeviceName"`
	RootDeviceType     string `json:"rootDeviceType"`
	ScheduleTags       string `json:"scheduleTags"`
	Shared             string `json:"shared"`
	Sharemode          string `json:"sharemode"`
	StateReason        string `json:"stateReason"`
	StorageId          string `json:"storageId"`
}

func (self *SRegion) GetImages() ([]SImage, error) {
	resp, err := self.invoke("DescribeImages", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		ImagesSet struct {
			Item []SImage
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.ImagesSet.Item, nil
}

func (self *SRegion) GetImage(id string) (*SImage, error) {
	image := &SImage{}
	return image, cloudprovider.ErrNotImplemented
}
