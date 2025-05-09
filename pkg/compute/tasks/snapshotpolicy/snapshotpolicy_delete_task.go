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

package snapshotpolicy

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotPolicyDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotPolicyDeleteTask{})
}

func (self *SnapshotPolicyDeleteTask) taskFail(ctx context.Context, sp *models.SSnapshotPolicy, err error) {
	sp.SetStatus(ctx, self.GetUserCred(), apis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(sp, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_DELOCATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, sp.Id, sp.Name, apis.STATUS_DELETE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SnapshotPolicyDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sp := obj.(*models.SSnapshotPolicy)

	region, err := sp.GetRegion()
	if err != nil {
		self.taskFail(ctx, sp, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnSnapshotPolicyDeleteComplete", nil)
	err = region.GetDriver().RequestDeleteSnapshotPolicy(ctx, self.UserCred, region, sp, self)
	if err != nil {
		self.taskFail(ctx, sp, errors.Wrapf(err, "RequestDeleteSnapshotPolicy"))
		return
	}
}

func (self *SnapshotPolicyDeleteTask) OnSnapshotPolicyDeleteComplete(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(sp, db.ACT_DELETE, sp.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    sp,
		Action: notifyclient.ActionDelete,
	})
	sp.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyDeleteTask) OnSnapshotPolicyDeleteCompleteFailed(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	self.taskFail(ctx, sp, errors.Errorf(data.String()))
}
