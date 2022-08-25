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
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/apis/devtool"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/devtool/utils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SshInfoCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SshInfoCreateTask{})
}

func (self *SshInfoCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sshInfo := obj.(*models.SSshInfo)
	serverId := sshInfo.ServerId
	session := auth.GetSession(ctx, self.GetUserCred(), "")
	sshable, cleanFunc, err := utils.CheckSSHable(session, serverId)
	if err != nil {
		sshInfo.MarkCreateFailed(err.Error())
		self.SetStageFailed(ctx, nil)
		return
	}
	_, err = db.Update(sshInfo, func() error {
		if cleanFunc != nil {
			sshInfo.NeedClean = tristate.True
		}
		sshInfo.ServerName = sshable.ServerName
		sshInfo.ServerHypervisor = sshable.ServerHypervisor
		sshInfo.Host = sshable.Host
		sshInfo.User = sshable.User
		sshInfo.Port = sshable.Port
		sshInfo.ForwardId = sshable.ProxyForwardId
		sshInfo.Status = devtool.SSHINFO_STATUS_READY
		return nil
	})
	if err != nil {
		log.Errorf("unable to update sshinfo: %v", err)
	}
	self.SetStageComplete(ctx, nil)
}
