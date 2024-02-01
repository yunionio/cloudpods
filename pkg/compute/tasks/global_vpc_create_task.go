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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or gvpcreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langugvpce governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GlobalVpcCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GlobalVpcCreateTask{})
}

func (self *GlobalVpcCreateTask) taskFailed(ctx context.Context, gvpc *models.SGlobalVpc, err error) {
	gvpc.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, gvpc, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *GlobalVpcCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	gvpc := obj.(*models.SGlobalVpc)

	opts := &cloudprovider.GlobalVpcCreateOptions{
		NAME: gvpc.Name,
		Desc: gvpc.Description,
	}

	log.Infof("global vpc create params: %s", jsonutils.Marshal(opts).String())

	provider, err := gvpc.GetDriver(ctx)
	if err != nil {
		self.taskFailed(ctx, gvpc, errors.Wrapf(err, "GetDriver"))
		return
	}

	iVpc, err := provider.CreateICloudGlobalVpc(opts)
	if err != nil {
		self.taskFailed(ctx, gvpc, errors.Wrapf(err, "CreateICloudGlobalVpc"))
		return
	}

	db.SetExternalId(gvpc, self.GetUserCred(), iVpc.GetGlobalId())

	cloudprovider.WaitMultiStatus(iVpc, []string{
		api.GLOBAL_VPC_STATUS_AVAILABLE,
		apis.STATUS_CREATE_FAILED,
		apis.STATUS_UNKNOWN,
	}, time.Second*5, time.Minute*10)

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionCreate,
	})

	self.SetStage("OnSyncstatusComplete", nil)
	gvpc.StartSyncstatusTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *GlobalVpcCreateTask) OnSyncstatusComplete(ctx context.Context, gvpc *models.SGlobalVpc, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GlobalVpcCreateTask) OnSyncstatusCompleteFailed(ctx context.Context, gvpc *models.SGlobalVpc, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
