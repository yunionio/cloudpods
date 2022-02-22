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

type SSnapshot struct{}

func (self *SRegion) GetSnapshots() ([]SSnapshot, int, error) {
	resp, err := self.invoke("DescribeSnapshots", nil)
	if err != nil {
		return nil, 0, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct{}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, 0, err
	}
	return nil, 0, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshot := &SSnapshot{}
	return snapshot, cloudprovider.ErrNotImplemented
}

func (self *SSnapshot) GetSizeMb() int32 {
	return 0
}

func (self *SSnapshot) GetDiskId() string {
	return ""
}

func (self *SSnapshot) GetDiskType() string {
	return ""
}

func (self *SSnapshot) Delete() error {
	return cloudprovider.ErrNotFound
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}

func (self *SSnapshot) GetId() string {
	return ""
}

func (self *SSnapshot) GetName() string {
	return ""
}

func (self *SSnapshot) GetGlobalId() string {
	return ""
}

func (self *SSnapshot) GetStatus() string {
	return ""
}

func (self *SSnapshot) Refresh() error {
	return cloudprovider.ErrNotFound
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SSnapshot) GetSysTags() map[string]string {
	return nil
}

func (self *SSnapshot) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SSnapshot) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotFound
}
