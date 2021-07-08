package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DetachAlertResourceTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(&DetachAlertResourceTask{})
}

func (self *DetachAlertResourceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	alert := obj.(*models.SCommonAlert)
	errs := alert.DetachAlertResourceOnDisable(ctx, self.GetUserCred())
	if len(errs) != 0 {
		msg := jsonutils.NewString(fmt.Sprintf("fail to DetachAlertResourceOnAlertDisable:%s.err:%v", alert.Name, errors.NewAggregate(errs)))
		self.taskFail(ctx, alert, msg)
		return
	}
	err := models.GetAlertResourceManager().NotifyAlertResourceCount(ctx)
	if err != nil {
		log.Errorf("DetachAlertResourceTask NotifyAlertResourceCount error:%v", err)
	}
	// detach MonitorResourceJoint when alert disabel
	err = models.MonitorResourceAlertManager.DetachJoint(ctx, self.GetUserCred(),
		monitor.MonitorResourceJointListInput{AlertId: alert.GetId()})
	if err != nil {
		log.Errorf("DetachJoint when alert:%s disable err:%v", alert.GetName(), err)
	}
	logclient.AddActionLogWithStartable(self, alert, logclient.ACT_DETACH_ALERTRESOURCE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DetachAlertResourceTask) taskFail(ctx context.Context, alert *models.SCommonAlert, msg jsonutils.JSONObject) {
	db.OpsLog.LogEvent(alert, db.ACT_DETACH, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, alert, logclient.ACT_DETACH_ALERTRESOURCE, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
	return
}
