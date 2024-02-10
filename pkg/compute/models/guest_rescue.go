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
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (self *SGuest) PerformStartRescue(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Cannot rescue guest hypervisor %s", self.Hypervisor)
	}

	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("guest status must be ready or running")
	}

	// Start rescue vm task
	err := self.StartRescueTask(ctx, userCred, jsonutils.NewDict(), "")
	if err != nil {
		return nil, httperrors.NewInvalidStatusError("guest.StartGuestRescueTask: %s", err.Error())
	}

	// Now it only support kvm guest os rescue
	return nil, nil
}

func (self *SGuest) PerformStopRescue(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.RescueMode {
		return nil, httperrors.NewInvalidStatusError("guest is not in rescue mode")
	}

	// Start rescue vm task
	err := self.StopRescueTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
	if err != nil {
		return nil, httperrors.NewInvalidStatusError("guest.StopGuestRescueTask: %s", err.Error())
	}

	// Now it only support kvm guest os rescue
	return nil, nil
}

func (self *SGuest) UpdateRescueMode(mode bool) error {
	_, err := db.Update(self, func() error {
		self.RescueMode = mode
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update LightMode")
	}
	return nil
}

func (self *SGuest) StartRescueTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	// Now only support KVM
	taskName := "StartRescueTask"
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return err
	}
	return nil
}

func (self *SGuest) StopRescueTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	taskName := "StopRescueTask"
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return err
	}
	return nil
}
