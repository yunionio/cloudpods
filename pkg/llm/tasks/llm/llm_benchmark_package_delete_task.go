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
	benchmarkPackageDeletePollInterval    = 30 * time.Second
	benchmarkPackageDeletePollMaxAttempts = 60
)

type LLMBenchmarkPackageDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMBenchmarkPackageDeleteTask{})
}

func benchmarkPackageImageNotFound(err error) bool {
	if err == nil {
		return false
	}
	if httputils.ErrorCode(err) == 404 {
		return true
	}
	return strings.Contains(err.Error(), "ResourceNotFoundError")
}

func (task *LLMBenchmarkPackageDeleteTask) taskFailed(ctx context.Context, pkg *models.SLLMBenchmarkPackage, err error) {
	_ = pkg.SetStatus(ctx, task.UserCred, commonapis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(pkg, db.ACT_DELETE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, pkg, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMBenchmarkPackageDeleteTask) imageID(pkg *models.SLLMBenchmarkPackage) string {
	if imageID, _ := task.Params.GetString("image_id"); imageID != "" {
		return imageID
	}
	return pkg.ImageId
}

func (task *LLMBenchmarkPackageDeleteTask) waitImageDeleted(ctx context.Context, imageID string) error {
	session := auth.GetAdminSession(ctx, options.Options.Region)
	var lastErr error
	for i := 0; i < benchmarkPackageDeletePollMaxAttempts; i++ {
		_, err := imagemodules.Images.Get(session, imageID, nil)
		if benchmarkPackageImageNotFound(err) {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		time.Sleep(benchmarkPackageDeletePollInterval)
	}
	if lastErr != nil {
		return errors.Wrapf(lastErr, "wait glance image %s deleted", imageID)
	}
	return errors.Errorf("wait glance image %s deleted timeout", imageID)
}

func (task *LLMBenchmarkPackageDeleteTask) deleteImage(ctx context.Context, session *mcclient.ClientSession, imageID string) error {
	protected := false
	_, err := imagemodules.Images.Update(session, imageID, jsonutils.Marshal(imageapi.ImageUpdateInput{
		Protected: &protected,
	}))
	if err != nil && !benchmarkPackageImageNotFound(err) {
		return errors.Wrapf(err, "unprotect glance image %s", imageID)
	}

	params := jsonutils.NewDict()
	params.Set("override_pending_delete", jsonutils.JSONTrue)
	if jsonutils.QueryBoolean(task.Params, "purge", false) {
		params.Set("purge", jsonutils.JSONTrue)
	}
	_, err = imagemodules.Images.DeleteWithParam(session, imageID, params, nil)
	if err != nil && !benchmarkPackageImageNotFound(err) {
		return errors.Wrapf(err, "delete glance image %s", imageID)
	}
	return task.waitImageDeleted(ctx, imageID)
}

func (task *LLMBenchmarkPackageDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pkg := obj.(*models.SLLMBenchmarkPackage)
	_ = pkg.SetStatus(ctx, task.UserCred, commonapis.STATUS_DELETING, "start delete")

	imageID := task.imageID(pkg)
	if imageID == "" {
		task.OnImageDeleted(ctx, pkg, nil)
		return
	}

	session := auth.GetAdminSession(ctx, options.Options.Region)
	_, err := imagemodules.Images.Get(session, imageID, nil)
	if benchmarkPackageImageNotFound(err) {
		task.OnImageDeleted(ctx, pkg, nil)
		return
	}
	if err != nil {
		task.taskFailed(ctx, pkg, errors.Wrapf(err, "get glance image %s", imageID))
		return
	}

	task.SetStage("OnImageDeleted", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, task.deleteImage(ctx, session, imageID)
	})
}

func (task *LLMBenchmarkPackageDeleteTask) OnImageDeletedFailed(ctx context.Context, pkg *models.SLLMBenchmarkPackage, body jsonutils.JSONObject) {
	task.taskFailed(ctx, pkg, errors.Error(body.String()))
}

func (task *LLMBenchmarkPackageDeleteTask) OnImageDeleted(ctx context.Context, pkg *models.SLLMBenchmarkPackage, body jsonutils.JSONObject) {
	if err := pkg.RealDelete(ctx, task.UserCred); err != nil {
		task.taskFailed(ctx, pkg, err)
		return
	}
	logclient.AddActionLogWithStartable(task, pkg, logclient.ACT_DELETE, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}
