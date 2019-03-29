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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type EipSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipSyncstatusTask{})
}

func (self *EipSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find ieip for eip %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = extEip.Refresh()
	if err != nil {
		msg := fmt.Sprintf("fail to refresh eip status %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.SyncWithCloudEip(ctx, self.UserCred, eip.GetCloudprovider(), extEip, "")
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = eip.SyncInstanceWithCloudEip(ctx, self.UserCred, extEip)
	if err != nil {
		msg := fmt.Sprintf("fail to sync eip status %s", err)
		eip.SetStatus(self.UserCred, models.EIP_STATUS_UNKNOWN, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
