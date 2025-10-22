package container

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ContainerRestartTask struct {
	ContainerBaseTask
}

func init() {
	taskman.RegisterTask(ContainerRestartTask{})
}

func (t *ContainerRestartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestStop(ctx, obj.(*models.SContainer))
}

func (t *ContainerRestartTask) requestStop(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnStopped", nil)

	input := &api.ContainerStopInput{}
	t.GetParams().Unmarshal(input)

	if err := container.StartStopTask(ctx, t.GetUserCred(), input, t.GetId()); err != nil {
		t.OnStoppedFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerRestartTask) OnStoppedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_STOP_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerRestartTask) OnStopped(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStage("OnStarted", nil)

	if err := container.StartStartTask(ctx, t.GetUserCred(), t.GetTaskId()); err != nil {
		t.OnStartedFailed(ctx, container, jsonutils.NewString(err.Error()))
	}
}

func (t *ContainerRestartTask) OnStarted(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerRestartTask) OnStartedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_START_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}
