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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type PodDeleteTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(PodDeleteTask{})
}

func (t *PodDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.SetStage("OnWaitContainerDeleted", nil)
	t.OnWaitContainerDeleted(ctx, obj.(*models.SGuest), nil)
}

func (t *PodDeleteTask) OnWaitContainerDeleted(ctx context.Context, pod *models.SGuest, _ jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_DELETING_CONTAINER, "")
	ctrs, err := models.GetContainerManager().GetContainersByPod(pod.GetId())
	if err != nil {
		if strings.Contains(err.Error(), "NotFoundError") {
			// already deleted
			t.OnContainerDeleted(ctx, pod)
			return
		}
		t.OnWaitContainerDeletedFailed(ctx, pod, jsonutils.NewString(errors.Wrap(err, "GetContainersByPod").Error()))
		return
	}
	if len(ctrs) == 0 {
		t.OnContainerDeleted(ctx, pod)
		return
	}
	curCtr := ctrs[0]
	curCtr.StartDeleteTask(ctx, t.GetUserCred(), t.GetTaskId(), jsonutils.QueryBoolean(t.GetParams(), "purge", false))
}

func (t *PodDeleteTask) OnWaitContainerDeletedFailed(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	pod.SetStatus(ctx, t.GetUserCred(), api.POD_STATUS_DELETE_CONTAINER_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

func (t *PodDeleteTask) OnContainerDeleted(ctx context.Context, pod *models.SGuest) {
	t.startDeletePod(ctx, pod)
}

func (t *PodDeleteTask) startDeletePod(ctx context.Context, pod *models.SGuest) {
	t.SetStage("OnPodDeleted", nil)
	err := pod.StartBaseDeleteTask(ctx, t)
	if err != nil {
		t.OnPodDeletedFailed(ctx, pod, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *PodDeleteTask) OnPodDeleted(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	pod := obj.(*models.SGuest)
	pod.FinalizeDeleteTask(ctx, t.GetUserCred(), t, data)
	t.SetStageComplete(ctx, nil)
}

func (t *PodDeleteTask) OnPodDeletedFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
