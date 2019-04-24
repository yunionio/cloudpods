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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipChangeBandwidthTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipChangeBandwidthTask{})
}

func (self *EipChangeBandwidthTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg string) {
	eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, msg)
	db.OpsLog.LogEvent(eip, db.ACT_CHANGE_BANDWIDTH, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_CHANGE_BANDWIDTH, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
}

func (self *EipChangeBandwidthTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find iEip %s", err)
		self.TaskFail(ctx, eip, msg)
		return
	}

	bandwidth, _ := self.Params.Int("bandwidth")
	if bandwidth <= 0 {
		msg := fmt.Sprintf("invalid bandwidth %d", bandwidth)
		self.TaskFail(ctx, eip, msg)
		return
	}

	err = extEip.ChangeBandwidth(int(bandwidth))

	if err != nil {
		msg := fmt.Sprintf("fail to find iEip %s", err)
		self.TaskFail(ctx, eip, msg)
		return
	}

	err = eip.DoChangeBandwidth(self.UserCred, int(bandwidth))

	if err != nil {
		msg := fmt.Sprintf("fail to synchronize iEip bandwidth %s", err)
		self.TaskFail(ctx, eip, msg)
		return
	}

	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_CHANGE_BANDWIDTH, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
