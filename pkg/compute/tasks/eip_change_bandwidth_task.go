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
	"yunion.io/x/pkg/errors"

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

func (self *EipChangeBandwidthTask) TaskFail(ctx context.Context, eip *models.SElasticip, err error) {
	eip.SetStatus(ctx, self.UserCred, api.EIP_STATUS_READY, err.Error())
	db.OpsLog.LogEvent(eip, db.ACT_CHANGE_BANDWIDTH, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_CHANGE_BANDWIDTH, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *EipChangeBandwidthTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)
	bandwidth, _ := self.Params.Int("bandwidth")
	if bandwidth <= 0 {
		self.TaskFail(ctx, eip, errors.Errorf("nvalid bandwidth %d", bandwidth))
		return
	}

	if eip.IsManaged() {
		extEip, err := eip.GetIEip(ctx)
		if err != nil {
			self.TaskFail(ctx, eip, errors.Wrapf(err, "GetIEip"))
			return
		}

		err = extEip.ChangeBandwidth(int(bandwidth))
		if err != nil {
			self.TaskFail(ctx, eip, errors.Wrapf(err, "ChangeBandwidth"))
			return
		}

	}

	if err := eip.DoChangeBandwidth(ctx, self.UserCred, int(bandwidth)); err != nil {
		self.TaskFail(ctx, eip, errors.Wrapf(err, "DoChangeBandwidth"))
		return
	}
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_CHANGE_BANDWIDTH, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
