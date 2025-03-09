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

package guest

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(PodRestartTask{})
}

type PodRestartTask struct {
	GuestRestartTask
}

func (t *PodRestartTask) OnServerStopComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.StartServer(ctx, guest)
}

func (t *PodRestartTask) StartServer(ctx context.Context, guest *models.SGuest) {
	t.SetStage("OnServerStartComplete", nil)

	task, err := taskman.TaskManager.NewTask(ctx, "PodStartTask", guest, t.GetUserCred(), nil, t.GetTaskId(), "", nil)
	if err != nil {
		t.OnServerStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	task.ScheduleRun(nil)
}

func (t *PodRestartTask) OnServerStartCompleteFailed(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	t.SetStageFailed(ctx, data)
}
