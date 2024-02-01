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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ContainerDeleteTask struct {
	ContainerBaseTask
}

func init() {
	taskman.RegisterTask(ContainerDeleteTask{})
}

func (t *ContainerDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestDelete(ctx, obj.(*models.SContainer))
}

func (t *ContainerDeleteTask) requestDelete(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnDeleted", nil)
	if err := t.GetPodDriver().RequestDeleteContainer(ctx, t.GetUserCred(), t); err != nil {
		t.OnDeleteFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerDeleteTask) OnDeleted(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	if err := container.RealDelete(ctx, t.GetUserCred()); err != nil {
		t.OnDeleteFailed(ctx, container, jsonutils.NewString(errors.Wrap(err, "RealDelete").Error()))
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerDeleteTask) OnDeleteFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_DELETE_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}
