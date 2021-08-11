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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type iTask interface {
	taskman.ITask
	TaskFailed(ctx context.Context, nat models.INatHelper, err jsonutils.JSONObject)
}

func NatToBindIPStage(ctx context.Context, task iTask, nat models.INatHelper) {
	nat.SetStatus(task.GetUserCred(), api.NAT_STATUS_ALLOCATE, "")
	natgateway, err := nat.GetNatgateway()
	if err != nil {
		task.TaskFailed(ctx, nat, jsonutils.NewString(fmt.Sprintf("fetch natgateway failed: %s", err)))
		return
	}
	task.SetStage("OnBindIPComplete", nil)
	if !task.GetParams().Contains("need_bind") {
		task.ScheduleRun(nil)
		return
	}

	eipId, _ := task.GetParams().GetString("eip_id")
	region, _ := natgateway.GetRegion()
	if err := region.GetDriver().RequestBindIPToNatgateway(ctx, task, natgateway, eipId); err != nil {
		task.TaskFailed(ctx, nat, jsonutils.NewString(err.Error()))
		return
	}
}

func CreateINatFailedRollback(ctx context.Context, task iTask, nat models.INatHelper) error {
	natgateway, err := nat.GetNatgateway()
	if err != nil {
		return errors.Wrap(err, "fetch natgateway failed")
	}
	eipId, _ := task.GetParams().GetString("eip_id")
	region, _ := natgateway.GetRegion()
	err = region.GetDriver().BindIPToNatgatewayRollback(ctx, eipId)
	if err != nil {
		return err
	}
	return nil
}
