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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ScalingGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ScalingGroupDeleteTask{})
}

func (self *ScalingGroupDeleteTask) taskFailed(ctx context.Context, sg *models.SScalingGroup, reason jsonutils.JSONObject) {
	log.Errorf("scaling group delete task fail: %s", reason)
	sg.SetStatus(ctx, self.UserCred, api.SG_STATUS_DELETE_FAILED, reason.String())
	db.OpsLog.LogEvent(sg, db.ACT_DELETE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *ScalingGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sg := obj.(*models.SScalingGroup)

	sg.SetStatus(ctx, self.UserCred, api.SG_STATUS_DELETING, "")
	// Set all scaling policy's status as deleting
	sps, err := sg.ScalingPolicies()
	if err != nil {
		self.taskFailed(ctx, sg, jsonutils.NewString(err.Error()))
		return
	}
	spids := make([]string, len(sps))
	for i := range sps {
		spids[i] = sps[i].GetId()
		err := func() error {
			lockman.LockObject(ctx, &sps[i])
			defer lockman.ReleaseObject(ctx, &sps[i])
			return sps[i].SetStatus(ctx, self.UserCred, api.SP_STATUS_DELETING, "delete scaling group")
		}()
		if err != nil {
			self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("set scaling policy %s as deleting status failed: %s",
				sps[i].GetId(), err)))
			return
		}
	}

	log.Debugf("finish to mark all scaling policies deleted")
	// wait for activites finished
	sg.SetStatus(ctx, self.UserCred, api.SG_STATUS_WAIT_ACTIVITY_OVER, "wait all activities over")
	waitSeconds, interval, seconds := 180, 5, 0
	checkids := spids
	allReady := false
	for seconds < waitSeconds {
		// onlyread, lock no need
		tmpIds, err := models.ScalingActivityManager.FetchByStatus(ctx, checkids, []string{api.SA_STATUS_SUCCEED,
			api.SA_STATUS_FAILED}, "not")
		log.Debugf("scalingactivities not in 'succeed' or 'failed': %v", tmpIds)
		if err != nil {
			log.Errorf("ScalingActivityManager.FetchByStatus: %s", err.Error())
			continue
		}
		if len(tmpIds) == 0 {
			allReady = true
			break
		}
		checkids = tmpIds
		seconds += interval
		time.Sleep(time.Duration(interval) * time.Second)
	}
	if err != nil {
		self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("wait for all scaling activities finished: %s", err)))
		return
	}
	if !allReady {
		self.taskFailed(ctx, sg, jsonutils.NewString("some scaling activities are still in progress"))
		return
	}

	count, err := sg.GuestNumber()
	if err != nil {
		self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("SScalingGroup.GuestNumber: %s", err)))
		return
	}
	if count != 0 {
		self.taskFailed(ctx, sg, jsonutils.NewString("There are some guests in ScalingGroup, please delete them firstly"))
		return
	}

	// delete SScalingPolicies
	policies, err := sg.ScalingPolicies()
	if err != nil {
		self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("SScalingGroup.ScalingPolicies: %s", err.Error())))
		return
	}
	for i := range policies {
		err := policies[i].RealDelete(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("delete scaling group '%s' failed: %s", policies[i].GetId(), err.Error())))
			return
		}
	}

	// delete SScalingAvtivities
	activities, err := sg.Activities()
	if err != nil {
		self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("ScalingGroup.Activities: %s", err.Error())))
	}
	for i := range activities {
		err := activities[i].Delete(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("delete scaling activity '%s' failed: %s", activities[i].Id, err.Error())))
			return
		}
	}

	err = sg.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, sg, jsonutils.NewString(fmt.Sprintf("ScalingGroup.RealDelete: %s", err.Error())))
	}
	db.OpsLog.LogEvent(sg, db.ACT_DELETE, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_DELETE, "", self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    sg,
		Action: notifyclient.ActionDelete,
	})
}
