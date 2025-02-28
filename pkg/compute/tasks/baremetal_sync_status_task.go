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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalSyncStatusTask struct {
	SBaremetalBaseTask
}

func (self *BaremetalSyncStatusTask) taskFailed(ctx context.Context, baremetal *models.SHost, err error) {
	baremetal.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *BaremetalSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	if baremetal.IsBaremetal {
		self.DoSyncStatus(ctx, baremetal)
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalSyncStatusTask) DoSyncStatus(ctx context.Context, baremetal *models.SHost) {
	drv, err := baremetal.GetHostDriver()
	if err != nil {
		self.taskFailed(ctx, baremetal, errors.Wrapf(err, "GetHostDriver"))
		return
	}
	self.SetStage("OnSyncstatusComplete", nil)
	err = drv.RequestSyncBaremetalHostStatus(ctx, self.GetUserCred(), baremetal, self)
	if err != nil {
		self.taskFailed(ctx, baremetal, errors.Wrapf(err, "GetHostDriver"))
		return
	}
}

func (self *BaremetalSyncStatusTask) OnSyncstatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalSyncStatusTask) OnSyncstatusCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.taskFailed(ctx, baremetal, errors.Errorf(body.String()))
}

func init() {
	taskman.RegisterTask(BaremetalSyncStatusTask{})
}
