package huawei

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
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
	snapshot, err := self.region.GetSnapshotById(self.GetId())
	if err != nil {
		return err
	}

	if err := jsonutils.Update(self, snapshot); err != nil {
		return err
	}

	return nil
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
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

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408627.html
func (self *SRegion) GetSnapshots(diskId string, snapshotName string) ([]SSnapshot, error) {
	params := make(map[string]string)

	if len(diskId) > 0 {
		params["volume_id"] = diskId
	}

	if len(snapshotName) > 0 {
		params["name"] = snapshotName
	}

	snapshots := make([]SSnapshot, 0)
	err := doListAllWithOffset(self.ecsClient.Snapshots.List, params, &snapshots)
	for i := range snapshots {
		snapshots[i].region = self
	}

	return snapshots, err
}

func (self *SRegion) GetSnapshotById(snapshotId string) (SSnapshot, error) {
	var snapshot SSnapshot
	err := DoGet(self.ecsClient.Snapshots.Get, snapshotId, nil, &snapshot)
	snapshot.region = self
	return snapshot, err
}

// 不能删除以autobk_snapshot_为前缀的快照。
// 当快照状态为available、error状态时，才可以删除。
func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	return DoDelete(self.ecsClient.Snapshots.Delete, snapshotId, nil, nil)
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408624.html
// 目前已设置force字段。云硬盘处于挂载状态时，能强制创建快照。
func (self *SRegion) CreateSnapshot(diskId, name, desc string) (string, error) {
	params := jsonutils.NewDict()
	snapshotObj := jsonutils.NewDict()
	snapshotObj.Add(jsonutils.NewString(name), "name")
	snapshotObj.Add(jsonutils.NewString(desc), "description")
	snapshotObj.Add(jsonutils.NewString(diskId), "volume_id")
	snapshotObj.Add(jsonutils.JSONTrue, "force")
	params.Add(snapshotObj, "snapshot")

	snapshot := SSnapshot{}
	err := DoCreate(self.ecsClient.Snapshots.Create, params, &snapshot)
	return snapshot.ID, err
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}
