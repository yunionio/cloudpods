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
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ContainerBaseTask struct {
	taskman.STask
}

func (t *ContainerBaseTask) GetContainer() *models.SContainer {
	return t.GetObject().(*models.SContainer)
}

func (t *ContainerBaseTask) GetPod() *models.SGuest {
	return t.GetContainer().GetPod()
}

func (t *ContainerBaseTask) GetPodDriver() models.IPodDriver {
	return t.GetPod().GetDriver().(models.IPodDriver)
}

type ContainerCreateTask struct {
	ContainerBaseTask
}

func init() {
	taskman.RegisterTask(ContainerCreateTask{})
}

func (t *ContainerCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.startPullImage(ctx, obj.(*models.SContainer))
}

func (t *ContainerCreateTask) startPullImage(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnImagePulled", nil)
	input := &hostapi.ContainerPullImageInput{
		Image:      container.Spec.Image,
		PullPolicy: container.Spec.ImagePullPolicy,
	}
	if err := container.StartPullImageTask(ctx, t.GetUserCred(), input, t.GetTaskId()); err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerCreateTask) OnImagePulled(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.requestCreate(ctx, container)
}

func (t *ContainerCreateTask) OnImagePulledFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerCreateTask) requestCreate(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnCreated", nil)
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_CREATING, "")
	if err := t.GetPodDriver().RequestCreateContainer(ctx, t.GetUserCred(), t); err != nil {
		t.OnCreatedFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerCreateTask) OnCreated(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStage("OnStarted", nil)
	if err := container.StartStartTask(ctx, t.GetUserCred(), t.GetTaskId()); err != nil {
		t.OnCreatedFailed(ctx, container, jsonutils.NewString(errors.Wrap(err, "StartStartTask").Error()))
	}
}

func (t *ContainerCreateTask) OnCreatedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(t.GetUserCred(), api.CONTAINER_STATUS_CREATE_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerCreateTask) OnStarted(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerCreateTask) OnStartedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
