package qcloud

import (
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SnapshotStatusType string

const (
	SnapshotStatusAccomplished SnapshotStatusType = "NORMAL"
	SnapshotStatusProgress     SnapshotStatusType = "CREATING"
	SnapshotStatusFailed       SnapshotStatusType = "failed"
)

type SSnapshot struct {
	region *SRegion

	SnapshotId       string             //	快照ID。
	Placement        Placement          //	快照所在的位置。
	DiskUsage        string             //	创建此快照的云硬盘类型。取值范围：SYSTEM_DISK：系统盘 DATA_DISK：数据盘。
	DiskId           string             //	创建此快照的云硬盘ID。
	DiskSize         int32              //	创建此快照的云硬盘大小，单位GB。
	SnapshotState    SnapshotStatusType //	快照的状态。取值范围： NORMAL：正常 CREATING：创建中 ROLLBACKING：回滚中 COPYING_FROM_REMOTE：跨地域复制快照拷贝中。
	SnapshotName     string             //	快照名称，用户自定义的快照别名。调用ModifySnapshotAttribute可修改此字段。
	Percent          int                //	快照创建进度百分比，快照创建成功后此字段恒为100。
	CreateTime       time.Time          //	快照的创建时间。
	DeadlineTime     time.Time          //	快照到期时间。如果快照为永久保留，此字段为空。
	Encrypt          bool               //	是否为加密盘创建的快照。取值范围：true：该快照为加密盘创建的 false:非加密盘创建的快照。
	IsPermanent      bool               //	是否为永久快照。取值范围： true：永久快照 false：非永久快照。
	CopyingToRegions []string           //	快照正在跨地域复制的目的地域，默认取值为[]。
	CopyFromRemote   bool               //	是否为跨地域复制的快照。取值范围：true：表示为跨地域复制的快照。 false:本地域的快照。
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, total, err := self.GetSnapshots("", "", "", []string{snapshotId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &snapshots[0], nil
}

func (self *SSnapshot) GetStatus() string {
	// NORMAL：正常
	// CREATING：创建中
	// ROLLBACKING：回滚中
	// COPYING_FROM_REMOTE：跨地域复制快照拷贝中。
	switch self.SnapshotState {
	case "NORMAL", "COPYING_FROM_REMOTE":
		return api.SNAPSHOT_READY
	case "CREATING":
		return api.SNAPSHOT_CREATING
	case "ROLLBACKING":
		return api.SNAPSHOT_ROLLBACKING
	}
	return api.SNAPSHOT_UNKNOWN
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SSnapshot) Refresh() error {
	snapshots, total, err := self.region.GetSnapshots("", "", "", []string{self.SnapshotId}, 0, 1)
	if err != nil {
		return err
	}
	if total > 1 {
		return cloudprovider.ErrDuplicateId
	}

	if total == 0 {
		return cloudprovider.ErrNotFound
	}
	return jsonutils.Update(self, snapshots[0])
}

func (self *SRegion) GetSnapshots(instanceId string, diskId string, snapshotName string, snapshotIds []string, offset int, limit int) ([]SSnapshot, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	filter := 0
	if len(instanceId) > 0 {
	}
	if len(diskId) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "disk-id"
		params[fmt.Sprintf("Filters.%d.Values", filter)] = diskId
		filter++
	}
	if len(snapshotName) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "snapshot-name"
		params[fmt.Sprintf("Filters.%d.Values", filter)] = snapshotName
		filter++
	}
	if snapshotIds != nil && len(snapshotIds) > 0 {
		for index, snapshotId := range snapshotIds {
			params[fmt.Sprintf("SnapshotIds.%d", index)] = snapshotId
		}
	}
	snapshots := []SSnapshot{}
	body, err := self.cbsRequest("DescribeSnapshots", params)
	if err != nil {
		log.Errorf("GetSnapshots fail %s", err)
		return nil, 0, err
	}
	body.Unmarshal(&snapshots, "SnapshotSet")
	if err != nil {
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self
	}
	return snapshots, int(total), nil
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, total, err := self.GetSnapshots("", "", "", []string{}, 0, 50)
	if err != nil {
		return nil, err
	}
	for len(snapshots) < total {
		var parts []SSnapshot
		parts, total, err = self.GetSnapshots("", "", "", []string{}, len(snapshots), 50)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, parts...)
	}
	ret := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSnapshot) GetRegionId() string {
	return self.region.GetId()
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.DiskSize * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.DiskId
}

func (self *SSnapshot) GetId() string {
	return self.SnapshotId
}

func (self *SSnapshot) GetGlobalId() string {
	return fmt.Sprintf("%s", self.SnapshotId)
}

func (self *SSnapshot) GetName() string {
	return self.SnapshotName
}

func (self *SSnapshot) Delete() error {
	if self.region == nil {
		return fmt.Errorf("not init region for snapshot %s", self.SnapshotId)
	}
	return self.region.DeleteSnapshot(self.SnapshotId)
}

func (self *SSnapshot) GetDiskType() string {
	switch self.DiskUsage {
	case "SYSTEM_DISK":
		return api.DISK_TYPE_SYS
	case "DATA_DISK":
		return api.DISK_TYPE_DATA
	}
	return api.DISK_TYPE_DATA
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	params := map[string]string{"SnapshotIds.0": snapshotId}
	_, err := self.cbsRequest("DeleteSnapshots", params)
	return err
}

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (string, error) {
	params := make(map[string]string)
	params["DiskId"] = diskId
	params["SnapshotName"] = name

	body, err := self.cbsRequest("CreateSnapshot", params)
	if err != nil {
		log.Errorf("CreateSnapshot fail %s", err)
		return "", err
	}
	return body.GetString("SnapshotId")
}

func (self *SSnapshot) GetProjectId() string {
	return strconv.Itoa(self.Placement.ProjectId)
}
