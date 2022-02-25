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
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin/pkg/log"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SStoragecache struct {
	ClusterId    string `json:"clusterId"`
	ResUsage     string `json:"resUsage"`
	ParameterSet struct {
		Item struct {
			Name  string `json:name`
			Value string `json:value`
		}
	}
	StorageId    string `json:"storageId"`
	StorageName  string `json:"storageName"`
	UsedBy       string `json:"usedBy"`
	SpaceUsed    string `json:"spaceUsed"`
	SpaceMax     string `json:"spaceMax"`
	Location     string `json:"location"`
	FileFormat   string `json:"fileFormat"`
	Disabled     string `json:"disabled"`
	IsDRStorage  string `json:"isDRStorage"`
	DrCloudId    string `json:"drCloudId"`
	ScheduleTags string `json:"scheduleTags"`

	multicloud.SResourceBase
	region *SRegion
}

func (self *SRegion) GetStoragecaches() ([]SStoragecache, error) {
	resp, err := self.invoke("DescribeStorageInfo", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		StorageSet struct {
			Item []SStoragecache
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetStoragecache(id string) (*SStoragecache, error) {
	snapshot := &SStoragecache{}
	return snapshot, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.cpcfg.Id, cache.region.GetGlobalId())
}

// 私有云需要实现
func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// 公有云需要实现
func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, callback func(float32)) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetId() string {
	return self.StorageId
}

func (self *SStoragecache) GetName() string {
	return self.StorageName
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetSysTags() map[string]string {
	return nil
}

func (self *SStoragecache) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}
