package aws

import "yunion.io/x/jsonutils"

type SnapshotStatusType string

const (
	SnapshotStatusAccomplished SnapshotStatusType = "accomplished"
	SnapshotStatusProgress     SnapshotStatusType = "progressing"
	SnapshotStatusFailed       SnapshotStatusType = "failed"
)

type SSnapshot struct {
	region *SRegion

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
	panic("implement me")
}

func (self *SSnapshot) GetName() string {
	panic("implement me")
}

func (self *SSnapshot) GetGlobalId() string {
	panic("implement me")
}

func (self *SSnapshot) GetStatus() string {
	panic("implement me")
}

func (self *SSnapshot) Refresh() error {
	panic("implement me")
}

func (self *SSnapshot) IsEmulated() bool {
	panic("implement me")
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SSnapshot) GetManagerId() string {
	panic("implement me")
}

func (self *SSnapshot) GetSize() int32 {
	panic("implement me")
}

func (self *SSnapshot) GetDiskId() string {
	panic("implement me")
}

func (self *SSnapshot) Delete() error {
	panic("implement me")
}

func (self *SSnapshot) GetRegionId() string {
	panic("implement me")
}

