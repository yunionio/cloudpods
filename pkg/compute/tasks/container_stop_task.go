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

type ContainerStopTask struct {
	ContainerBaseTask
}

func init() {
	taskman.RegisterTask(ContainerStopTask{})
}

func (t *ContainerStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestStop(ctx, obj.(*models.SContainer))
}
func (t *ContainerStopTask) requestStop(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnStopped", nil)
	if err := t.GetPodDriver().RequestStopContainer(ctx, t.GetUserCred(), t); err != nil {
		t.OnStoppedFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerStopTask) OnStoppedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_STOP_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerStopTask) OnStopped(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStage("OnSyncStatus", nil)
	container.StartSyncStatusTask(ctx, t.GetUserCred(), t.GetTaskId())
}

func (t *ContainerStopTask) OnSyncStatus(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
func (t *ContainerStopTask) OnSyncStatusFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
