package llm

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	commonapis "yunion.io/x/onecloud/pkg/apis"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

const (
	instantModelDeletePollInterval    = 30 * time.Second
	instantModelDeletePollMaxAttempts = 60
)

type LLMInstantModelDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMInstantModelDeleteTask{})
}

func (task *LLMInstantModelDeleteTask) taskFailed(ctx context.Context, model *models.SInstantModel, err error) {
	model.SetStatus(ctx, task.UserCred, commonapis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(model, db.ACT_DELETE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMInstantModelDeleteTask) getImageId(model *models.SInstantModel) string {
	if imageId, _ := task.Params.GetString("image_id"); len(imageId) > 0 {
		return imageId
	}
	return model.ImageId
}

func (task *LLMInstantModelDeleteTask) isImageNotFound(err error) bool {
	if err == nil {
		return false
	}
	if httputils.ErrorCode(err) == 404 {
		return true
	}
	return strings.Contains(err.Error(), "ResourceNotFoundError")
}

func (task *LLMInstantModelDeleteTask) unprotectGlanceImage(s *mcclient.ClientSession, imageId string) error {
	protected := false
	updateInput := imageapi.ImageUpdateInput{
		Protected: &protected,
	}
	_, err := imagemodules.Images.Update(s, imageId, jsonutils.Marshal(updateInput))
	if err != nil && !task.isImageNotFound(err) {
		return errors.Wrapf(err, "unprotect glance image %s", imageId)
	}
	return nil
}

func (task *LLMInstantModelDeleteTask) waitImageDeleted(ctx context.Context, imageId string) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	var lastErr error
	for i := 0; i < instantModelDeletePollMaxAttempts; i++ {
		_, err := imagemodules.Images.Get(s, imageId, nil)
		if task.isImageNotFound(err) {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		time.Sleep(instantModelDeletePollInterval)
	}
	if lastErr != nil {
		return errors.Wrapf(lastErr, "wait glance image %s deleted", imageId)
	}
	return errors.Errorf("wait glance image %s deleted timeout", imageId)
}

func (task *LLMInstantModelDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)
	model.SetStatus(ctx, task.UserCred, commonapis.STATUS_DELETING, "start delete")

	imageId := task.getImageId(model)
	if len(imageId) == 0 {
		task.OnImageDeleteComplete(ctx, model, nil)
		return
	}

	s := auth.GetAdminSession(ctx, options.Options.Region)
	_, err := imagemodules.Images.Get(s, imageId, nil)
	if task.isImageNotFound(err) {
		task.OnImageDeleteComplete(ctx, model, nil)
		return
	}
	if err != nil {
		task.taskFailed(ctx, model, errors.Wrapf(err, "get glance image %s", imageId))
		return
	}

	task.SetStage("OnImageDeleteComplete", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if err := task.unprotectGlanceImage(s, imageId); err != nil {
			return nil, err
		}
		deleteParams := jsonutils.NewDict()
		deleteParams.Set("override_pending_delete", jsonutils.JSONTrue)
		if jsonutils.QueryBoolean(task.Params, "purge", false) {
			deleteParams.Set("purge", jsonutils.JSONTrue)
		}
		_, err := imagemodules.Images.DeleteWithParam(s, imageId, deleteParams, nil)
		if err != nil && !task.isImageNotFound(err) {
			return nil, errors.Wrapf(err, "delete glance image %s", imageId)
		}
		if err := task.waitImageDeleted(ctx, imageId); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func (task *LLMInstantModelDeleteTask) OnImageDeleteCompleteFailed(ctx context.Context, model *models.SInstantModel, err jsonutils.JSONObject) {
	task.taskFailed(ctx, model, errors.Error(err.String()))
}

func (task *LLMInstantModelDeleteTask) OnImageDeleteComplete(ctx context.Context, model *models.SInstantModel, body jsonutils.JSONObject) {
	err := model.RealDelete(ctx, task.GetUserCred())
	if err != nil {
		task.taskFailed(ctx, model, err)
		return
	}
	task.SetStageComplete(ctx, nil)
}
