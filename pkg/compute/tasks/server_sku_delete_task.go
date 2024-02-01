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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ServerSkuDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ServerSkuDeleteTask{})
}

func (self *ServerSkuDeleteTask) taskFail(ctx context.Context, sku *models.SServerSku, err error) {
	sku.SetStatus(ctx, self.UserCred, api.SkuStatusDeleteFailed, err.Error())
	db.OpsLog.LogEvent(sku, db.ACT_DELOCATE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ServerSkuDeleteTask) taskComplete(ctx context.Context, sku *models.SServerSku) {
	err := sku.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFail(ctx, sku, errors.Wrapf(err, "RealDelete"))
		return
	}
	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ServerSkuDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sku := obj.(*models.SServerSku)
	purge := jsonutils.QueryBoolean(self.GetParams(), "purge", false)
	if !purge && utils.IsInStringArray(sku.Provider, api.PRIVATE_CLOUD_PROVIDERS) {
		iSku, err := sku.GetICloudSku(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				self.taskComplete(ctx, sku)
				return
			}
			self.taskFail(ctx, sku, errors.Wrapf(err, "GetICloudSku"))
			return
		}
		err = iSku.Delete()
		if err != nil {
			self.taskFail(ctx, sku, errors.Wrapf(err, "iSku.Delete"))
			return
		}
	}
	self.taskComplete(ctx, sku)
}
