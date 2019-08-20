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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SnapshotCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotCreateTask{})
}

func (self *SnapshotCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	self.DoDiskSnapshot(ctx, snapshot)
}

func (self *SnapshotCreateTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason string) {
	snapshot.SetStatus(self.UserCred, api.SNAPSHOT_FAILED, reason)
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_FAIL, reason, self.UserCred)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotCreateTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshot.SetStatus(self.UserCred, api.SNAPSHOT_READY, "")
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DONE, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotCreateTask) DoDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	self.SetStage("OnCreateSnapshot", nil)
	if err := snapshot.GetRegionDriver().RequestCreateSnapshot(ctx, snapshot, self); err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotCreateTask) OnCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotCreateTask) OnCreateSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data.String())
}
