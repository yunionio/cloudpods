package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SGuestQgaBaseTask struct {
	SGuestBaseTask
}

func (self *SGuestQgaBaseTask) guestPing(ctx context.Context, guest *models.SGuest) error {
	host, err := guest.GetHost()
	if err != nil {
		return err
	}
	return guest.GetDriver().QgaRequestGuestPing(ctx, self, host, guest)
}

func (self *SGuestQgaBaseTask) taskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, api.VM_RUNNING, "on qga set user password failed")
	guest.UpdateQgaStatus(api.QGA_STATUS_EXECUTE_FAILED)
	db.OpsLog.LogEvent(guest, db.ACT_SET_USER_PASSWORD_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SET_USER_PASSWORD, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

type GuestQgaSetPasswordTask struct {
	SGuestQgaBaseTask
}

func init() {
	taskman.RegisterTask(GuestQgaSetPasswordTask{})
}

func (self *GuestQgaSetPasswordTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnQgaGuestPing", nil)
	if err := self.guestPing(ctx, guest); err != nil {
		self.OnQgaGuestPingFailed(ctx, guest, nil)
	}
}

func (self *GuestQgaSetPasswordTask) OnQgaGuestPing(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	input := &api.ServerQgaSetPasswordInput{}
	self.GetParams().Unmarshal(input)
	self.SetStage("OnQgaSetUserPassword", nil)
	host, err := guest.GetHost()
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	err = guest.GetDriver().QgaRequestSetUserPassword(ctx, self, host, guest, input)
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestQgaSetPasswordTask) OnQgaGuestPingFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.taskFailed(ctx, guest, data.String())
}

func (self *GuestQgaSetPasswordTask) OnQgaSetUserPassword(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_RUNNING, "on qga set user password success")
	guest.UpdateQgaStatus(api.QGA_STATUS_AVAILABLE)
	db.OpsLog.LogEvent(guest, db.ACT_SET_USER_PASSWORD, "", self.UserCred)

	input := &api.ServerQgaSetPasswordInput{}
	self.GetParams().Unmarshal(input)
	info := make(map[string]interface{})
	secret, _ := utils.EncryptAESBase64(guest.Id, input.Password)
	info["login_account"] = input.Username
	info["login_key"] = secret
	guest.SetAllMetadata(ctx, info, self.UserCred)

	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SET_USER_PASSWORD, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestQgaSetPasswordTask) OnQgaSetUserPasswordFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.taskFailed(ctx, guest, data.String())
}
