package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

type LLMBatchCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMBatchCreateTask{})
}

func (task *LLMBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	task.SetStage("OnLLMCreateCompleteAll", nil)

	inputs := make([]api.LLMCreateInput, 0)
	task.GetParams().Unmarshal(&inputs, "data")

	for i := range objs {
		llm := objs[i].(*models.SLLM)
		err := llm.StartCreateTask(ctx, task.UserCred, inputs[i], task.GetTaskId())
		if err != nil {
			log.Errorf("start task for %d llm %s(%s) fail %s", i, llm.Id, llm.Name, err)
		}
	}
}

func (task *LLMBatchCreateTask) OnLLMCreateCompleteAll(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}
