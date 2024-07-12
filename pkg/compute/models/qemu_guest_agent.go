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
	self.SetStatus(ctx, userCred, api.VM_QGA_SET_PASSWORD, "")
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "GuestQgaSetPasswordTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SGuest) PerformQgaPing(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaTimeoutInput,
) (jsonutils.JSONObject, error) {
	if self.PowerStates != api.VM_POWER_STATES_ON {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}

	res := jsonutils.NewDict()
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	drv, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	err = drv.QgaRequestGuestPing(ctx, mcclient.GetTokenHeaders(userCred), host, self, false, input)
	if err != nil {
		res.Set("ping_error", jsonutils.NewString(err.Error()))
	}
	return res, nil
}

func (self *SGuest) PerformQgaCommand(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaCommandInput,
) (jsonutils.JSONObject, error) {
	if self.PowerStates != api.VM_POWER_STATES_ON {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}
	if input.Command == "" {
		return nil, httperrors.NewMissingParameterError("command")
	}
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	drv, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	return drv.RequestQgaCommand(ctx, userCred, jsonutils.Marshal(input), host, self)
}

func (self *SGuest) PerformQgaGuestInfoTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaGuestInfoTaskInput,
) (jsonutils.JSONObject, error) {
	if self.PowerStates != api.VM_POWER_STATES_ON {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	drv, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	return drv.QgaRequestGuestInfoTask(ctx, userCred, nil, host, self)
}

func (self *SGuest) PerformQgaGetNetwork(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerQgaGetNetworkInput,
) (jsonutils.JSONObject, error) {
	if self.PowerStates != api.VM_POWER_STATES_ON {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	drv, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	return drv.QgaRequestGetNetwork(ctx, userCred, nil, host, self)
}

func (self *SGuest) startQgaSyncOsInfoTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(ctx, userCred, api.VM_QGA_SYNC_OS_INFO, "")
	kwargs := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "GuestQgaSyncOsInfoTask", self, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
