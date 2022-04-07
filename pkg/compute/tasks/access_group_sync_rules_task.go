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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AccessGroupSyncRulesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AccessGroupSyncRulesTask{})
}

func (self *AccessGroupSyncRulesTask) taskFail(ctx context.Context, ag *models.SAccessGroup, err error) {
	ag.SetStatus(self.GetUserCred(), api.ACCESS_GROUP_STATUS_AVAILABLE, err.Error())
	logclient.AddActionLogWithStartable(self, ag, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AccessGroupSyncRulesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ag := obj.(*models.SAccessGroup)

	caches, err := ag.GetAccessGroupCaches()
	if err != nil {
		self.taskFail(ctx, ag, errors.Wrapf(err, "ag.GetAccessGroupCaches"))
		return
	}

	for i := range caches {
		err := caches[i].SyncRules(ctx)
		if err != nil {
			caches[i].SetStatus(self.GetUserCred(), api.ACCESS_GROUP_STATUS_SYNC_RULES_FAILED, err.Error())
		}
	}

	ag.SetStatus(self.GetUserCred(), api.ACCESS_GROUP_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
