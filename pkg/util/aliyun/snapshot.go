package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SnapshotStatusType string

const (
	SnapshotStatusAccoplished SnapshotStatusType = "accomplished"
	SnapshotStatusProgress    SnapshotStatusType = "progressing"
)

type SSnapshot struct {
	disk           *SDisk
	Progress       string
	SnapshotId     string
	SnapshotName   string
	SourceDiskId   string
	SourceDiskSize int32
	SourceDiskType string
	Status         SnapshotStatusType
	Usage          string
}

func (self *SSnapshot) GetId() string {
	return self.SnapshotId
}

func (self *SSnapshot) GetName() string {
	return self.SnapshotName
}

func (self *SSnapshot) GetStatus() string {
	return string(self.Status)
}

func (self *SSnapshot) Refresh() error {
	if snapshot, err := self.disk.getSnapshot(self.SnapshotId); err != nil {
		return err
	} else if err := jsonutils.Update(self, snapshot); err != nil {
		return err
	}
	return nil
}

func (self *SSnapshot) GetGlobalId() string {
	return fmt.Sprintf("%s", self.SnapshotId)
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	params := make(map[string]string)
	params["SnapshotId"] = snapshotId
	_, err := self.ecsRequest("DeleteSnapshot", params)
	return err
}

func (self *SSnapshot) Delete() error {
	if self.disk == nil {
		return fmt.Errorf("not init disk for snapshot %s", self.SnapshotId)
	}
	return self.disk.storage.zone.region.DeleteSnapshot(self.SnapshotId)
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}
