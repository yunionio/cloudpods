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

type PodStartTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(PodStartTask{})
}

func (t *PodStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.SetStage("OnPodStarted", nil)
	pod := obj.(*models.SGuest)
	pod.StartGueststartTask(ctx, t.GetUserCred(), jsonutils.NewDict(), t.GetTaskId())
}

func (t *PodStartTask) OnPodStarted(ctx context.Context, pod *models.SGuest, _ jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_STARTING_CONTAINER, "")
	ctrs, err := models.GetContainerManager().GetContainersByPod(pod.GetId())
	if err != nil {
		t.OnContainerStartedFailed(ctx, pod, jsonutils.NewString(errors.Wrap(err, "GetContainersByPod").Error()))
		return
	}
	t.SetStage("OnContainerStarted", nil)
	if err := models.GetContainerManager().StartBatchStartTask(ctx, t.GetUserCred(), ctrs, t.GetId()); err != nil {
		t.OnContainerStartedFailed(ctx, pod, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *PodStartTask) OnPodStartedFailed(ctx context.Context, pod *models.SGuest, reason jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.VM_START_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *PodStartTask) OnContainerStarted(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.VM_RUNNING, "")
	t.SetStageComplete(ctx, nil)
}

func (t *PodStartTask) OnContainerStartedFailed(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_START_CONTAINER_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}
