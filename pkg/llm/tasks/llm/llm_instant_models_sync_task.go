package llm

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	models "yunion.io/x/onecloud/pkg/llm/models"
	options "yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var (
	instantModelSyncTaskWorkerMan *appsrv.SHashedWorkerManager
)

type LLMInstantModelsSyncTask struct {
	taskman.STask
}

func InitInstantModelSyncTaskManager() {
	instantModelSyncTaskWorkerMan = appsrv.NewHashWorkerManager("InstantModelSyncTaskManager", options.Options.InstantModelSyncTaskWorkerCount, 1, 1024, true)
	taskman.RegisterTaskAndHashedWorkerManager(LLMInstantModelsSyncTask{}, instantModelSyncTaskWorkerMan)
}

func (task *LLMInstantModelsSyncTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	defer llm.ClearPendingInstantModelQuota(task.Id)
	input := apis.LLMSyncModelTaskInput{}
	task.Params.Unmarshal(&input)

	db.OpsLog.LogEvent(llm, db.ACT_SYNC_CONF_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_SYNC_CONF, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMInstantModelsSyncTask) taskComplete(ctx context.Context, llm *models.SLLM) {
	defer llm.ClearPendingInstantModelQuota(task.Id)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMInstantModelsSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMSyncModelTaskInput{}
	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	removedModelIds, unmountOverlays, err := llm.RequestUnmountModel(ctx, task.UserCred, input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
	if len(removedModelIds) > 0 || len(unmountOverlays) > 0 {
		input.UninstallModelIds = removedModelIds
		task.SetStage("OnModelsUnmountComplete", jsonutils.Marshal(input).(*jsonutils.JSONDict))
		if len(unmountOverlays) > 0 {
			// try unmount post_overlay
			s := auth.GetSession(ctx, task.GetUserCred(), options.Options.Region)
			err := s.WithTaskCallback(task.GetId(), func() error {
				return llm.TryContainerUnmountPaths(ctx, task.UserCred, s, unmountOverlays, 7200)
			})
			if err != nil {
				task.OnModelsUnmountCompleteFailed(ctx, llm, jsonutils.NewString(errors.Wrap(err, "TryContainerUnmountPaths").Error()))
			}
		}
	} else {
		task.OnModelsUnmountComplete(ctx, llm, nil)
	}
}

func (task *LLMInstantModelsSyncTask) OnModelsUnmountComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMSyncModelTaskInput{}
	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	if len(input.UninstallModelIds) > 0 {
		// did uninstall models, clear the flags
		log.Debugf("uninstall_models %s", jsonutils.Marshal(input.UninstallModelIds).PrettyString())
		err := llm.MarkInstantModelsUnmounted(ctx, task.UserCred, input.LLMStatus, input.UninstallModelIds)
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "MarkModelsUnmounted").Error())
			return
		}
	}

	mdlIds, installDirs, overlays, err := llm.RequestMountModels(ctx, task.UserCred, input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}
	log.Debugf("=======RequestMountModels: %s, overlays: %s", jsonutils.Marshal(mdlIds).PrettyString(), jsonutils.Marshal(overlays).PrettyString())
	if len(mdlIds) == 0 {
		task.taskComplete(ctx, llm)
		return
	}

	input.InstallModelIds = mdlIds
	input.InstallDirs = installDirs
	if len(overlays) > 0 {
		task.SetStage("OnModelsMountComplete", jsonutils.Marshal(input).(*jsonutils.JSONDict))

		s := auth.GetSession(ctx, task.GetUserCred(), options.Options.Region)
		err := s.WithTaskCallback(task.GetId(), func() error {
			return llm.TryContainerMountPaths(ctx, task.UserCred, s, overlays, 7200)
		})
		if err != nil {
			task.OnModelsMountCompleteFailed(ctx, llm, jsonutils.NewString(errors.Wrap(err, "TryContainerMountPaths").Error()))
		}
	} else {
		task.OnModelsMountComplete(ctx, llm, nil)
	}
}

func (task *LLMInstantModelsSyncTask) OnModelsUnmountCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMInstantModelsSyncTask) OnModelsMountComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMSyncModelTaskInput{}
	err := task.Params.Unmarshal(&input)
	if err != nil {
		task.taskFailed(ctx, llm, err.Error())
		return
	}

	log.Debugf("install_models %s", jsonutils.Marshal(input.InstallModelIds).PrettyString())
	err = llm.MarkInstantModelsMounted(ctx, task.UserCred, input.LLMStatus, input.InstallModelIds)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "MarkAppsMounted").Error())
		return
	}

	if input.LLMStatus == apis.LLM_STATUS_RUNNING {
		err := llm.InstallInstantModels(ctx, task.UserCred, input.InstallDirs, input.InstallModelIds)
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "llm.InstallInstantModels").Error())
			return
		}
		task.SetStage("OnWaitModelsMountedComplete", jsonutils.Marshal(input).(*jsonutils.JSONDict))
		ModelSyncTaskRun(task, llm.Id, func() (jsonutils.JSONObject, error) {
			tried := 0
			const intvSecs = 2
			waitSecs := options.Options.ModelSyncTaskWaitSecs
			var errs []error
			succ := false
			for tried < waitSecs/intvSecs && !succ {
				tried++
				time.Sleep(intvSecs * time.Second)
				if err := llm.EnsureInstantModelsInstalled(ctx, task.UserCred, input.InstallModelIds); err != nil {
					errs = append(errs, err)
				} else {
					succ = true
				}
			}
			if !succ && len(errs) > 0 {
				return nil, errors.NewAggregate(errs)
			}
			err := llm.RefreshInstantModels(ctx, task.UserCred, true)
			if err != nil {
				return nil, errors.Wrap(err, "RefreshInstantModels")
			}
			return nil, nil
		})
	} else {
		task.taskComplete(ctx, llm)
	}
}

func (task *LLMInstantModelsSyncTask) OnModelsMountCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	// get duplicated error
	errStr := err.String()
	dupMarker := "duplicated container target dirs map"
	dupIndex := strings.Index(errStr, dupMarker)
	if dupIndex != -1 {
		drv := llm.GetLLMContainerDriver()
		errStr = drv.CheckDuplicateMounts(errStr, dupIndex)
	}

	task.taskFailed(ctx, llm, errStr)
	// sync status to clear failed status of container
	llm.StartSyncStatusTask(ctx, task.UserCred, "")
}

func (task *LLMInstantModelsSyncTask) OnWaitModelsMountedComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskComplete(ctx, llm)
}

func (task *LLMInstantModelsSyncTask) OnWaitModelsMountedCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}
