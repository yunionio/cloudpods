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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/notify"
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
	failedReasons := make([]string, 0)
	// pull contacts
	receiver := obj.(*models.SReceiver)
	if len(receiver.Mobile) == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}
	// sync email and mobile to keystone
	s := auth.GetSession(ctx, self.UserCred, "")
	mobile := receiver.Mobile
	if strings.HasPrefix(mobile, "+86 ") {
		mobile = strings.TrimSpace(mobile[4:])
	}
	params := map[string]string{
		"email":  receiver.Email,
		"mobile": receiver.Mobile,
	}
	_, err := identity.UsersV3.Update(s, receiver.Id, jsonutils.Marshal(params))
	if err != nil {
		log.Errorf("update user email and mobile fail %s", err)
	}
	var contactTypes []string
	if self.Params.Contains("contact_types") {
		jArray, _ := self.Params.Get("contact_types")
		contactTypes = jArray.(*jsonutils.JSONArray).GetStringArray()
	} else {
		contactTypes, _ = receiver.GetEnabledContactTypes()
	}
	for _, cType := range contactTypes {
		if !utils.IsInStringArray(cType, PullContactType) {
			continue
		}
		userid, err := models.NotifyService.ContactByMobile(ctx, receiver.Mobile, cType, receiver.GetDomainId())
		if err != nil {
			var reason string
			if errors.Cause(err) == notify.ErrNoSuchMobile {
				receiver.MarkContactTypeUnVerified(cType, notify.ErrNoSuchMobile.Error())
				reason = fmt.Sprintf("%q: no such mobile %s", cType, receiver.Mobile)
			} else if errors.Cause(err) == notify.ErrIncompleteConfig {
				receiver.MarkContactTypeUnVerified(cType, notify.ErrIncompleteConfig.Error())
				reason = fmt.Sprintf("%q: %v", cType, err)
			} else {
				receiver.MarkContactTypeUnVerified(cType, "service exceptions")
				reason = fmt.Sprintf("%q: %v", cType, err)
			}
			failedReasons = append(failedReasons, reason)
			continue
		}
		receiver.SetContact(cType, userid)
		receiver.MarkContactTypeVerified(cType)
	}
	// push cache
	err = receiver.PushCache(ctx)
	if err != nil {
		reason := fmt.Sprintf("PushCache: %v", err)
		self.taskFailed(ctx, receiver, reason)
		return
	}
	if len(failedReasons) > 0 {
		reason := strings.Join(failedReasons, "; ")
		self.taskFailed(ctx, receiver, reason)
		return
	}
	// success
	receiver.SetStatus(self.UserCred, apis.RECEIVER_STATUS_READY, "")
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
