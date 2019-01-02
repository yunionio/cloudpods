package huawei

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
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
		return models.SNAPSHOT_READY
	case SnapshotStatusCreating:
		return models.SNAPSHOT_CREATING
	case SnapshotStatusDeleting:
		return models.SNAPSHOT_DELETING
	case SnapshotStatusErrorDeleting, SnapshotStatusError:
		return models.SNAPSHOT_FAILED
	case SnapshotStatusRollbacking:
		return models.SNAPSHOT_ROLLBACKING
	default:
		return models.SNAPSHOT_UNKNOWN
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

func (self *SSnapshot) GetSize() int32 {
	return self.Size
}

func (self *SSnapshot) GetDiskId() string {
	return self.VolumeID
}

func (self *SSnapshot) GetDiskType() string {
	if self.Metadata.SystemEnableActive == "true" {
		return models.DISK_TYPE_SYS
	} else {
		return models.DISK_TYPE_DATA
	}
}

func (self *SSnapshot) Delete() error {
	if self.region == nil {
		return fmt.Errorf("not init region for snapshot %s", self.GetId())
	}
	return self.region.DeleteSnapshot(self.GetId())
}

func (self *SRegion) GetSnapshots(diskId string, snapshotName string, offset int, limit int) ([]SSnapshot, int, error) {
	params := make(map[string]string)
	params["limit"] = fmt.Sprintf("%d", limit)
	params["offset"] = fmt.Sprintf("%d", offset)

	if len(diskId) > 0 {
		params["volume_id"] = diskId
	}
	if len(snapshotName) > 0 {
		params["name"] = snapshotName
	}

	snapshots := make([]SSnapshot, 0)
	err := DoList(self.ecsClient.Snapshots.List, params, &snapshots)
	for i := range snapshots {
		snapshots[i].region = self
	}
	return snapshots, len(snapshots), err
}

func (self *SRegion) GetSnapshotById(snapshotId string) (SSnapshot, error) {
	var snapshot SSnapshot
	err := DoGet(self.ecsClient.Snapshots.Get, snapshotId, nil, &snapshot)
	snapshot.region = self
	return snapshot, err
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	// todo: implement me
	return nil
}
