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

type SStorage struct {
	Disabled     bool   `json:"disabled"`
	DrCloudId    string `json:"drCloudId"`
	ParameterSet struct {
		Item struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"item"`
	} `json:"parameterSet"`
	StorageId    string `json:"storageId"`
	ClusterId    string `json:"clusterId"`
	UsedBy       string `json:"usedBy"`
	SpaceMax     string `json:"spaceMax"`
	IsDRStorage  string `json:"isDRStorage"`
	ScheduleTags string `json:"scheduleTags"`
	StorageName  string `json:"storageName"`
	Location     string `json:"location"`
	SpaceUsed    string `json:"spaceUsed"`
	StorageType  string `json:"storageType"`
	FileFormat   string `json:"fileFormat"`
	ResUsage     string `json:"resUsage"`
}

func (self *SRegion) GetStorages() ([]SStorage, error) {
	resp, err := self.invoke("DescribeStorages", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		StorageSet struct {
			Item []SStorage
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.StorageSet.Item, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	storage := &SStorage{}
	// return storage, self.get("storage_containers", id, nil, storage)
	return storage, cloudprovider.ErrNotImplemented
}
