package llm

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMSkuCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMSkuCreateTask{})
}

func (task *LLMSkuCreateTask) taskFailed(ctx context.Context, sku *models.SLLMSku, err error) {
	sku.SetStatus(ctx, task.UserCred, api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED, err.Error())
	db.OpsLog.LogEvent(sku, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, sku, logclient.ACT_CREATE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMSkuCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sku := obj.(*models.SLLMSku)
	importInput := api.InstantModelImportInput{}
	if err := task.GetParams().Unmarshal(&importInput, "import_input"); err != nil {
		task.taskFailed(ctx, sku, errors.Wrap(err, "unmarshal import_input"))
		return
	}
	if importInput.LlmType == "" {
		importInput.LlmType = api.LLMContainerType(sku.LLMType)
	}

	if existing, err := models.GetInstantModelManager().FindReadyInstantModel(
		string(importInput.LlmType),
		importInput.ModelName,
		importInput.ModelTag,
	); err != nil {
		log.Warningf("LLMSkuCreateTask FindReadyInstantModel: %s; importing a fresh InstantModel", err)
	} else if existing != nil {
		task.completeWithInstantModel(ctx, sku, existing.GetId())
		return
	}

	task.SetStage("OnInstantModelReady", nil)
	instantModel, err := models.GetInstantModelManager().DoImportWithParent(ctx, task.UserCred, importInput, task.GetTaskId())
	if err != nil {
		task.taskFailed(ctx, sku, errors.Wrap(err, "DoImportWithParent"))
		return
	}
	extra := jsonutils.NewDict()
	extra.Set("imported_instant_model_id", jsonutils.NewString(instantModel.GetId()))
	if err := task.SaveParams(extra); err != nil {
		log.Warningf("LLMSkuCreateTask persist imported instant model id: %s", err)
	}
	if err := sku.AttachMountedModel(ctx, task.UserCred, instantModel.GetId()); err != nil {
		task.taskFailed(ctx, sku, errors.Wrapf(err, "attach InstantModel %s to SKU", instantModel.GetId()))
		return
	}
}

func (task *LLMSkuCreateTask) OnInstantModelReady(ctx context.Context, sku *models.SLLMSku, body jsonutils.JSONObject) {
	instantId, _ := task.GetParams().GetString("imported_instant_model_id")
	if instantId == "" {
		task.taskFailed(ctx, sku, errors.Error("missing imported_instant_model_id in task params"))
		return
	}
	task.completeWithInstantModel(ctx, sku, instantId)
}

func (task *LLMSkuCreateTask) OnInstantModelReadyFailed(ctx context.Context, sku *models.SLLMSku, body jsonutils.JSONObject) {
	task.taskFailed(ctx, sku, fmt.Errorf("InstantModel import failed: %s", body))
}

func (task *LLMSkuCreateTask) completeWithInstantModel(ctx context.Context, sku *models.SLLMSku, instantId string) {
	if err := models.EnableInstantModelForUse(ctx, task.UserCred, instantId); err != nil {
		task.taskFailed(ctx, sku, errors.Wrapf(err, "enable InstantModel %s", instantId))
		return
	}
	if err := sku.AttachMountedModel(ctx, task.UserCred, instantId); err != nil {
		task.taskFailed(ctx, sku, errors.Wrapf(err, "attach InstantModel %s to SKU", instantId))
		return
	}
	if err := sku.SetStatus(ctx, task.UserCred, api.STATUS_READY, "instant model imported"); err != nil {
		task.taskFailed(ctx, sku, errors.Wrap(err, "set sku ready"))
		return
	}
	db.OpsLog.LogEvent(sku, db.ACT_CREATE, sku.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, sku, logclient.ACT_CREATE, sku.GetShortDesc(ctx), task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}
