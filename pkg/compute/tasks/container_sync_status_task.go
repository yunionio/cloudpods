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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(ContainerSyncStatusTask{})
}

type ContainerSyncStatusTask struct {
	ContainerBaseTask
}

func (t *ContainerSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.SetStage("OnSyncStatus", nil)
	if err := t.GetPodDriver().RequestSyncContainerStatus(ctx, t.GetUserCred(), t); err != nil {
		t.OnSyncStatusFailed(ctx, obj.(*models.SContainer), jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerSyncStatusTask) OnSyncStatus(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	log.Infof("========recv from status data: %s", data.PrettyString())
	resp := new(api.ContainerSyncStatusResponse)
	data.Unmarshal(resp)
	container.SetStatus(t.GetUserCred(), resp.Status, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerSyncStatusTask) OnSyncStatusFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_SYNC_STATUS_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}
