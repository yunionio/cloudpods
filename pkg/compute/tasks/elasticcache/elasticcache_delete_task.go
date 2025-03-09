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

package elasticcache

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheDeleteTask{})
}

func (self *ElasticcacheDeleteTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	elasticcache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RELEASE_FAILED, reason.String())
	db.OpsLog.LogEvent(elasticcache, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_DELETE, reason, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    elasticcache,
		Action: notifyclient.ActionDelete,
		IsFail: true,
	})
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ec := obj.(*models.SElasticcache)
	region, _ := ec.GetRegion()
	if region == nil {
		self.taskFail(ctx, ec, jsonutils.NewString(fmt.Sprintf("failed to find region for elastic cache %s", ec.GetName())))
		return
	}

	self.SetStage("OnElasticcacheDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteElasticcache(ctx, self.GetUserCred(), ec, self); err != nil {
		self.taskFail(ctx, ec, jsonutils.NewString(err.Error()))
		return
	} else {
		ec.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RELEASED, "")
		ec.RealDelete(ctx, self.UserCred)
		logclient.AddActionLogWithStartable(self, ec, logclient.ACT_DELETE, "", self.UserCred, true)
		notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
			Obj:    ec,
			Action: notifyclient.ActionDelete,
		})
		self.SetStageComplete(ctx, nil)
	}
}
