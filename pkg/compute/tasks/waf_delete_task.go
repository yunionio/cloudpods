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
	"time"

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

type WafDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(WafDeleteTask{})
}

func (self *WafDeleteTask) taskFailed(ctx context.Context, waf *models.SWafInstance, err error) {
	waf.SetStatus(ctx, self.UserCred, api.WAF_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, waf, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *WafDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	waf := obj.(*models.SWafInstance)
	iWaf, err := waf.GetICloudWafInstance(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, waf)
			return
		}
		self.taskFailed(ctx, waf, errors.Wrapf(err, "GetICloudWafInstance"))
		return
	}
	err = iWaf.Delete()
	if err != nil {
		self.taskFailed(ctx, waf, errors.Wrapf(err, "iWaf.Delete"))
		return
	}
	err = cloudprovider.WaitDeleted(iWaf, time.Second*5, time.Minute*5)
	if err != nil {
		self.taskFailed(ctx, waf, errors.Wrapf(err, "WaitDeleted"))
		return
	}
	self.taskComplete(ctx, waf)
}

func (self *WafDeleteTask) taskComplete(ctx context.Context, waf *models.SWafInstance) {
	waf.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    waf,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
