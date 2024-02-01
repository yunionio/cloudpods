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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotpolicyUnbindDisksTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotpolicyUnbindDisksTask{})
}

func (self *SnapshotpolicyUnbindDisksTask) taskFailed(ctx context.Context, sp *models.SSnapshotPolicy, err error) {
	sp.SetStatus(ctx, self.UserCred, api.SNAPSHOT_POLICY_CANCEL_FAILED, err.Error())
	db.OpsLog.LogEvent(sp, db.ACT_UNBIND, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_UNBIND, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SnapshotpolicyUnbindDisksTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sp := obj.(*models.SSnapshotPolicy)

	region, err := sp.GetRegion()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrapf(err, "GetRegion"))
		return
	}

	diskIds := jsonutils.GetQueryStringArray(self.Params, "disk_ids")

	self.SetStage("OnSnapshotPolicyUnbindDisksComplete", nil)
	err = region.GetDriver().RequestSnapshotPolicyUnbindDisks(ctx, self.UserCred, sp, diskIds, self)
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrapf(err, "RequestSnapshotPolicyUnbindDisks"))
		return
	}
}

func (self *SnapshotpolicyUnbindDisksTask) OnSnapshotPolicyUnbindDisksComplete(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	sp.SetStatus(ctx, self.UserCred, apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotpolicyUnbindDisksTask) OnSnapshotPolicyUnbindDisksCompleteFailed(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	self.taskFailed(ctx, sp, errors.Errorf(data.String()))
}
