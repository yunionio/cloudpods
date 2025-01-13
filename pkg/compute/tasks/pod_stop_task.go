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

type PodStopTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(PodStopTask{})
}

func (t *PodStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.OnWaitContainerStopped(ctx, obj.(*models.SGuest), nil)
}

type ServerStopTaskParams struct {
	IsForce bool  `json:"is_force"`
	Timeout int64 `json:"timeout"`
}

func (t *PodStopTask) OnWaitContainerStopped(ctx context.Context, pod *models.SGuest, _ jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_STOPPING_CONTAINER, "")
	ctrs, err := models.GetContainerManager().GetContainersByPod(pod.GetId())
	if err != nil {
		t.OnWaitContainerStoppedFailed(ctx, pod, jsonutils.NewString(errors.Wrap(err, "GetContainersByPod").Error()))
		return
	}
	if len(ctrs) == 0 {
		t.OnContainerStopped(ctx, pod, nil)
	} else {
		t.SetStage("OnContainerStopped", nil)
		input := new(ServerStopTaskParams)
		t.GetParams().Unmarshal(input)
		if input.Timeout == 0 {
			input.Timeout = 1
		}
		if err := models.GetContainerManager().StartBatchStopTask(ctx, t.GetUserCred(), ctrs, int(input.Timeout), input.IsForce, t.GetId()); err != nil {
			t.OnWaitContainerStoppedFailed(ctx, pod, jsonutils.NewString(err.Error()))
			return
		}
	}
}

func (t *PodStopTask) OnWaitContainerStoppedFailed(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_STOP_CONTAINER_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

func (t *PodStopTask) OnContainerStopped(ctx context.Context, pod *models.SGuest, _ jsonutils.JSONObject) {
	t.SetStage("OnPodStopped", nil)
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStopTask", pod, t.GetUserCred(), nil, t.GetTaskId(), "", nil)
	if err != nil {
		t.OnPodStoppedFailed(ctx, pod, jsonutils.NewString(err.Error()))
		return
	}
	task.ScheduleRun(nil)
}

func (t *PodStopTask) OnPodStopped(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.VM_READY, "")
	t.SetStageComplete(ctx, nil)
}

func (t *PodStopTask) OnPodStoppedFailed(ctx context.Context, pod *models.SGuest, reason jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.VM_STOP_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}
