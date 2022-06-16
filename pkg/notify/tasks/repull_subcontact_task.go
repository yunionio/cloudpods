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

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type RepullSuncontactTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(RepullSuncontactTask{})
}

func (self *RepullSuncontactTask) taskFailed(ctx context.Context, config *models.SConfig, err error) {
	logclient.AddActionLogWithContext(ctx, config, logclient.ACT_PULL_SUBCONTACT, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *RepullSuncontactTask) taskComplete(ctx context.Context, config *models.SConfig) {
	if jsonutils.QueryBoolean(self.GetParams(), "deleted", false) {
		config.RealDelete(ctx, self.GetUserCred())
	}
	self.SetStageComplete(ctx, nil)
}

func (self *RepullSuncontactTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	config := obj.(*models.SConfig)
	driver := models.GetDriver(config.Type)
	if !driver.IsPullType() {
		self.taskComplete(ctx, config)
		return
	}

	receivers, err := config.GetReceivers()
	if err != nil {
		self.taskFailed(ctx, config, errors.Wrapf(err, "GetRece"))
		return
	}

	for i := range receivers {
		receiver := &receivers[i]
		lockman.LockObject(ctx, receiver)
		defer lockman.ReleaseObject(ctx, receiver)

		cts, err := receiver.GetVerifiedContactTypes()
		if err != nil {
			self.taskFailed(ctx, config, errors.Wrapf(err, "GetVerifiedContactTypes"))
			return
		}
		// unverify
		if utils.IsInStringArray(config.Type, cts) {
			receiver.MarkContactTypeUnVerified(ctx, config.Type, "config update")
		}
		// pull
		receiver.StartSubcontactPullTask(ctx, self.UserCred, []string{config.Type}, self.Id)
	}
	self.taskComplete(ctx, config)
}
