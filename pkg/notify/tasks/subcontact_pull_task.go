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

	"yunion.io/x/jsonutils"

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SubcontactPullTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SubcontactPullTask{})
}

func (self *SubcontactPullTask) taskFailed(ctx context.Context, receiver *models.SReceiver, err error) {
	receiver.SetStatus(self.UserCred, apis.RECEIVER_STATUS_PULL_FAILED, err.Error())
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SubcontactPullTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	receiver := obj.(*models.SReceiver)
	contactTypes := []string{}
	self.Params.Unmarshal(&contactTypes, "contact_types")
	if len(contactTypes) == 0 {
		contactTypes, _ = receiver.GetEnabledContactTypes()
	}
	for _, cType := range contactTypes {
		driver := models.GetDriver(cType)
		if !driver.IsPullType() {
			continue
		}
		userId, err := driver.ContactByMobile(ctx, receiver.Mobile, receiver.DomainId)
		if err != nil {
			receiver.MarkContactTypeUnVerified(ctx, cType, err.Error())
			continue
		}
		receiver.SetContact(cType, userId)
		receiver.MarkContactTypeVerified(ctx, cType)
	}
	// success
	receiver.SetStatus(self.UserCred, apis.RECEIVER_STATUS_READY, "")
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
