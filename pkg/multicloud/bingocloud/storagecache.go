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
	"github.com/coredns/coredns/plugin/pkg/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SStragecache struct {
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
}

func (self *SRegion) GetStoragecaches() ([]SStragecache, error) {
	resp, err := self.invoke("DescribeStorageInfo", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		StorageSet struct {
			Item []SStragecache
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetStoragecache(id string) (*SStragecache, error) {
	snapshot := &SStragecache{}
	return snapshot, cloudprovider.ErrNotImplemented
}
