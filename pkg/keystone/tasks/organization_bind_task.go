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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type OrganizationBindTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(OrganizationBindTask{})
}

func (task *OrganizationBindTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	orgNode := obj.(*models.SOrganizationNode)

	input := api.OrganizationNodePerformBindInput{}
	err := task.Params.Unmarshal(&input, "input")
	if err != nil {
		log.Errorf("Unmarshal input %s fail %s", task.Params, err)
		task.SetStageFailed(ctx, jsonutils.Marshal(err))
		return
	}
	bind := jsonutils.QueryBoolean(task.Params, "bind", true)

	log.Debugf("start bind objs to organization %s label %s bind: %v", orgNode.OrgId, orgNode.FullLabel, bind)

	err = orgNode.BindTargetIds(ctx, task.UserCred, input, bind)
	if err != nil {
		log.Errorf("BindTargetIds fail %s", err)
		task.SetStageFailed(ctx, jsonutils.Marshal(err))
		return
	}

	notes := jsonutils.NewDict()
	notes.Set("input", jsonutils.Marshal(input))
	notes.Set("organization", orgNode.GetShortDesc(ctx))
	logclient.AddActionLogWithStartable(task, orgNode, logclient.ACT_BIND, notes, task.UserCred, true)
	db.OpsLog.LogEvent(orgNode, db.ACT_BIND, notes, task.UserCred)

	task.SetStageComplete(ctx, notes)
}
