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
	taskman.RegisterTask(ContainerSaveVolumeMountImageTask{})
}

type ContainerSaveVolumeMountImageTask struct {
	ContainerBaseTask
}

func (t *ContainerSaveVolumeMountImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestSaveImage(ctx, obj.(*models.SContainer))
}

func (t *ContainerSaveVolumeMountImageTask) requestSaveImage(ctx context.Context, container *models.SContainer) {
	t.SetStage("OnImageSaved", nil)
	if err := t.GetPodDriver().RequestSaveVolumeMountImage(ctx, t.GetUserCred(), t); err != nil {
		t.OnImageSavedFailed(ctx, container, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerSaveVolumeMountImageTask) OnImageSaved(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStage("OnSyncStatus", nil)
	container.StartSyncStatusTask(ctx, t.GetUserCred(), t.GetTaskId())
}

func (t *ContainerSaveVolumeMountImageTask) OnImageSavedFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	container.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_SAVE_IMAGE_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerSaveVolumeMountImageTask) OnSyncStatus(ctx context.Context, container *models.SContainer, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
func (t *ContainerSaveVolumeMountImageTask) OnSyncStatusFailed(ctx context.Context, container *models.SContainer, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
