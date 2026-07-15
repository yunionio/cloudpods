package llm

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMBenchmarkDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMBenchmarkDeleteTask{})
}

func (task *LLMBenchmarkDeleteTask) taskFailed(ctx context.Context, benchmark *models.SLLMBenchmark, err error) {
	_ = benchmark.SetState(ctx, task.UserCred, benchmark.State, err.Error())
	db.OpsLog.LogEvent(benchmark, db.ACT_DELETE_FAIL, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, benchmark, logclient.ACT_DELETE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMBenchmarkDeleteTask) deleteBenchmark(ctx context.Context, benchmark *models.SLLMBenchmark) {
	if err := benchmark.CleanupArtifacts(ctx); err != nil {
		log.Warningf("cleanup benchmark %s artifacts during delete: %s", benchmark.Id, err)
	}
	if err := benchmark.RealDelete(ctx, task.UserCred); err != nil {
		task.taskFailed(ctx, benchmark, err)
		return
	}
	logclient.AddActionLogWithStartable(task, benchmark, logclient.ACT_DELETE, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}

func benchmarkPackageDeleteRequired(otherReferences int) bool {
	return otherReferences == 0
}

func (task *LLMBenchmarkDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	benchmark := obj.(*models.SLLMBenchmark)
	packageID := benchmark.BenchmarkPackageId
	if packageID == "" {
		task.deleteBenchmark(ctx, benchmark)
		return
	}

	manager := models.GetLLMBenchmarkManager()
	lockKey := db.GetLockClassKey(manager, benchmark.GetOwnerId())
	lockman.LockClass(ctx, manager, lockKey)
	defer lockman.ReleaseClass(ctx, manager, lockKey)

	otherReferences, err := models.CountBenchmarkPackageReferences(packageID, benchmark.Id)
	if err != nil {
		task.taskFailed(ctx, benchmark, err)
		return
	}
	if !benchmarkPackageDeleteRequired(otherReferences) {
		task.deleteBenchmark(ctx, benchmark)
		return
	}

	pkgObj, err := models.GetLLMBenchmarkPackageManager().FetchById(packageID)
	if err != nil {
		cause := errors.Cause(err)
		if cause == errors.ErrNotFound || cause == sql.ErrNoRows {
			task.deleteBenchmark(ctx, benchmark)
			return
		}
		task.taskFailed(ctx, benchmark, errors.Wrap(err, "fetch benchmark package"))
		return
	}
	pkg := pkgObj.(*models.SLLMBenchmarkPackage)

	task.SetStage("OnPackageDeleted", nil)
	if err := pkg.StartDeleteTask(ctx, task.UserCred, nil, task.GetTaskId()); err != nil {
		task.taskFailed(ctx, benchmark, errors.Wrap(err, "start package delete task"))
	}
}

func (task *LLMBenchmarkDeleteTask) OnPackageDeleted(ctx context.Context, benchmark *models.SLLMBenchmark, body jsonutils.JSONObject) {
	task.deleteBenchmark(ctx, benchmark)
}

func (task *LLMBenchmarkDeleteTask) OnPackageDeletedFailed(ctx context.Context, benchmark *models.SLLMBenchmark, body jsonutils.JSONObject) {
	task.taskFailed(ctx, benchmark, errors.Errorf("benchmark package delete failed: %s", body))
}
