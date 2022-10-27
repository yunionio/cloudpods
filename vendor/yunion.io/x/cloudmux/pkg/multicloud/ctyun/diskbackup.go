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

package ctyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDiskBackup struct {
	multicloud.SResourceBase
	CtyunTags
	region *SRegion

	Status           string `json:"status"`
	Description      string `json:"description"`
	AvailabilityZone string `json:"availability_zone"`
	VolumeID         string `json:"volume_id"`
	FailReason       string `json:"fail_reason"`
	ID               string `json:"id"`
	Size             int32  `json:"size"`
	Container        string `json:"container"`
	Name             string `json:"name"`
	CreatedAt        string `json:"created_at"`
}

func (self *SDiskBackup) GetId() string {
	return self.ID
}

func (self *SDiskBackup) GetName() string {
	return self.Name
}

func (self *SDiskBackup) GetGlobalId() string {
	return self.GetId()
}

func (self *SDiskBackup) GetStatus() string {
	switch self.Status {
	case "available":
		return api.SNAPSHOT_READY
	case "creating":
		return api.SNAPSHOT_CREATING
	case "deleting":
		return api.SNAPSHOT_DELETING
	case "error_deleting", "error":
		return api.SNAPSHOT_FAILED
	case "rollbacking":
		return api.SNAPSHOT_ROLLBACKING
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (self *SDiskBackup) Refresh() error {
	snapshot, err := self.region.GetDiskBackup(self.VolumeID, self.GetId())
	if err != nil {
		return err
	}

	if err := jsonutils.Update(self, snapshot); err != nil {
		return err
	}

	return nil
}

func (self *SDiskBackup) IsEmulated() bool {
	return false
}

func (self *SDiskBackup) GetProjectId() string {
	return ""
}

func (self *SDiskBackup) GetSizeMb() int32 {
	return self.Size * 1024
}

func (self *SDiskBackup) GetDiskId() string {
	return self.VolumeID
}

func (self *SDiskBackup) GetDiskType() string {
	disk, err := self.region.GetDisk(self.VolumeID)
	if err != nil {
		log.Debugf("SDiskBackup.GetDiskType.GetDisk %s", err)
		return ""
	}

	return disk.GetDiskType()
}

func (self *SDiskBackup) Delete() error {
	return errors.ErrNotSupported
}

func (self *SRegion) GetDiskBackup(diskId string, backupId string) (*SDiskBackup, error) {
	backups, err := self.GetDiskBackups(diskId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetSnapshot.GetDiskBackups")
	}

	for i := range backups {
		backup := backups[i]
		if backup.ID == backupId || backup.Container == backupId {
			backup.region = self
			return &backup, nil
		}
	}

	return nil, errors.Wrap(errors.ErrNotFound, "SRegion.GetDiskBackup")
}

func (self *SRegion) GetDiskBackups(diskId string) ([]SDiskBackup, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	if len(diskId) > 0 {
		params["volumeId"] = diskId
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVBSDetails", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetDiskBackups.DoGet")
	}

	ret := make([]SDiskBackup, 0)
	err = resp.Unmarshal(&ret, "returnObj", "backups")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetDiskBackups.Unmarshal")
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

func (self *SRegion) DeleteDiskBackup(vbsId string) (string, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId": jsonutils.NewString(self.GetId()),
		"vbsId":    jsonutils.NewString(vbsId),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/deleteVBS", params)
	if err != nil {
		return "", errors.Wrap(err, "SRegion.DeleteDiskBackup.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj", "status")
	if !ok {
		msg, _ := resp.GetString("message")
		return "", fmt.Errorf("SRegion.DeleteDiskBackup.JobFailed %s", msg)
	}

	var jobId string
	err = resp.Unmarshal(&jobId, "returnObj", "data")
	if err != nil {
		return "", errors.Wrap(err, "SRegion.DeleteDiskBackup.Unmarshal")
	}

	return jobId, nil
}
