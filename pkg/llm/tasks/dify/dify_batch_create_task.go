package dify

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
)

type DifyBatchCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DifyBatchCreateTask{})
}

func (task *DifyBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	task.SetStage("OnDifyCreateCompleteAll", nil)

	inputs := make([]api.DifyCreateInput, 0)
	task.GetParams().Unmarshal(&inputs, "data")

	for i := range objs {
		dify := objs[i].(*models.SDify)
		err := dify.StartCreateTask(ctx, task.UserCred, inputs[i], task.GetTaskId())
		if err != nil {
			log.Errorf("start task for %d dify %s(%s) fail %s", i, dify.Id, dify.Name, err)
		}
	}
}

func (task *DifyBatchCreateTask) OnDifyCreateCompleteAll(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}
