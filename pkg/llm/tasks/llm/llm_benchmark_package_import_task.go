package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMBenchmarkPackageImportTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMBenchmarkPackageImportTask{})
}

func (task *LLMBenchmarkPackageImportTask) taskFailed(ctx context.Context, pkg *models.SLLMBenchmarkPackage, err error, cleanup bool) {
	_ = pkg.SetStatus(ctx, task.UserCred, imageapi.IMAGE_STATUS_KILLED, err.Error())
	db.OpsLog.LogEvent(pkg, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, pkg, logclient.ACT_CREATE, err, task.UserCred, false)
	if cleanup {
		if cleanupErr := pkg.StartDeleteTask(ctx, task.UserCred, nil, ""); cleanupErr != nil {
			log.Errorf("cleanup benchmark package %s: %s", pkg.Id, cleanupErr)
		}
	}
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMBenchmarkPackageImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pkg := obj.(*models.SLLMBenchmarkPackage)
	input := api.LLMBenchmarkPackageImportInput{}
	if err := task.Params.Unmarshal(&input, "import_input"); err != nil {
		task.taskFailed(ctx, pkg, err, false)
		return
	}

	task.SetStage("OnImportComplete", nil)
	session := auth.GetAdminSession(ctx, options.Options.Region)
	tmpDir, err := pkg.DoImport(ctx, task.UserCred, session, input.LLMBenchmarkPackageCreateInput)
	if tmpDir != "" {
		defer pkg.CleanupImportTmpDir(ctx, task.UserCred, tmpDir)
	}
	if err != nil {
		task.OnImportCompleteFailed(ctx, pkg, jsonutils.NewString(err.Error()))
		return
	}
	task.OnImportComplete(ctx, pkg, nil)
}

func (task *LLMBenchmarkPackageImportTask) OnImportComplete(ctx context.Context, pkg *models.SLLMBenchmarkPackage, body jsonutils.JSONObject) {
	input := api.LLMBenchmarkPackageImportInput{}
	if err := task.Params.Unmarshal(&input, "import_input"); err != nil {
		task.taskFailed(ctx, pkg, err, false)
		return
	}

	result := jsonutils.NewDict()
	if input.BenchmarkSpec != nil {
		benchmarkInput, err := models.PrepareLLMBenchmarkCreateInput(pkg, input.BenchmarkSpec)
		if err != nil {
			task.taskFailed(ctx, pkg, err, true)
			return
		}
		benchmark, err := models.GetLLMBenchmarkManager().CreateAndStart(ctx, task.UserCred, pkg.GetOwnerId(), benchmarkInput)
		if err != nil {
			task.taskFailed(ctx, pkg, errors.Wrap(err, "create benchmark"), true)
			return
		}
		result.Set("benchmark_id", jsonutils.NewString(benchmark.Id))
	}

	db.OpsLog.LogEvent(pkg, db.ACT_CREATE, pkg.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, pkg, logclient.ACT_CREATE, pkg.GetShortDesc(ctx), task.UserCred, true)
	task.SetStageComplete(ctx, result)
}

func (task *LLMBenchmarkPackageImportTask) OnImportCompleteFailed(ctx context.Context, pkg *models.SLLMBenchmarkPackage, body jsonutils.JSONObject) {
	input := api.LLMBenchmarkPackageImportInput{}
	_ = task.Params.Unmarshal(&input, "import_input")
	task.taskFailed(
		ctx,
		pkg,
		errors.Errorf("benchmark package import failed: %s", body),
		input.BenchmarkSpec != nil,
	)
}
