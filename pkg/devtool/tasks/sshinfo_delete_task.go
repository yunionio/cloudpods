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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/devtool/utils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SshInfoDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SshInfoDeleteTask{})
}

func (self *SshInfoDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sshInfo := obj.(*models.SSshInfo)
	if sshInfo.NeedClean.IsFalse() {
		sshInfo.RealDelete(ctx, self.GetUserCred())
		self.SetStageComplete(ctx, nil)
		return
	}
	session := auth.GetSession(ctx, self.GetUserCred(), "")
	clean := utils.GetCleanFunc(session, sshInfo.ServerHypervisor, sshInfo.ServerId, sshInfo.Host, sshInfo.ForwardId, sshInfo.Port)
	err := clean()
	if err != nil {
		sshInfo.MarkDeleteFailed(err.Error())
		self.SetStageFailed(ctx, nil)
		return
	}
	sshInfo.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
