package aliyun

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SnapshotStatusType string

const (
	SnapshotStatusAccomplished SnapshotStatusType = "accomplished"
	SnapshotStatusProgress     SnapshotStatusType = "progressing"
	SnapshotStatusFailed       SnapshotStatusType = "failed"

	SnapshotTypeSystem string = "System"
	SnapshotTypeData   string = "Data"
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
	return self.SnapshotId
}

func (self *SSnapshot) GetName() string {
	return self.SnapshotName
}

func (self *SSnapshot) GetStatus() string {
	if self.Status == SnapshotStatusAccomplished {
		return api.SNAPSHOT_READY
	} else if self.Status == SnapshotStatusProgress {
		return api.SNAPSHOT_CREATING
	} else { // if self.Status == SnapshotStatusFailed
		return api.SNAPSHOT_FAILED
	}
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.SourceDiskSize * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.SourceDiskId
}

func (self *SSnapshot) GetDiskType() string {
	if self.SourceDiskType == SnapshotTypeSystem {
		return api.DISK_TYPE_SYS
	} else if self.SourceDiskType == SnapshotTypeData {
		return api.DISK_TYPE_DATA
	} else {
		return ""
	}
}

func (self *SSnapshot) Refresh() error {
	if snapshots, total, err := self.region.GetSnapshots("", "", "", []string{self.SnapshotId}, 0, 1); err != nil {
		return err
	} else if total != 1 {
		return cloudprovider.ErrNotFound
	} else if err := jsonutils.Update(self, snapshots[0]); err != nil {
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
	for i := 0; i < len(snapshots); i += 1 {
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SSnapshot) Delete() error {
	if self.region == nil {
		return fmt.Errorf("not init region for snapshot %s", self.SnapshotId)
	}
	if err := self.region.SnapshotPreDelete(self.SnapshotId); err != nil {
		return err
	}
	return self.region.DeleteSnapshot(self.SnapshotId)
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetSnapshots(instanceId string, diskId string, snapshotName string, snapshotIds []string, offset int, limit int) ([]SSnapshot, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}
	if len(diskId) > 0 {
		params["diskId"] = diskId
	}
	if len(snapshotName) > 0 {
		params["SnapshotName"] = snapshotName
	}
	if snapshotIds != nil && len(snapshotIds) > 0 {
		params["SnapshotIds"] = jsonutils.Marshal(snapshotIds).String()
	}

	body, err := self.ecsRequest("DescribeSnapshots", params)
	if err != nil {
		log.Errorf("GetSnapshots fail %s", err)
		return nil, 0, err
	}

	snapshots := make([]SSnapshot, 0)
	if err := body.Unmarshal(&snapshots, "Snapshots", "Snapshot"); err != nil {
		log.Errorf("Unmarshal snapshot details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = self
	}
	return snapshots, int(total), nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, total, err := self.GetSnapshots("", "", "", []string{snapshotId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	} else if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &snapshots[0], nil
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	params := make(map[string]string)
	params["SnapshotId"] = snapshotId
	_, err := self.ecsRequest("DeleteSnapshot", params)
	return err
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}

// If snapshot linked images can't be delete
// delete images first -- Aliyun
func (self *SRegion) SnapshotPreDelete(snapshotId string) error {
	images, _, err := self.GetImagesBySnapshot(snapshotId, 0, 0)
	if err != nil {
		return fmt.Errorf("PreDelete get images by snapshot %s error: %s", snapshotId, err)
	}
	for _, image := range images {
		image.storageCache = &SStoragecache{region: self}
		if err := image.Delete(context.Background()); err != nil {
			return fmt.Errorf("PreDelete image %s error: %s", image.GetId(), err)
		}
		if err := cloudprovider.WaitDeleted(&image, 3*time.Second, 300*time.Second); err != nil {
			return fmt.Errorf("PreDelete waite image %s deleted error: %s", image.GetId(), err)
		}
	}
	return nil
}
