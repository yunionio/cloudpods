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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ScalingGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ScalingGroupDeleteTask{})
}

func (self *ScalingGroupDeleteTask) taskFailed(ctx context.Context, sg *models.SScalingGroup, reason string) {
	log.Errorf("scaling group delete task fail: %s", reason)
	sg.SetStatus(self.UserCred, api.SG_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(sg, db.ACT_DELETE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *ScalingGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sg := obj.(*models.SScalingGroup)

	sg.SetStatus(self.UserCred, api.SG_STATUS_DELETING, "")
	// Set all scaling policy's status as deleting
	sps, err := sg.ScalingPolicies()
	if err != nil {
		self.taskFailed(ctx, sg, err.Error())
		return
	}
	spids := make([]string, len(sps))
	for i := range sps {
		spids[i] = sps[i].GetId()
		lockman.LockObject(ctx, &sps[i])
		err := sps[i].SetStatus(self.UserCred, api.SP_STATUS_DELETING, "delete scaling group")
		if err != nil {
			self.taskFailed(ctx, sg, fmt.Sprintf("set scaling policy %s as deleting status failed: %s",
				sps[i].GetId(), err))
			return
		}
		lockman.ReleaseObject(ctx, &sps[i])
	}

	log.Debugf("finish to mark all scaling policies deleted")
	// wait for activites finished
	sg.SetStatus(self.UserCred, api.SG_STATUS_WAIT_ACTIVITY_OVER, "wait all activities over")
	waitSeconds, interval, seconds := 180, 5, 0
	checkids := spids
	allReady := false
	for seconds < waitSeconds {
		// onlyread, lock no need
		tmpIds, err := models.ScalingActivityManager.FetchByStatus(ctx, checkids, []string{api.SA_STATUS_SUCCEED,
			api.SA_STATUS_FAILED}, "not")
		log.Debugf("scalingactivities not in 'succeed' or 'failed': %v", tmpIds)
		if err != nil {
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
		self.taskFailed(ctx, sg, fmt.Sprintf("wait for all scaling activities finished: %s", err))
		return
	}
	if !allReady {
		self.taskFailed(ctx, sg, "some scaling activities are still in progress")
		return
	}

	// detach all instances
	err = sg.RemoveAllGuests(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, sg, err.Error())
		return
	}

	// dlete all instances
	self.SetStage("OnDeleteGuestComplete", nil)
	sg.SetStatus(self.UserCred, api.SG_STATUS_DESTROY_INSTANCE, "")
	guests, err := sg.Guests()
	if err != nil {
		self.taskFailed(ctx, sg, fmt.Sprintf("SScalingGroup.Guests: %s", err))
		return
	}
	log.Debugf("guests in scalinggroup '%s': %s", sg.Id, guests)
	if len(guests) == 0 {
		self.OnDeleteGuestComplete(ctx, sg, body)
		return
	}

	for _, guest := range guests {
		err := guest.StartDeleteGuestTask(ctx, self.UserCred, self.Id, true, true, true)
		if err != nil {
			self.taskFailed(ctx, sg, fmt.Sprintf("SGuest.StartDeleteGuestTask for %s: %s", guest.GetId(), err))
			return
		}
	}
}

func (self *ScalingGroupDeleteTask) OnDeleteGuestComplete(ctx context.Context, sg *models.SScalingGroup,
	data jsonutils.JSONObject) {

	log.Debugf("insert OnDeleteGuestComplete")

	subTasks := taskman.SubTaskManager.GetTotalSubtasks(self.Id, "OnDeleteGuestComplete", taskman.SUBTASK_FAIL)
	if len(subTasks) > 0 {
		self.taskFailed(ctx, sg, "some delete guest task failed")
		return
	}

	// delete GuestGroup
	group, err := models.GroupManager.FetchById(sg.GroupId)
	if err == nil {
		self.taskFailed(ctx, sg, fmt.Sprintf("delete guest group %s failed", sg.GroupId))
		return
	}
	err = group.Delete(ctx, self.UserCred)
	if err == nil {
		self.taskFailed(ctx, sg, fmt.Sprintf("delete group failed: %s", err))
		return
	}

	// delete SScalingPolicies
	policies, err := sg.ScalingPolicies()
	if err != nil {
		self.taskFailed(ctx, sg, fmt.Sprintf("SScalingGroup.ScalingPolicies: ", err))
		return
	}
	for _, policy := range policies {
		err := policy.Purge(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, sg, fmt.Sprintf("purge scaling group %s failed: %s", policy.GetId(), err))
			return
		}
	}

	err = sg.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, sg, fmt.Sprintf("ScalingGroup.RealDelete: %s", err))
	}
}

func (self *ScalingGroupDeleteTask) OnDeleteGuestCompleteFailed(ctx context.Context, sg *models.SScalingGroup,
	data jsonutils.JSONObject) {
	log.Errorf("Guest save image failed: %s", data.PrettyString())
	self.taskFailed(ctx, sg, "")
}
