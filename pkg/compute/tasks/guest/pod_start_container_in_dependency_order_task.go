package guest

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/utils"
)

type PodStartContainerInDependencyOrderTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(PodStartContainerInDependencyOrderTask{})
}

func (task *PodStartContainerInDependencyOrderTask) taskFailed(ctx context.Context, pod *models.SGuest, err string) {
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *PodStartContainerInDependencyOrderTask) taskComplete(ctx context.Context, pod *models.SGuest) {
	task.SetStageComplete(ctx, nil)
}

func (t *PodStartContainerInDependencyOrderTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pod := obj.(*models.SGuest)
	ctrs, err := models.GetContainerManager().GetContainersByPod(pod.GetId())
	if err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	// init graph
	dep, err := utils.NewDependencyTopoGraph(
		ctrs,
		func(ctr models.SContainer) string { return ctr.Id },
		func(ctr models.SContainer) string { return ctr.Name },
		func(ctr models.SContainer) []string { return ctr.Spec.DependsOn },
	)
	if err != nil {
		t.taskFailed(ctx, pod, errors.Wrap(err, "New pod container dependency").Error())
		return
	}

	t.SaveParams(jsonutils.Marshal(dep).(*jsonutils.JSONDict))

	t.requestContainersStart(ctx, pod, nil)
}

func (t *PodStartContainerInDependencyOrderTask) requestContainersStart(ctx context.Context, pod *models.SGuest, body jsonutils.JSONObject) {
	dep := new(utils.DependencyTopoGraph[models.SContainer])
	if err := t.GetParams().Unmarshal(dep); err != nil {
		t.taskFailed(ctx, pod, errors.Wrap(err, "Unmarshal container order").Error())
		return
	}

	fetchById := func(uuid string) models.SContainer {
		pctr, err := models.GetContainerManager().FetchById(uuid)
		if err != nil {
			log.Infof("FetchById %s error: %s", uuid, err.Error())
			return models.SContainer{}
		}
		return *pctr.(*models.SContainer)
	}
	currentBatch := dep.GetNextBatch(fetchById)
	if currentBatch == nil {
		t.taskComplete(ctx, pod)
		return
	}

	// start current Batch
	t.SetStage("OnContainerStarted", jsonutils.Marshal(dep).(*jsonutils.JSONDict))
	if err := models.GetContainerManager().StartBatchStartTask(ctx, t.GetUserCred(), currentBatch, t.GetId()); err != nil {
		t.OnContainerStartedFailed(ctx, pod, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *PodStartContainerInDependencyOrderTask) OnContainerStarted(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	t.requestContainersStart(ctx, pod, nil)
}

func (t *PodStartContainerInDependencyOrderTask) OnContainerStartedFailed(ctx context.Context, pod *models.SGuest, data jsonutils.JSONObject) {
	t.taskFailed(ctx, pod, data.String())
}
