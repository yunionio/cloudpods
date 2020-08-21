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
	"yunion.io/x/pkg/utils"

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var PullContactType = []string{
	apis.DINGTALK,
	apis.FEISHU,
	apis.WORKWX,
}

type SubcontactPullTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SubcontactPullTask{})
}

func (self *SubcontactPullTask) taskFailed(ctx context.Context, receiver *models.SReceiver, reason string) {
	log.Errorf("fail to pull subcontact of receiver %q: %s", receiver.Id, reason)
	receiver.SetStatus(self.UserCred, apis.RECEIVER_STATUS_PULL_FAILED, reason)
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (self *SubcontactPullTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	// pull contacts
	receiver := obj.(*models.SReceiver)
	if len(receiver.Mobile) == 0 {
		return
	}
	enabledContactTypes, _ := receiver.GetEnabledContactTypes()
	for _, cType := range enabledContactTypes {
		if !utils.IsInStringArray(cType, PullContactType) {
			continue
		}
		userid, err := models.NotifyService.ContactByMobile(ctx, receiver.Mobile, cType)
		if err != nil {
			reason := fmt.Sprintf("fail to get %s contact by mobile %q: %v", cType, receiver.Mobile, err)
			self.taskFailed(ctx, receiver, reason)
			return
		}
		receiver.SetContact(cType, userid)
		receiver.MarkContactTypeVerified(cType)
	}
	receiver.SetContact(apis.WEBCONSOLE, receiver.Id)
	receiver.MarkContactTypeVerified(apis.WEBCONSOLE)
	// push cache
	err := receiver.PushCache(ctx)
	if err != nil {
		reason := fmt.Sprintf("PushCache: %v", err)
		self.taskFailed(ctx, receiver, reason)
	}
	// success
	receiver.SetStatus(self.UserCred, apis.RECEIVER_STATUS_READY, "")
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, "", self.UserCred, true)
}
