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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDetachScalingGroupTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestDetachScalingGroupTask{})
}

func (self *GuestDetachScalingGroupTask) taskFailed(ctx context.Context, sgg *models.SScalingGroupGuest, sg *models.SScalingGroup, reason string) {
	if sgg == nil {
		return
	}
	sgg.SetGuestStatus(api.SG_GUEST_STATUS_REMOVE_FAILED)
	if sg == nil {
		model, _ := models.ScalingGroupManager.FetchById(sgg.ScalingGroupId)
		if model == nil {
			return
		}
		sg = model.(*models.SScalingGroup)
	}
	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_REMOVE_GUEST, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestDetachScalingGroupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	sgId, _ := self.Params.GetString("scaling_group")

	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	self.SetStage("OnDetachLoadbalancerComplete", nil)
	// 进行一些准备工作, 比如从负载均衡组移除，从数据库组移除
	q := models.LoadbalancerBackendManager.Query().Equals("backend_id", guest.Id)
	var lbBackend models.SLoadbalancerBackend
	err := q.First(&lbBackend)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			self.taskFailed(ctx, sgg, nil, fmt.Sprintf("Fetch loadbalancer backend failed: %s", err.Error()))
			return
		}
		self.OnDetachLoadbalancerComplete(ctx, guest, body)
		return
	} else {
		lbBackend.SetStatus(self.UserCred, api.LB_STATUS_DELETING, "")
		if err = lbBackend.StartLoadBalancerBackendDeleteTask(ctx, self.UserCred, jsonutils.NewDict(), self.Id); err != nil {
			self.taskFailed(ctx, sgg, nil, fmt.Sprintf("Detach guest with loadbalancer group failed: ", err))
		}
	}
}

func (self *GuestDetachScalingGroupTask) OnDetachLoadbalancerComplete(ctx context.Context, guest *models.SGuest,
	data jsonutils.JSONObject) {
	sgId, _ := self.Params.GetString("scaling_group")
	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	delete, _ := self.Params.Bool("delete_server")
	self.SetStage("OnDeleteGuestComplete", nil)
	if !delete {
		self.OnDeleteGuestComplete(ctx, guest, data)
		return
	}
	if err := guest.StartDeleteGuestTask(ctx, self.UserCred, self.Id, true, true, true); err != nil {
		self.taskFailed(ctx, sgg, nil, err.Error())
	}
}

func (self *GuestDetachScalingGroupTask) OnDetachLoadbalancerCompleteFailed(ctx context.Context, guest *models.SGuest,
	data jsonutils.JSONObject) {
	sgId, _ := self.Params.GetString("scaling_group")

	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	self.taskFailed(ctx, sgg, nil, "detach loadbalancer failed")
}

func (self *GuestDetachScalingGroupTask) OnDeleteGuestComplete(ctx context.Context, guest *models.SGuest,
	data jsonutils.JSONObject) {

	sgId, _ := self.Params.GetString("scaling_group")

	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sggs[0].Detach(ctx, self.UserCred)
	}

	model, err := models.ScalingGroupManager.FetchById(sgId)
	if err != nil {
		log.Errorf("ScalingGroupManager.FetchById failed: %s", err.Error())
		return
	}
	logclient.AddActionLogWithStartable(self, model, logclient.ACT_REMOVE_GUEST, fmt.Sprintf("Instance '%s' was removed", guest.Id), self.UserCred, true)
	if auto, err := self.Params.Bool("auto"); err != nil && auto {
		// scale; change the desire number
		err := model.(*models.SScalingGroup).Scale(ctx, SScalingTriggerDesc{guest.Name}, SScalingActionDesc{})
		if err != nil {
			log.Errorf("ScalingGroup '%s' scale after removing instance '%s' failed: %s", model.GetId(), guest.Id, err.Error())
		}
	}
}

type SScalingTriggerDesc struct {
	Guest string
}

func (s SScalingTriggerDesc) TriggerDescription() string {
	return fmt.Sprintf("A user remove instance '%s' from the scaling group manually", s.Guest)
}

type SScalingActionDesc struct {
}

func (s SScalingActionDesc) Exec(desire int) int {
	if desire < 1 {
		log.Errorf("desire should not be less than 1")
	}
	return desire - 1
}

func (self *GuestDetachScalingGroupTask) OnDeleteGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	sgId, _ := self.Params.GetString("scaling_group")

	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	self.taskFailed(ctx, sgg, nil, "delete guest failed")
}
