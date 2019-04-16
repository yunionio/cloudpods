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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type EipDeallocateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipDeallocateTask{})
}

func (self *EipDeallocateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	if len(eip.ExternalId) > 0 {
		expEip, err := eip.GetIEip()
		if err != nil {
			if err != cloudprovider.ErrNotFound && err != cloudprovider.ErrInvalidProvider {
				msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
				eip.SetStatus(self.UserCred, api.EIP_STATUS_DEALLOCATE_FAIL, msg)
				self.SetStageFailed(ctx, msg)
				return
			}
		} else {
			err = expEip.Delete()
			if err != nil {
				msg := fmt.Sprintf("fail to delete iEIP %s", err)
				eip.SetStatus(self.UserCred, api.EIP_STATUS_DEALLOCATE_FAIL, msg)
				self.SetStageFailed(ctx, msg)
				return
			}
		}
	}

	err := eip.RealDelete(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to delete EIP %s", err)
		eip.SetStatus(self.UserCred, api.EIP_STATUS_DEALLOCATE_FAIL, msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
