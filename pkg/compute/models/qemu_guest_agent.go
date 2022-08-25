package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (self *SGuest) UpdateQgaStatus(status string) error {
	_, err := db.Update(self, func() error {
		self.QgaStatus = status
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update QgaStatus")
	}
	return nil
}

func (self *SGuest) PerformQgaSetPassword(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaSetPasswordInput,
) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_RUNNING {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}
	if input.Username == "" {
		return nil, httperrors.NewMissingParameterError("username")
	}
	if input.Password == "" {
		return nil, httperrors.NewMissingParameterError("password")
	}
	err := seclib2.ValidatePassword(input.Password)
	if err != nil {
		return nil, err
	}
	self.SetStatus(userCred, api.VM_RESET_PASSWORD, "")
	self.UpdateQgaStatus(api.QGA_STATUS_EXCUTING)
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "GuestQgaSetPasswordTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}
