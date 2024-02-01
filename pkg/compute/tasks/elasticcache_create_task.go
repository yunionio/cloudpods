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
	"yunion.io/x/pkg/errors"

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

func (self *ElasticcacheCreateTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, err error) {
	elasticcache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(elasticcache, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    elasticcache,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ElasticcacheCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	elasticcache := obj.(*models.SElasticcache)
	region, err := elasticcache.GetRegion()
	if err != nil {
		self.taskFail(ctx, elasticcache, errors.Wrapf(err, "GetRegion"))
		return
	}

	// sync security group here
	self.SetStage("OnSyncSecurityGroupComplete", data.(*jsonutils.JSONDict))
	if region.GetDriver().IsSupportedElasticcacheSecgroup() {
		secgroups := []string{}
		self.GetParams().Unmarshal(&secgroups, "secgroup_ids")
		secgroupInput := api.ElasticcacheSecgroupsInput{SecgroupIds: secgroups}
		_, err = elasticcache.ProcessElasticcacheSecgroupsInput(ctx, self.UserCred, "set", &secgroupInput)
		if err != nil {
			self.taskFail(ctx, elasticcache, errors.Wrapf(err, "ProcessElasticcacheSecgroupsInput"))
			return
		}

		if err := region.GetDriver().RequestSyncSecgroupsForElasticcache(ctx, self.UserCred, elasticcache, self); err != nil {
			self.taskFail(ctx, elasticcache, errors.Wrapf(err, "RequestSyncSecgroupsForElasticcache"))
			return
		}
	} else {
		self.OnSyncSecurityGroupComplete(ctx, elasticcache, data)
	}
}

func (self *ElasticcacheCreateTask) OnSyncSecurityGroupComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	region, err := elasticcache.GetRegion()
	if err != nil {
		self.taskFail(ctx, elasticcache, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnElasticcacheCreateComplete", nil)
	err = region.GetDriver().RequestCreateElasticcache(ctx, self.GetUserCred(), elasticcache, self, data.(*jsonutils.JSONDict))
	if err != nil {
		self.taskFail(ctx, elasticcache, errors.Wrapf(err, "RequestCreateElasticcache"))
		return
	}
}

func (self *ElasticcacheCreateTask) OnSyncSecurityGroupCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, fmt.Errorf(reason.String()))
}

func (self *ElasticcacheCreateTask) OnElasticcacheCreateComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	elasticcache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RUNNING, "")
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_CREATE, "", self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    elasticcache,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheCreateTask) OnElasticcacheCreateCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, fmt.Errorf(reason.String()))
}
