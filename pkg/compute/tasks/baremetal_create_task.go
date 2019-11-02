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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalCreateTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalCreateTask{})
}

func (self *BaremetalCreateTask) taskComplete(ctx context.Context, baremetal *models.SHost) {
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_ALLOCATE, baremetal.GetShortDesc(ctx), self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalCreateTask) taskFailed(ctx context.Context, baremetal *models.SHost, err string) {
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	self.SetStage("OnIpmiProbeComplete", nil)
	baremetal.StartIpmiProbeTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *BaremetalCreateTask) OnIpmiProbeComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	ipmiInfo, _ := baremetal.GetIpmiInfo()
	if !ipmiInfo.Verified {
		self.taskComplete(ctx, baremetal)
		return
	}
	if jsonutils.QueryBoolean(self.Params, "no_prepare", false) {
		self.taskComplete(ctx, baremetal)
		return
	}
	if (baremetal.EnablePxeBoot.IsFalse() || !ipmiInfo.PxeBoot) && !ipmiInfo.CdromBoot {
		self.taskComplete(ctx, baremetal)
		return
	}
	if baremetal.AccessMac == "" && baremetal.Uuid == "" && !ipmiInfo.CdromBoot {
		msg := "Fail to find access_mac or uuid, host-prepare aborted. Please supply either access_mac or uuid and try host-prepare"
		log.Errorf(msg)
		self.taskFailed(ctx, baremetal, msg)
		baremetal.SetStatus(self.UserCred, api.BAREMETAL_PREPARE_FAIL, msg)
		return
	}
	self.SetStage("OnPrepareComplete", nil)
	baremetal.StartPrepareTask(ctx, self.UserCred, "", self.GetTaskId())
}

func (self *BaremetalCreateTask) OnIpmiProbeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	self.taskFailed(ctx, baremetal, body.String())
}

func (self *BaremetalCreateTask) OnPrepareComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	self.taskComplete(ctx, baremetal)
}

func (self *BaremetalCreateTask) OnPrepareCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	self.taskFailed(ctx, baremetal, body.String())
}
