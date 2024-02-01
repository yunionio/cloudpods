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
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotSyncstatusTask{})
}

func (self *SnapshotSyncstatusTask) taskFailed(ctx context.Context, snapshot *models.SSnapshot, err jsonutils.JSONObject) {
	snapshot.SetStatus(ctx, self.GetUserCred(), api.DISK_UNKNOWN, err.String())
	self.SetStageFailed(ctx, err)
	db.OpsLog.LogEvent(snapshot, db.ACT_SYNC_STATUS, snapshot.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, snapshot, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
}

func (self *SnapshotSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)

	region, err := snapshot.GetRegion()
	if err != nil {
		self.taskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
		return
	}

	self.SetStage("OnSnapshotSyncStatusComplete", nil)
	err = region.GetDriver().RequestSyncSnapshotStatus(ctx, self.GetUserCred(), snapshot, self)
	if err != nil {
		self.taskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *SnapshotSyncstatusTask) OnSnapshotSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotSyncstatusTask) OnSnapshotSyncStatusCompleteFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.taskFailed(ctx, snapshot, data)
}
