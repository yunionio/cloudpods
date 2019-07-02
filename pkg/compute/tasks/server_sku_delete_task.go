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

	"yunion.io/x/onecloud/pkg/cloudprovider"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"

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

func (self *ServerSkuDeleteTask) taskFail(ctx context.Context, sku *models.SServerSku, msg string) {
	sku.SetStatus(self.UserCred, api.SkuStatusDeleteFailed, msg)
	db.OpsLog.LogEvent(sku, db.ACT_DELOCATE, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_DELETE, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
	return
}

func (self *ServerSkuDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	sku := obj.(*models.SServerSku)

	iregion, err := sku.GetIRegion()
	if err != nil {
		self.taskFail(ctx, sku, err.Error())
		return
	}

	if len(sku.ExternalId) > 0 {
		isku, err := iregion.GetISkuById(sku.ExternalId)
		if err != nil {
			if err != cloudprovider.ErrNotFound {
				self.taskFail(ctx, sku, err.Error())
				return
			}
		}
		if isku != nil {
			err = isku.Delete()
			if err != nil {
				self.taskFail(ctx, sku, err.Error())
				return
			}
		}
	}

	err = sku.RealDelete(ctx, self.UserCred)
	if err != nil {
		err = errors.Wrapf(err, "sku.RealDelete")
		self.taskFail(ctx, sku, err.Error())
		return
	}

	logclient.AddActionLogWithStartable(self, sku, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
