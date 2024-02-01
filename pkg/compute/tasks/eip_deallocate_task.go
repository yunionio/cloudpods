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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipDeallocateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipDeallocateTask{})
}

func (self *EipDeallocateTask) taskFail(ctx context.Context, eip *models.SElasticip, msg jsonutils.JSONObject) {
	eip.SetStatus(ctx, self.UserCred, api.EIP_STATUS_DEALLOCATE_FAIL, msg.String())
	db.OpsLog.LogEvent(eip, db.ACT_DELOCATE, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_DELETE, msg, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    eip,
		Action: notifyclient.ActionDelete,
		IsFail: true,
	})
	self.SetStageFailed(ctx, msg)
	return
}

func (self *EipDeallocateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	if eip.ExternalId != "" {
		expEip, err := eip.GetIEip(ctx)
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotFound && errors.Cause(err) != cloudprovider.ErrInvalidProvider {
				msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
				self.taskFail(ctx, eip, jsonutils.NewString(msg))
				return
			}
		} else {
			err = expEip.Delete()
			if err != nil {
				msg := fmt.Sprintf("fail to delete iEIP %s", err)
				self.taskFail(ctx, eip, jsonutils.NewString(msg))
				return
			}
		}
	}

	if eip.IsManaged() {
		// TODO clear out guestnics
	}

	err := eip.RealDelete(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to delete EIP %s", err)
		self.taskFail(ctx, eip, jsonutils.NewString(msg))
		return
	}

	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_DELETE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    eip,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
