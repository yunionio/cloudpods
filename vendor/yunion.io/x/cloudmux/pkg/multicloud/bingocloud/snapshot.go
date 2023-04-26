package bingocloud

import (
	"strconv"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SSnapshot struct {
	region       *SRegion
	SnapshotId   string
	SnapshotName string
	BackupId     string
	VolumeId     string
	Status       string
	StartTime    string
	Progress     string
	OwnerId      string
	VolumeSize   string
	Description  string
}

func (self SSnapshot) GetId() string {
	return self.SnapshotId
}

func (self SSnapshot) GetName() string {
	return self.SnapshotName
}

func (self SSnapshot) GetGlobalId() string {
	return self.SnapshotId
}

func (self SSnapshot) GetCreatedAt() time.Time {
	ct, _ := time.Parse("2006-01-02T15:04:05.000Z", self.StartTime)
	return ct
}

func (self SSnapshot) GetDescription() string {
	return self.Description
}

func (self SSnapshot) GetStatus() string {
	return self.Status
}

func (self SSnapshot) Refresh() error {
	newSnapshot, err := self.region.getSnapshots(self.SnapshotId, "")
	if err != nil {
		return err
	}
	if len(newSnapshot) == 1 {
		newSnapshot[0].region = self.region
		return jsonutils.Update(&self, newSnapshot[0])
	}
	return cloudprovider.ErrNotFound
}

func (self SSnapshot) IsEmulated() bool {
	return false
}

func (self SSnapshot) GetSysTags() map[string]string {
	return nil
}

func (self SSnapshot) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self SSnapshot) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (self SSnapshot) GetProjectId() string {
	return ""
}

func (self SSnapshot) GetSizeMb() int32 {
	size, _ := strconv.Atoi(self.VolumeSize)
	return int32(size * 1024)
}

func (self SSnapshot) GetDiskId() string {
	return self.VolumeId
}

func (self SSnapshot) GetDiskType() string {
	return ""
}

func (self SSnapshot) Delete() error {
	if self.BackupId != "" {
		return self.region.deleteInstanceBackup(self.BackupId)
	}
	return self.region.deleteSnapshot(self.SnapshotId)
}

func (self *SRegion) createSnapshot(volumeId, name string, desc string) (string, error) {
	params := map[string]string{}
	params["VolumeId"] = volumeId
	params["SnapshotName"] = name
	params["Description"] = desc

	resp, err := self.invoke("CreateSnapshot", params)
	if err != nil {
		return "", err
	}
	newId := ""
	err = resp.Unmarshal(&newId, "snapshotId")

	return newId, err
}

func (self *SRegion) getSnapshots(id, name string) ([]SSnapshot, error) {
	params := map[string]string{}
	if id != "" {
		params["SnapshotId.1"] = id
	}
	if name != "" {
		params["Filter.1.Name"] = name
	}

	resp, err := self.invoke("DescribeSnapshots", params)
	if err != nil {
		return nil, err
	}

	var ret []SSnapshot
	_ = resp.Unmarshal(&ret, "snapshotSet")

	return ret, err
}

func (self *SRegion) deleteSnapshot(id string) error {
	params := map[string]string{}
	params["SnapshotId"] = id
	_, err := self.invoke("DeleteSnapshot", params)
	return err
}
