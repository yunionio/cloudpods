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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestRenewTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestRenewTask{})
	taskman.RegisterTask(PrepaidRecycleHostRenewTask{})
}

func (self *GuestRenewTask) taskFailed(ctx context.Context, guest *models.SGuest, err error) {
	db.OpsLog.LogEvent(guest, db.ACT_REW_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RENEW, err, self.UserCred, false)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_RENEW_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	return
}

func (self *GuestRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	drv, err := guest.GetDriver()
	if err != nil {
		self.taskFailed(ctx, guest, errors.Wrapf(err, "GetDriver"))
		return
	}

	exp, err := drv.RequestRenewInstance(ctx, guest, bc)
	if err != nil {
		self.taskFailed(ctx, guest, errors.Wrapf(err, "RequestRenewInstance"))
		return
	}

	err = guest.SaveRenewInfo(ctx, self.UserCred, &bc, &exp, "")
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RENEW, nil, self.UserCred, true)

	guest.StartSyncstatus(ctx, self.UserCred, "")
	self.SetStageComplete(ctx, nil)
}

type PrepaidRecycleHostRenewTask struct {
	taskman.STask
}

func (self *PrepaidRecycleHostRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	ihost, err := host.GetIHost(ctx)
	if err != nil {
		msg := fmt.Sprintf("host.GetIHost fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	iVM, err := ihost.GetIVMById(host.RealExternalId)
	if err != nil {
		msg := fmt.Sprintf("ihost.GetIVMById fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	log.Debugf("expire before %s", iVM.GetExpiredAt())

	err = iVM.Renew(bc)
	if err != nil {
		msg := fmt.Sprintf("iVM.Renew fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	err = iVM.Refresh()
	if err != nil {
		msg := fmt.Sprintf("refresh after renew fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	log.Debugf("expire after %s", iVM.GetExpiredAt())

	exp := iVM.GetExpiredAt()

	err = host.DoSaveRenewInfo(ctx, self.UserCred, &bc, &exp)
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	self.SetStageComplete(ctx, nil)
}
