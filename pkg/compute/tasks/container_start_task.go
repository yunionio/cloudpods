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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ContainerStartTask struct {
	ContainerBaseTask
}

func init() {
	taskman.RegisterTask(ContainerStartTask{})
}

func (t *ContainerStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestStart(ctx, obj.(*models.SContainer))
}

func (t *ContainerStartTask) requestStart(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnStarted", nil)
	if err := t.GetPodDriver().RequestStartContainer(ctx, t.GetUserCred(), t); err != nil {
		t.OnStartedFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerStartTask) OnStarted(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStage("OnSyncStatus", nil)
	container.StartSyncStatusTask(ctx, t.GetUserCred(), t.GetTaskId())
}

func (t *ContainerStartTask) OnStartedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_START_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerStartTask) OnSyncStatus(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerStartTask) OnSyncStatusFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
