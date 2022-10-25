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

package hcs

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408624.html
type SSnapshot struct {
	multicloud.SResourceBase
	multicloud.HcsTags
	region *SRegion

	Metadata                              Metadata `json:"metadata"`
	CreatedAt                             string   `json:"created_at"`
	Description                           string   `json:"description"`
	Id                                    string   `json:"id"`
	Name                                  string   `json:"name"`
	OSExtendedSnapshotAttributesProgress  string   `json:"os-extended-snapshot-attributes:progress"`
	OSExtendedSnapshotAttributesProjectId string   `json:"os-extended-snapshot-attributes:project_id"`
	Size                                  int32    `json:"size"` // GB
	Status                                string   `json:"status"`
	UpdatedAt                             string   `json:"updated_at"`
	VolumeId                              string   `json:"volume_id"`
}

func (self *SSnapshot) GetId() string {
	return self.Id
}

func (self *SSnapshot) GetName() string {
	return self.Name
}

func (self *SSnapshot) GetGlobalId() string {
	return self.Id
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
	ret, err := self.region.GetSnapshot(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.Size * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.VolumeId
}

func (self *SSnapshot) GetDiskType() string {
	if self.Metadata.SystemEnableActive == "true" {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.Id)
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408627.html
func (self *SRegion) GetSnapshots(diskId string, name string) ([]SSnapshot, error) {
	params := url.Values{}
	if len(diskId) > 0 {
		params.Set("volume_id", diskId)
	}

	if len(name) > 0 {
		params.Set("name", name)
	}
	ret := []SSnapshot{}
	return ret, self.evsList("snapshots", params, &ret)
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	ret := &SSnapshot{region: self}
	res := fmt.Sprintf("snapshots/%s", id)
	return ret, self.evsGet(res, ret)
}

// 不能删除以autobk_snapshot_为前缀的快照。
// 当快照状态为available、error状态时，才可以删除。
func (self *SRegion) DeleteSnapshot(id string) error {
	res := fmt.Sprintf("snapshots/%s", id)
	return self.evsDelete(res)
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408624.html
// 目前已设置force字段。云硬盘处于挂载状态时，能强制创建快照。
func (self *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]interface{}{
		"snapshot": map[string]interface{}{
			"name":        name,
			"description": desc,
			"volume_id":   diskId,
			"force":       true,
		},
	}
	ret := &SSnapshot{region: self}
	return ret, self.evsCreate("snapshots", params, ret)
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetSnapshots")
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = self
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	ret, err := self.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
