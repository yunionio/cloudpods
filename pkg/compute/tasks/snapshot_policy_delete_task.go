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

type SnapshotPolicyDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotPolicyDeleteTask{})
}

func (self *SnapshotPolicyDeleteTask) taskFail(ctx context.Context, sp *models.SSnapshotPolicy, reason string) {
	sp.SetStatus(self.GetUserCred(), api.SNAPSHOT_POLICY_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(sp, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_DELOCATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(sp.Id, sp.Name, api.SNAPSHOT_POLICY_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotPolicyDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sp := obj.(*models.SSnapshotPolicy)
	region := sp.GetRegion()
	if region == nil {
		self.taskFail(ctx, sp, fmt.Sprintf("failed to find region for sp %s", sp.Name))
		return
	}
	self.SetStage("OnSnapshotPolicyDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteSnapshotPolicy(ctx, self.GetUserCred(), sp, self); err != nil {
		self.taskFail(ctx, sp, err.Error())
	}
}

func (self *SnapshotPolicyDeleteTask) OnSnapshotPolicyDeleteComplete(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(sp, db.ACT_DELETE, sp.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	sp.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyDeleteTask) OnSnapshotPolicyDeleteCompleteFailed(ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) {
	self.taskFail(ctx, sp, data.String())
}
