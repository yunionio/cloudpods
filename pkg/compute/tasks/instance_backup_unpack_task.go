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

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceBackupUnpackTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceBackupUnpackTask{})
}

func (self *InstanceBackupUnpackTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_CREATE_FROM_PACKAGE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_UNPACK, reason, self.UserCred, false)
	db.OpsLog.LogEvent(ib, db.ACT_UNPACK_FAIL, ib.GetShortDesc(ctx), self.GetUserCred())
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceBackupUnpackTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup) {
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_UNPACK, nil, self.UserCred, true)
	db.OpsLog.LogEvent(ib, db.ACT_UNPACK, ib.GetShortDesc(ctx), self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceBackupUnpackTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_CREATING_FROM_PACKAGE, "")
	packageName, _ := self.GetParams().GetString("package_name")
	// fetch metadata first
	self.SetStage("OnUnpackMetadata", nil)
	err := ib.GetRegionDriver().RequestUnpackInstanceBackup(ctx, ib, self, packageName, true)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceBackupUnpackTask) OnUnpackMetadata(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	metadata := &compute.InstanceBackupPackMetadata{}
	err := data.Unmarshal(metadata, "metadata")
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	_, err = ib.FillFromPackMetadata(ctx, self.GetUserCred(), nil, metadata)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	packageName, _ := self.GetParams().GetString("package_name")
	// fetch full backup info, including disks
	self.SetStage("OnUnpackComplete", nil)
	err = ib.GetRegionDriver().RequestUnpackInstanceBackup(ctx, ib, self, packageName, false)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceBackupUnpackTask) OnUnpackMetadataFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}

func (self *InstanceBackupUnpackTask) OnUnpackComplete(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	diskBackupIds := make([]string, 0)
	err := data.Unmarshal(&diskBackupIds, "disk_backup_ids")
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	metadata := &compute.InstanceBackupPackMetadata{}
	err = data.Unmarshal(metadata, "metadata")
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	_, err = ib.FillFromPackMetadata(ctx, self.GetUserCred(), diskBackupIds, metadata)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	self.taskSuccess(ctx, ib)
}

func (self *InstanceBackupUnpackTask) OnUnpackCompleteFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}
