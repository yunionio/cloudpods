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
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheCreateTask{})
}

func (self *ElasticcacheCreateTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	elasticcache.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(elasticcache, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    elasticcache,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	elasticcache := obj.(*models.SElasticcache)
	region, _ := elasticcache.GetRegion()
	if region == nil {
		self.taskFail(ctx, elasticcache, jsonutils.NewString(fmt.Sprintf("failed to find region for elastic cache %s", elasticcache.GetName())))
		return
	}

	// sync security group here
	self.SetStage("OnSyncSecurityGroupComplete", data.(*jsonutils.JSONDict))
	if region.GetDriver().IsSupportedElasticcacheSecgroup() {
		secgroups := []string{}
		err := self.GetParams().Unmarshal(&secgroups, "secgroup_ids")
		if err != nil {
			notes := fmt.Sprintf("Unmarshal.secgroup_ids %s", err)
			self.taskFail(ctx, elasticcache, jsonutils.NewString(notes))
			return
		}
		secgroupInput := api.ElasticcacheSecgroupsInput{SecgroupIds: secgroups}
		_, err = elasticcache.ProcessElasticcacheSecgroupsInput(ctx, self.UserCred, "set", &secgroupInput)
		if err != nil {
			notes := fmt.Sprintf("ProcessElasticcacheSecgroupsInput %s", err)
			self.taskFail(ctx, elasticcache, jsonutils.NewString(notes))
			return
		}

		if err := region.GetDriver().RequestSyncSecgroupsForElasticcache(ctx, self.UserCred, elasticcache, self); err != nil {
			self.taskFail(ctx, elasticcache, jsonutils.NewString(err.Error()))
			return
		}
	} else {
		self.OnSyncSecurityGroupComplete(ctx, elasticcache, data)
	}
}

func (self *ElasticcacheCreateTask) OnSyncSecurityGroupComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	region, _ := elasticcache.GetRegion()
	self.SetStage("OnElasticcacheCreateComplete", nil)
	if err := region.GetDriver().RequestCreateElasticcache(ctx, self.GetUserCred(), elasticcache, self, data.(*jsonutils.JSONDict)); err != nil {
		self.taskFail(ctx, elasticcache, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *ElasticcacheCreateTask) OnSyncSecurityGroupCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, reason)
}

func (self *ElasticcacheCreateTask) OnElasticcacheCreateComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	elasticcache.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RUNNING, "")
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_CREATE, "", self.UserCred, true)
	// notifyclient.NotifyWebhook(ctx, self.UserCred, elasticcache, notifyclient.ActionCreate)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    elasticcache,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheCreateTask) OnElasticcacheCreateCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, reason)
}
