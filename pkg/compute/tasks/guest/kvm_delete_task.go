package guest

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type KvmDeleteTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(KvmDeleteTask{})
}

func (t *KvmDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.startDeleteKvm(ctx, obj.(*models.SGuest))
}

func (t *KvmDeleteTask) startDeleteKvm(ctx context.Context, kvm *models.SGuest) {
	t.SetStage("OnKvmDeleted", nil)
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDeleteTask", kvm, t.GetUserCred(), t.GetParams(), t.GetTaskId(), "", nil)
	if err != nil {
		t.OnKvmDeleted(ctx, kvm, jsonutils.NewString(err.Error()))
		return
	}
	task.ScheduleRun(nil)
}

func (t *KvmDeleteTask) OnKvmDeleted(ctx context.Context, kvm *models.SGuest, data jsonutils.JSONObject) {
	kvm.RealDelete(ctx, t.GetUserCred())
	db.OpsLog.LogEvent(kvm, db.ACT_DELOCATE, kvm.GetShortDesc(ctx), t.UserCred)
	logclient.AddActionLogWithStartable(t, kvm, logclient.ACT_DELOCATE, nil, t.UserCred, true)
	if !kvm.IsSystem {
		kvm.EventNotify(ctx, t.UserCred, notifyclient.ActionDelete)
	}
	models.HostManager.ClearSchedDescCache(kvm.HostId)
	t.SetStageComplete(ctx, nil)
}

func (t *KvmDeleteTask) OnKvmDeletedFailed(ctx context.Context, kvm *models.SGuest, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
