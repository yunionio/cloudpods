package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ResoleBucketAclTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ResoleBucketAclTask{})
}

func (self *ResoleBucketAclTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	suggestSysAlert := obj.(*models.SSuggestSysAlert)
	err := suggestSysAlert.GetDriver().Resolve(suggestSysAlert)
	if err != nil {
		msg := fmt.Sprintf("fail to change bucket acl: %s", err)
		self.taskFail(ctx, suggestSysAlert, jsonutils.NewString(msg))
		return
	}
	suggestSysAlert.SetStatus(self.UserCred, api.SUGGEST_ALERT_DELETING, "")
	err = suggestSysAlert.RealDelete(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to delete SSuggestSysAlert %s", err)
		self.taskFail(ctx, suggestSysAlert, jsonutils.NewString(msg))
		return
	}
	db.OpsLog.LogEvent(suggestSysAlert, db.ACT_DELETE, nil, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, suggestSysAlert, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ResoleBucketAclTask) taskFail(ctx context.Context, alert *models.SSuggestSysAlert, msg jsonutils.JSONObject) {
	alert.SetStatus(self.UserCred, api.SUGGEST_ALERT_DELETE_FAIL, msg.String())
	db.OpsLog.LogEvent(alert, db.ACT_DELETE, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, alert, logclient.ACT_DELETE, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
	return
}
