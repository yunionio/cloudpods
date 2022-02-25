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

type SSnapshot struct {
	BackupId     string `json:"backupId"`
	Description  string `json:"description"`
	DrMirrorId   string `json:"drMirrorId"`
	FileSize     string `json:"fileSize"`
	FileType     string `json:"fileType"`
	IsBackup     string `json:"isBackup"`
	IsHead       string `json:"isHead"`
	IsRoot       string `json:"isRoot"`
	OwnerId      string `json:"ownerId"`
	Progress     string `json:"progress"`
	SnapshotId   string `json:"snapshotId"`
	SnapshotName string `json:"snapshotName"`
	StartTime    string `json:"startTime"`
	Status       string `json:"status"`
	StorageId    string `json:"storageId"`
	VolumeId     string `json:"volumeId"`
	VolumeSize   string `json:"volumeSize"`
}

func (self *SRegion) GetSnapshots() ([]SSnapshot, error) {
	resp, err := self.invoke("DescribeSnapshots", nil)
	//	[ERROR] resp=:{"-xmlns":"http://ec2.amazonaws.com/doc/2009-08-15/","snapshotSet":{"NextToken":""}}
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		SnapshotSet struct {
			Item []SSnapshot
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	log.Errorf("result:", result)
	return result.SnapshotSet.Item, nil
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
