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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AccessGroupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AccessGroupCreateTask{})
}

func (self *AccessGroupCreateTask) taskFailed(ctx context.Context, ag *models.SAccessGroup, err error) {
	ag.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, ag, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AccessGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	ag := obj.(*models.SAccessGroup)

	iRegion, err := ag.GetIRegion(ctx)
	if err != nil {
		self.taskFailed(ctx, ag, errors.Wrapf(err, "GetIRegion"))
		return
	}
	opts := &cloudprovider.SAccessGroup{
		Name:           ag.Name,
		Desc:           ag.Description,
		FileSystemType: ag.FileSystemType,
		NetworkType:    ag.NetworkType,
	}
	iGroup, err := iRegion.CreateICloudAccessGroup(opts)
	if err != nil {
		self.taskFailed(ctx, ag, errors.Wrapf(err, "CreateICloudAccessGroup"))
		return
	}

	err = ag.SyncWithAccessGroup(ctx, self.UserCred, iGroup)
	if err != nil {
		self.taskFailed(ctx, ag, errors.Wrapf(err, "SyncAccessGroups"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
