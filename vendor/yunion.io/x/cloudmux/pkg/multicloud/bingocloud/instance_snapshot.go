package bingocloud

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceSnapshot struct {
	region               *SRegion
	InstanceSnapshotId   string
	InstanceSnapshotName string
	InstanceId           string
	DiskOnly             bool
	Size                 int64
	Status               string
	StatusReason         string
	CreateTime           string
}

func (self SInstanceSnapshot) GetId() string {
	return self.InstanceSnapshotId
}

func (self SInstanceSnapshot) GetName() string {
	return self.InstanceSnapshotName
}

func (self SInstanceSnapshot) GetGlobalId() string {
	return self.InstanceSnapshotId
}

func (self SInstanceSnapshot) GetCreatedAt() time.Time {
	ct, _ := time.Parse("2006-01-02T15:04:05.000Z", self.CreateTime)
	return ct
}

func (self SInstanceSnapshot) GetStatus() string {
	return self.Status
}

func (self SInstanceSnapshot) Refresh() error {
	newSnapshot, err := self.region.getInstanceSnapshots(self.InstanceId, self.InstanceSnapshotId)
	if err != nil {
		return err
	}
	if len(newSnapshot) == 1 {
		return jsonutils.Update(self, &newSnapshot[0])
	}
	return cloudprovider.ErrNotFound
}

func (self SInstanceSnapshot) IsEmulated() bool {
	return false
}

func (self SInstanceSnapshot) GetSysTags() map[string]string {
	return nil
}

func (self SInstanceSnapshot) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self SInstanceSnapshot) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (self SInstanceSnapshot) GetProjectId() string {
	return ""
}

func (self SInstanceSnapshot) GetDescription() string {
	return ""
}

func (self SInstanceSnapshot) Delete() error {
	return self.region.deleteInstanceSnapshot(self.InstanceSnapshotId)
}

func (self *SRegion) createInstanceSnapshot(instanceId, name string, desc string) (string, error) {
	params := map[string]string{}
	params["InstanceId"] = instanceId
	params["InstanceSnapshotName"] = name
	params["Description"] = desc
	params["DiskOnly"] = "false"

	resp, err := self.client.invoke("CreateInstanceSnapshot", params)
	if err != nil {
		return "", err
	}
	newId := ""
	err = resp.Unmarshal(&newId, "instanceSnapshotId")

	return newId, err
}

func (self *SRegion) getInstanceSnapshots(instanceId, snapshotId string) ([]SInstanceSnapshot, error) {
	params := map[string]string{}
	if instanceId != "" {
		params["InstanceId"] = instanceId
	}
	if snapshotId != "" {
		params["InstanceSnapshotId.1"] = snapshotId
	}

	resp, err := self.client.invoke("DescribeInstanceSnapshots", params)
	if err != nil {
		return nil, err
	}

	var ret []SInstanceSnapshot
	_ = resp.Unmarshal(&ret, "instanceSnapshotSet")

	return ret, err
}

func (self *SRegion) deleteInstanceSnapshot(id string) error {
	params := map[string]string{}
	params["InstanceSnapshotId.1"] = id
	_, err := self.client.invoke("DeleteInstanceSnapshots", params)
	return err
}

func (self *SRegion) revertInstanceSnapshot(id string) error {
	params := map[string]string{}
	params["InstanceSnapshotId.1"] = id
	_, err := self.client.invoke("RevertInstanceSnapshot", params)
	return err
}

func (self *SRegion) deleteInstanceBackup(id string) error {
	params := map[string]string{}
	params["BackupId"] = id
	_, err := self.client.invoke("DeleteInstanceBackup", params)
	return err
}
