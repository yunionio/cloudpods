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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerLoadbalancerBackendGroupCreateTask struct {
	taskman.STask
}

type HuaweiLoadbalancerLoadbalancerBackendGroupCreateTask struct {
	LoadbalancerLoadbalancerBackendGroupCreateTask
}

type AwsLoadbalancerLoadbalancerBackendGroupCreateTask struct {
	LoadbalancerLoadbalancerBackendGroupCreateTask
}

func init() {
	taskman.RegisterTask(LoadbalancerLoadbalancerBackendGroupCreateTask{})
	taskman.RegisterTask(HuaweiLoadbalancerLoadbalancerBackendGroupCreateTask{})
	taskman.RegisterTask(AwsLoadbalancerLoadbalancerBackendGroupCreateTask{})
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerBackendGroup, reason string) {
	lbacl.SetStatus(self.GetUserCred(), api.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lbacl, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbacl.Id, lbacl.Name, api.LB_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region := lbbg.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbbg, fmt.Sprintf("failed to find region for lb backendgroup %s", lbbg.Name))
		return
	}
	backends := []cloudprovider.SLoadbalancerBackend{}
	self.GetParams().Unmarshal(&backends, "backends")
	self.SetStage("OnLoadbalancerBackendGroupCreateComplete", nil)

	if err := region.GetDriver().RequestCreateLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, backends, self); err != nil {
		self.taskFail(ctx, lbbg, err.Error())
	}
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnLoadbalancerBackendGroupCreateComplete(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, data jsonutils.JSONObject) {
	lbbg.SetStatus(self.GetUserCred(), api.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbbg, db.ACT_ALLOCATE, lbbg.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnLoadbalancerBackendGroupCreateCompleteFailed(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbbg, reason.String())
}

func (self *HuaweiLoadbalancerLoadbalancerBackendGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region := lbbg.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbbg, fmt.Sprintf("failed to find region for lb backendgroup %s", lbbg.Name))
		return
	}

	backends, err := lbbg.GetBackendsParams()
	if err != nil {
		self.taskFail(ctx, lbbg, err.Error())
		return
	}

	// 必须指定listenerId或ruleId
	listenerId, _ := self.GetParams().GetString("listenerId")
	ruleId, _ := self.GetParams().GetString("ruleId")
	if len(listenerId) == 0 && len(ruleId) == 0 {
		self.taskFail(ctx, lbbg, fmt.Sprintf("CreateLoadbalancerBackendGroup listener/rule id should not be emtpy"))
		return
	}

	self.SetStage("OnLoadbalancerBackendGroupCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, backends, self); err != nil {
		self.taskFail(ctx, lbbg, err.Error())
	}
}

func (self *AwsLoadbalancerLoadbalancerBackendGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region := lbbg.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbbg, fmt.Sprintf("failed to find region for lb backendgroup %s", lbbg.Name))
		return
	}

	backends, err := lbbg.GetBackendsParams()
	if err != nil {
		self.taskFail(ctx, lbbg, err.Error())
		return
	}

	// 必须指定listenerId
	listenerId, _ := self.GetParams().GetString("listener_id")
	if len(listenerId) == 0 {
		self.taskFail(ctx, lbbg, fmt.Sprintf("CreateLoadbalancerBackendGroup listener id should not be emtpy"))
		return
	}

	self.SetStage("OnLoadbalancerBackendGroupCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, backends, self); err != nil {
		self.taskFail(ctx, lbbg, err.Error())
	}
}
