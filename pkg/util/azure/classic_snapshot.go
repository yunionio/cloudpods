package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClassicSnapshot struct {
	region *SRegion

	Name     string
	sizeMB   int32
	diskID   string
	diskName string
}

func (self *SClassicSnapshot) GetId() string {
	return fmt.Sprintf("%s?snapshot=%s", self.diskID, self.Name)
}

func (self *SClassicSnapshot) GetGlobalId() string {
	return self.GetId()
}

func (self *SClassicSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicSnapshot) GetName() string {
	return fmt.Sprintf("%s-%s", self.diskName, self.Name)
}

func (self *SClassicSnapshot) GetStatus() string {
	return api.SNAPSHOT_READY
}

func (self *SClassicSnapshot) IsEmulated() bool {
	return false
}

func (self *SRegion) CreateClassicSnapshot(diskId, snapName, desc string) (*SClassicSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicSnapshot) Delete() error {
	return self.region.DeleteClassicSnapshot(self.GetId())
}

func (self *SClassicSnapshot) GetSizeMb() int32 {
	return self.sizeMB
}

func (self *SRegion) DeleteClassicSnapshot(snapshotId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicSnapshot) Refresh() error {
	return nil
}

func (self *SClassicSnapshot) GetDiskId() string {
	return self.diskID
}

func (self *SClassicSnapshot) GetRegionId() string {
	return self.region.GetId()
}

func (self *SClassicSnapshot) GetDiskType() string {
	return ""
}

func (self *SClassicSnapshot) GetProjectId() string {
	return ""
}
