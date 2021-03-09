package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type VerificationSendTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VerificationSendTask{})
}

func (self *VerificationSendTask) taskFailed(ctx context.Context, receiver *models.SReceiver, reason string) {
	log.Errorf("fail to send verification: %s", reason)
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_SEND_VERIFICATION, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (self *VerificationSendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	receiver := obj.(*models.SReceiver)
	contactType, _ := self.Params.GetString("contact_type")
	verification, err := models.VerificationManager.Get(receiver.GetId(), contactType)
	if err != nil {
		self.taskFailed(ctx, receiver, fmt.Sprintf("VerificationManager.Get for receiver_id %q and contact_type %q: %s", receiver.GetId(), contactType, err.Error()))
		return
	}
	contact, err := receiver.GetContact(contactType)
	if err != nil {
		self.taskFailed(ctx, receiver, fmt.Sprintf("fail to get contact(type: %s): %s", contactType, err.Error()))
		return
	}

	// build message
	var message string
	switch contactType {
	case api.EMAIL:
		info, err := models.TemplateManager.GetCompanyInfo(ctx)
		if err != nil {
			self.taskFailed(ctx, receiver, fmt.Sprintf("fail to get company info: %s", err.Error()))
			return
		}
		data := struct {
			models.SCompanyInfo
			ReceiverName string
			Code         string
		}{
			ReceiverName: receiver.Name,
			Code:         verification.Token,
			SCompanyInfo: info,
		}
		message = jsonutils.Marshal(data).String()
	case api.MOBILE:
		message = fmt.Sprintf(`{"code": "%s"}`, verification.Token)
	default:
		// no way
	}
	tLang, err := receiver.GetTemplateLang(ctx)
	if err != nil {
		self.taskFailed(ctx, receiver, fmt.Sprintf("unable to GetTemplateLang for receiver %q: %v", receiver.Id, err))
	}

	param, err := models.TemplateManager.FillWithTemplate(ctx, tLang, notifyv2.SNotification{
		ContactType: contactType,
		Topic:       "VERIFY",
		Message:     message,
	})
	if err != nil {
		self.taskFailed(ctx, receiver, err.Error())
		return
	}
    param.Receiver = &apis.SReceiver{
    	Contact:              contact,
    	DomainId:             receiver.DomainId,
    }
	err = models.NotifyService.Send(ctx, contactType, param)
	if err != nil {
		self.taskFailed(ctx, receiver, err.Error())
		return
	}
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_SEND_VERIFICATION, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
