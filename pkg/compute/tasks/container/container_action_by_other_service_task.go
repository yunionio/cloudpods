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

package container

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	llm "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	"yunion.io/x/onecloud/pkg/mcclient/modules/tasks"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(ContainerActionByOtherServiceTask{})
}

type ContainerActionByOtherServiceTask struct {
	ContainerBaseTask
}

func (t *ContainerActionByOtherServiceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestHostAction(ctx, obj.(*models.SContainer))
}

func (t *ContainerActionByOtherServiceTask) requestHostAction(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnAction", nil)

	input := new(api.ContainerRequestHostActionByOtherServiceInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		t.OnActionFailed(ctx, container, jsonutils.NewString(err.Error()))
	}

	if input.HostAction != "" {
		if err := t.GetPodDriver().RequestHostActionByOtherService(ctx, t.GetUserCred(), t); err != nil {
			t.OnActionFailed(ctx, container, jsonutils.NewString(err.Error()))
			return
		}
	}
}

func (t *ContainerActionByOtherServiceTask) OnActionFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	t.Notify(ctx, container, reason, false)
}

func (t *ContainerActionByOtherServiceTask) OnAction(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.Notify(ctx, container, data, true)
}

func (t *ContainerActionByOtherServiceTask) Notify(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject, success bool) {
	manager, id, _ := t.getOtherSerivceTaskId()
	session := auth.GetSession(ctx, t.GetUserCred(), "")
	if success {
		manager.TaskComplete(session, id, data)
	} else {
		manager.TaskFailed2(session, id, data.String())
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerActionByOtherServiceTask) getOtherSerivceTaskId() (*tasks.TasksManager, string, error) {
	input := new(api.ContainerRequestHostActionByOtherServiceInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		return nil, "", err
	}
	return &llm.LLMTasks, input.TaskId, nil
}
