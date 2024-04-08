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

type PodSyncstatusTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(PodSyncstatusTask{})
}

func (t *PodSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.SetStage("OnWaitContainerSynced", nil)
	pod := obj.(*models.SGuest)
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_SYNCING_CONTAINER_STATUS, "")
	ctrs, err := models.GetContainerManager().GetContainersByPod(pod.GetId())
	if err != nil {
		t.OnWaitContainerSyncedFailed(ctx, pod, jsonutils.NewString(errors.Wrap(err, "GetContainersByPod").Error()))
		return
	}
	if len(ctrs) == 0 {
		t.OnWaitContainerSynced(ctx, pod, nil)
	} else {
		for i := range ctrs {
			curCtr := ctrs[i]
			curCtr.StartSyncStatusTask(ctx, t.GetUserCred(), t.GetTaskId())
		}
	}
}

func (t *PodSyncstatusTask) OnWaitContainerSynced(ctx context.Context, pod *models.SGuest, _ jsonutils.JSONObject) {
	isAllSynced := true
	ctrs, err := models.GetContainerManager().GetContainersByPod(pod.GetId())
	if err != nil {
		t.OnWaitContainerSyncedFailed(ctx, pod, jsonutils.NewString(errors.Wrap(err, "GetContainersByPod").Error()))
		return
	}
	for i := range ctrs {
		curCtr := ctrs[i]
		if curCtr.GetStatus() == api.CONTAINER_STATUS_SYNC_STATUS {
			isAllSynced = false
		}
	}
	if isAllSynced {
		t.OnContainerSynced(ctx, pod)
	}
}

func (t *PodSyncstatusTask) OnWaitContainerSyncedFailed(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_SYNCING_CONTAINER_STATUS_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

func (t *PodSyncstatusTask) OnContainerSynced(ctx context.Context, pod *models.SGuest) {
	t.SetStage("OnPodSynced", nil)
	models.StartResourceSyncStatusTask(ctx, t.GetUserCred(), pod, "GuestSyncstatusTask", t.GetTaskId())
}

func (t *PodSyncstatusTask) OnPodSynced(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *PodSyncstatusTask) OnPodSyncedFailed(ctx context.Context, pod *models.SGuest, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
