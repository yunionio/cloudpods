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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotPolicyCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotPolicyCreateTask{})
}

func (self *SnapshotPolicyCreateTask) taskFailed(ctx context.Context, sp *models.SSnapshotPolicy, err error) {
	sp.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(sp, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SnapshotPolicyCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sp := obj.(*models.SSnapshotPolicy)

	region, err := sp.GetRegion()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnCreateSnapshotPolicyComplete", nil)
	err = region.GetDriver().RequestCreateSnapshotPolicy(ctx, self.UserCred, region, sp, self)
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrapf(err, "RequestCreateSnapshotPolicy"))
		return
	}
}

func (self *SnapshotPolicyCreateTask) OnCreateSnapshotPolicyComplete(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    sp,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyCreateTask) OnCreateSnapshotPolicyCompleteFailed(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	self.taskFailed(ctx, sp, errors.Errorf(data.String()))
}
