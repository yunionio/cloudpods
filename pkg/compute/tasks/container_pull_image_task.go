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

func init() {
	taskman.RegisterTask(ContainerPullImageTask{})
}

type ContainerPullImageTask struct {
	ContainerBaseTask
}

func (t *ContainerPullImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestPullImage(ctx, obj.(*models.SContainer))
}

func (t *ContainerPullImageTask) requestPullImage(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnPulled", nil)
	if err := t.GetPodDriver().RequestPullContainerImage(ctx, t.GetUserCred(), t); err != nil {
		t.OnPulledFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerPullImageTask) OnPulledFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_PULL_IMAGE_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerPullImageTask) OnPulled(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_PULLED_IMAGE, "")
	t.SetStageComplete(ctx, nil)
}
