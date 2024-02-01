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

func (self *GuestDetachScalingGroupTask) taskFailed(ctx context.Context, sg *models.SScalingGroup, sgg *models.SScalingGroupGuest, reason jsonutils.JSONObject) {
	if sg == nil {
		return
	}
	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_REMOVE_GUEST, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
	if sgg == nil {
		guestId, _ := self.Params.GetString("guest")
		sggs, _ := models.ScalingGroupGuestManager.Fetch(sg.GetId(), guestId)
		if len(sggs) > 0 {
			sgg = &sggs[0]
		} else {
			return
		}
	}
	sgg.SetGuestStatus(api.SG_GUEST_STATUS_REMOVE_FAILED)
}

func (self *GuestDetachScalingGroupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sg := obj.(*models.SScalingGroup)
	guestId, _ := self.Params.GetString("guest")
	guest := models.GuestManager.FetchGuestById(guestId)
	if guest == nil {
		self.taskFailed(ctx, sg, nil, jsonutils.NewString("unable to FetchGuestById"))
		return
	}
	self.SetStage("OnDetachLoadbalancerComplete", nil)
	q := models.LoadbalancerBackendManager.Query().Equals("backend_id", guest.Id)
	var lbBackend models.SLoadbalancerBackend
	err := q.First(&lbBackend)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			self.taskFailed(ctx, sg, nil, jsonutils.NewString(fmt.Sprintf("Fetch loadbalancer backend failed: %s", err.Error())))
			return
		}
		self.OnDetachLoadbalancerComplete(ctx, sg, body)
		return
	} else {
		lbBackend.SetModelManager(models.LoadbalancerBackendManager, &lbBackend)
		lbBackend.SetStatus(ctx, self.UserCred, api.LB_STATUS_DELETING, "")
		if err = lbBackend.StartLoadBalancerBackendDeleteTask(ctx, self.UserCred, jsonutils.NewDict(), self.Id); err != nil {
			self.taskFailed(ctx, sg, nil, jsonutils.NewString(fmt.Sprintf("Detach guest with loadbalancer group failed: %s", err)))
		}
	}
}

func (self *GuestDetachScalingGroupTask) OnDetachLoadbalancerComplete(ctx context.Context, sg *models.SScalingGroup, data jsonutils.JSONObject) {
	guestId, _ := self.Params.GetString("guest")
	delete, _ := self.Params.Bool("delete_server")
	if !delete {
		self.OnDeleteGuestComplete(ctx, sg, data)
		return
	}
	guest := models.GuestManager.FetchGuestById(guestId)
	if guest == nil {
		self.taskFailed(ctx, sg, nil, jsonutils.NewString("unable to FetchGuestById"))
		return
	}
	self.Params.Set("guest_name", jsonutils.NewString(guest.GetName()))
	self.SetStage("OnDeleteGuestComplete", nil)
	opts := api.ServerDeleteInput{Purge: false, OverridePendingDelete: true, DeleteSnapshots: true}
	err := guest.StartDeleteGuestTask(ctx, self.UserCred, self.Id, opts)
	if err != nil {
		self.taskFailed(ctx, sg, nil, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestDetachScalingGroupTask) OnDetachLoadbalancerCompleteFailed(ctx context.Context, sg *models.SScalingGroup,
	data jsonutils.JSONObject) {
	self.taskFailed(ctx, sg, nil, data)
}

func (self *GuestDetachScalingGroupTask) OnDeleteGuestComplete(ctx context.Context, sg *models.SScalingGroup, data jsonutils.JSONObject) {
	guestId, _ := self.Params.GetString("guest")
	guestName, _ := self.Params.GetString("guest_name")

	sggs, _ := models.ScalingGroupGuestManager.Fetch(sg.GetId(), guestId)
	if len(sggs) > 0 {
		sggs[0].Detach(ctx, self.UserCred)
	}

	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_REMOVE_GUEST, fmt.Sprintf("Instance '%s' was removed", guestId), self.UserCred, true)
	if auto, _ := self.Params.Bool("auto"); !auto {
		// scale; change the desire number
		err := sg.Scale(ctx, SScalingTriggerDesc{guestName}, SScalingActionDesc{}, 0)
		if err != nil {
			log.Errorf("ScalingGroup '%s' scale after removing instance '%s' failed: %s", sg.GetId(), guestId, err.Error())
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

func (s SScalingActionDesc) CheckCoolTime() bool {
	return false
}

func (self *GuestDetachScalingGroupTask) OnDeleteGuestCompleteFailed(ctx context.Context, sg *models.SScalingGroup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, sg, nil, data)
}
