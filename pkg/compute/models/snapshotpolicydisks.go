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

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSnapshotPolicyDiskManager struct {
	db.SVirtualJointResourceBaseManager
}

func (manager *SSnapshotPolicyDiskManager) GetMasterFieldName() string {
	return "disk_id"
}

func (manager *SSnapshotPolicyDiskManager) GetSlaveFieldName() string {
	return "snapshotpolicy_id"
}

var SnapshotPolicyDiskManager *SSnapshotPolicyDiskManager

func init() {
	db.InitManager(func() {
		SnapshotPolicyDiskManager = &SSnapshotPolicyDiskManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SSnapshotPolicyDisk{},
				"snapshotpolicydisks_tbl",
				"snapshotpolicydisk",
				"snapshotpolicydisks",
				DiskManager,
				SnapshotPolicyManager,
			),
		}
		SnapshotPolicyDiskManager.SetVirtualObject(SnapshotPolicyDiskManager)
	})

}

type SSnapshotPolicyDisk struct {
	db.SVirtualJointResourceBase

	SnapshotpolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	DiskId           string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (self *SSnapshotPolicyDisk) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (self *SSnapshotPolicyDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	diskId, _ := data.GetString(self.GetMasterFieldName())
	disk := DiskManager.FetchDiskById(diskId)
	err := disk.GetStorage().GetRegion().GetDriver().ValidateCreateSnapshopolicyDiskData(ctx, userCred, diskId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (self *SSnapshotPolicyDiskManager) FetchAllSnapshotPolicyOfDisk(ctx context.Context, userCred mcclient.TokenCredential, diskID string) ([]SSnapshotPolicyDisk, error) {
	q := self.Query()
	q.Equals(self.GetMasterFieldName(), diskID)
	ret := make([]SSnapshotPolicyDisk, 0)
	err := db.FetchModelObjects(self, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SSnapshotPolicyDisk) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	diskID := self.DiskId
	model, err := DiskManager.FetchById(diskID)
	if err != nil {
		log.Errorf("Fetch disk by ID %s failed", diskID)
	}
	disk := model.(*SDisk)
	snapshotPolicyID := self.SnapshotpolicyId
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(snapshotPolicyID), "snapshot_policy_id")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyApplyTask", disk, userCred, taskData, "", "", nil)
	if err != nil {
		log.Errorf("SnapshotPolicyApplyTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (self *SSnapshotPolicyDisk) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	diskID := self.DiskId
	model, err := DiskManager.FetchById(diskID)
	if err != nil {
		return errors.Wrapf(err, "Fetch disk by ID %s failed", diskID)
	}
	disk := model.(*SDisk)
	snapshotPolicyID := self.SnapshotpolicyId
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(snapshotPolicyID), "snapshot_policy_id")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCancelTask", disk, userCred, taskData, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "SnapshotPolicyCancelTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}
