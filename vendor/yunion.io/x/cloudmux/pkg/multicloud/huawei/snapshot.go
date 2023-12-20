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

package huawei

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

/*
限制：
https://support.huaweicloud.com/api-evs/zh-cn_topic_0058762427.html
1. 从快照创建云硬盘时，volume_type字段必须和快照源云硬盘保持一致。
2. 当指定的云硬盘类型在avaliability_zone内不存在时，则创建云硬盘失败。
*/

type SnapshotStatusType string

const (
	SnapshotStatusCreating      SnapshotStatusType = "creating"
	SnapshotStatusAvailable     SnapshotStatusType = "available"      // 云硬盘快照创建成功，可以使用。
	SnapshotStatusError         SnapshotStatusType = "error"          // 云硬盘快照在创建过程中出现错误。
	SnapshotStatusDeleting      SnapshotStatusType = "deleting"       //   云硬盘快照处于正在删除的过程中。
	SnapshotStatusErrorDeleting SnapshotStatusType = "error_deleting" //    云硬盘快照在删除过程中出现错误
	SnapshotStatusRollbacking   SnapshotStatusType = "rollbacking"    // 云硬盘快照处于正在回滚数据的过程中。
	SnapshotStatusBackingUp     SnapshotStatusType = "backing-up"     //  通过快照创建备份，快照状态就会变为backing-up
)

type Metadata struct {
	SystemEnableActive string `json:"__system__enableActive"` // 如果为true。则表明是系统盘快照
}

type SSnapshot struct {
	multicloud.SResourceBase
	HuaweiTags
	region *SRegion

	Metadata                              Metadata `json:"metadata"`
	CreatedAt                             string   `json:"created_at"`
	Description                           string   `json:"description"`
	ID                                    string   `json:"id"`
	Name                                  string   `json:"name"`
	OSExtendedSnapshotAttributesProgress  string   `json:"os-extended-snapshot-attributes:progress"`
	OSExtendedSnapshotAttributesProjectID string   `json:"os-extended-snapshot-attributes:project_id"`
	Size                                  int32    `json:"size"` // GB
	Status                                string   `json:"status"`
	UpdatedAt                             string   `json:"updated_at"`
	VolumeID                              string   `json:"volume_id"`
}

func (self *SSnapshot) GetId() string {
	return self.ID
}

func (self *SSnapshot) GetName() string {
	return self.Name
}

func (self *SSnapshot) GetGlobalId() string {
	return self.ID
}

func (self *SSnapshot) GetStatus() string {
	switch SnapshotStatusType(self.Status) {
	case SnapshotStatusAvailable:
		return api.SNAPSHOT_READY
	case SnapshotStatusCreating:
		return api.SNAPSHOT_CREATING
	case SnapshotStatusDeleting:
		return api.SNAPSHOT_DELETING
	case SnapshotStatusErrorDeleting, SnapshotStatusError:
		return api.SNAPSHOT_FAILED
	case SnapshotStatusRollbacking:
		return api.SNAPSHOT_ROLLBACKING
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshot(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.Size * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.VolumeID
}

func (self *SSnapshot) GetDiskType() string {
	if self.Metadata.SystemEnableActive == "true" {
		return api.DISK_TYPE_SYS
	} else {
		return api.DISK_TYPE_DATA
	}
}

func (self *SSnapshot) Delete() error {
	if self.region == nil {
		return fmt.Errorf("not init region for snapshot %s", self.GetId())
	}
	return self.region.DeleteSnapshot(self.GetId())
}

func (self *SRegion) GetSnapshots(diskId string, snapshotName string) ([]SSnapshot, error) {
	params := url.Values{}
	if len(diskId) > 0 {
		params.Set("volume_id", diskId)
	}

	if len(snapshotName) > 0 {
		params.Set("name", snapshotName)
	}
	ret := []SSnapshot{}
	for {
		resp, err := self.list(SERVICE_EVS, "cloudsnapshots/detail", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Snapshots []SSnapshot
			Count     int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Snapshots...)
		if len(ret) >= part.Count || len(part.Snapshots) == 0 {
			break
		}
		params.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	resp, err := self.list(SERVICE_EVS, "cloudsnapshots/"+id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SSnapshot{region: self}
	err = resp.Unmarshal(ret, "snapshot")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) DeleteSnapshot(id string) error {
	_, err := self.delete(SERVICE_EVS, "cloudsnapshots/"+id)
	return err
}

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]interface{}{
		"name":        name,
		"description": desc,
		"volume_id":   diskId,
		"force":       true,
	}
	resp, err := self.post(SERVICE_EVS, "cloudsnapshots", map[string]interface{}{"snapshot": params})
	if err != nil {
		return nil, err
	}
	ret := &SSnapshot{region: self}
	err = resp.Unmarshal(ret, "snapshot")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}
