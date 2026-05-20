package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
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
	model.SetStatus(ctx, task.UserCred, imageapi.IMAGE_STATUS_KILLED, err)
	db.OpsLog.LogEvent(model, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, model, logclient.ACT_CREATE, err, task.UserCred, false)
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
	var fileDir string
	// err = s.WithTaskCallback(task.GetId(), func() error {
	// 	fileDir, err = model.DoImport(ctx, task.UserCred, s, input)
	// 	// 将 fileDir 存储到 task.Params 中，以便后续阶段可以访问
	// 	if fileDir != "" {
	// 		task.Params.Set("file_dir", jsonutils.NewString(fileDir))
	// 	}
	// 	return err
	// })
	fileDir, err = model.DoImport(ctx, task.UserCred, s, input)
	// 将 fileDir 存储到 task.Params 中，以便后续阶段可以访问
	if fileDir != "" {
		task.Params.Set("file_dir", jsonutils.NewString(fileDir))
	}
	if err != nil {
		task.OnImportCompleteFailed(ctx, model, jsonutils.NewString(err.Error()))
		return
	}
	task.OnImportComplete(ctx, model, nil)
}

func (task *LLMInstantModelImportTask) OnImportComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)

	// 确保删除 fileDir
	if fileDirObj, err := task.Params.Get("file_dir"); err == nil {
		if fileDir, _ := fileDirObj.GetString(); fileDir != "" {
			model.CleanupImportTmpDir(ctx, task.GetUserCred(), fileDir)
		}
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

// fetchWeightSizeForImport dispatches by import source. Only HuggingFace is
// supported in this phase; ModelScope / local_path / ollama silently return 0
// (left as TODO; UI handles the unknown case gracefully).
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
	return 0
}

func (task *LLMInstantModelImportTask) OnImportCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	model := obj.(*models.SInstantModel)

	task.taskFailed(ctx, model, err.String())
}
