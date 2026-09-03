package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	commonapis "yunion.io/x/onecloud/pkg/apis"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/llm/tasks/worker"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMInstantModelImportTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMInstantModelImportTask{})
}

func (task *LLMInstantModelImportTask) taskFailed(ctx context.Context, model *models.SInstantModel, err string) {
	// Deleting: do not overwrite status with killed; download was cancelled by delete.
	if model.Status == commonapis.STATUS_DELETING || model.Deleted || model.PendingDeleted {
		task.SetStageFailed(ctx, jsonutils.NewString(err))
		return
	}
	notes := err
	if notes != "" {
		notes = notes + "; resume with resume-import to continue download"
	}
	model.SetStatus(ctx, task.UserCred, imageapi.IMAGE_STATUS_KILLED, notes)
	db.OpsLog.LogEvent(model, db.ACT_CREATE, notes, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_CREATE, notes, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMInstantModelImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)

	input := apis.InstantModelImportInput{}
	err := task.Params.Unmarshal(&input, "import_input")
	if err != nil {
		task.taskFailed(ctx, model, err.Error())
		return
	}

	task.SetStage("OnImportComplete", nil)
	s := auth.GetAdminSession(ctx, options.Options.Region)
	// Run DoImport off the object lock so Delete API is not blocked by long downloads.
	worker.ImportTaskRun(task, func() (jsonutils.JSONObject, error) {
		fileDir, err := model.DoImport(ctx, task.UserCred, s, input)
		result := jsonutils.NewDict()
		if fileDir != "" {
			result.Set("file_dir", jsonutils.NewString(fileDir))
		}
		if err != nil {
			return result, err
		}
		return result, nil
	})
}

func (task *LLMInstantModelImportTask) OnImportComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)

	if model.Status == commonapis.STATUS_DELETING || model.Deleted || model.PendingDeleted {
		task.SetStageComplete(ctx, nil)
		return
	}

	fileDir := ""
	if body != nil {
		fileDir, _ = body.GetString("file_dir")
	}
	if fileDir == "" {
		if fileDirObj, err := task.Params.Get("file_dir"); err == nil {
			fileDir, _ = fileDirObj.GetString()
		}
	}
	if fileDir != "" {
		model.CleanupImportTmpDir(ctx, task.GetUserCred(), fileDir)
	}

	// Best-effort: estimate the model's weight-file size for downstream
	// VRAM-claim calculation. Failure here is a warning, not a fatal — most
	// consumers tolerate `weight_size_bytes = 0` (treated as "unknown").
	if model.WeightSizeBytes == 0 {
		input := apis.InstantModelImportInput{}
		if err := task.Params.Unmarshal(&input, "import_input"); err == nil {
			if w := fetchWeightSizeForImport(ctx, input); w > 0 {
				if _, err := db.Update(model, func() error {
					model.WeightSizeBytes = w
					return nil
				}); err != nil {
					log.Warningf("LLMInstantModelImportTask: persist weight_size_bytes for %s: %s", model.Name, err)
				} else {
					log.Infof("LLMInstantModelImportTask: %s weight_size_bytes=%d", model.Name, w)
				}
			}
		}
	}

	db.OpsLog.LogEvent(model, db.ACT_CREATE, model.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_CREATE, model.GetShortDesc(ctx), task.UserCred, true)

	task.SetStageComplete(ctx, nil)
}

// fetchWeightSizeForImport dispatches by import source.
func fetchWeightSizeForImport(ctx context.Context, input apis.InstantModelImportInput) int64 {
	if input.Source == apis.InstantModelSourceHuggingFace && input.RepoId != "" {
		rev := input.Revision
		if rev == "" {
			rev = "main"
		}
		w, err := models.FetchHuggingFaceWeightSize(ctx, input.RepoId, rev)
		if err != nil {
			log.Warningf("LLMInstantModelImportTask: fetch HF weight size for %s@%s: %s", input.RepoId, rev, err)
			return 0
		}
		return w
	}
	if input.Source == apis.InstantModelSourceModelScope && input.RepoId != "" {
		rev := input.Revision
		if rev == "" {
			rev = "master"
		}
		w, err := models.FetchModelScopeWeightSize(ctx, input.RepoId, rev)
		if err != nil {
			log.Warningf("LLMInstantModelImportTask: fetch ModelScope weight size for %s@%s: %s", input.RepoId, rev, err)
			return 0
		}
		return w
	}
	return 0
}

func (task *LLMInstantModelImportTask) OnImportCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)
	reason, _ := err.GetString("__reason__")
	if reason == "" {
		reason = err.String()
	}
	task.taskFailed(ctx, model, reason)
}
