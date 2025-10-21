package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

var difyManager *SDifyManager

func init() {
	GetDifyManager()
}

func GetDifyManager() *SDifyManager {
	if difyManager != nil {
		return difyManager
	}
	difyManager = &SDifyManager{
		SLLMBaseManager: NewSLLMBaseManager(
			SDify{},
			"difies_tbl",
			"dify",
			"difies",
		),
	}
	difyManager.SetVirtualObject(difyManager)
	return difyManager
}

type SDifyManager struct {
	SLLMBaseManager
}

type SDify struct {
	SLLMBase

	DifyModelId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (dm *SDifyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.DifyCreateInput) (*api.DifyCreateInput, error) {
	var err error
	input.LLMBaseCreateInput, err = dm.SLLMBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate VirtualResourceCreateInput")
	}
	model, err := GetDifyModelManager().FetchByIdOrName(ctx, userCred, input.DifyModelId)
	if err != nil {
		return input, errors.Wrap(err, "fetch DifyModel")
	}
	dModel := model.(*SDifyModel)
	input.DifyModelId = dModel.Id

	return input, nil
}

func (dm *SDifyManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	parentTaskId, _ := data[0].GetString("parent_task_id")
	err := runBatchCreateTask(ctx, items, userCred, data, "DifyBatchCreateTask", parentTaskId)
	if err != nil {
		for i := range items {
			llm := items[i].(*SDify)
			llm.SetStatus(ctx, userCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
		}
	}
}

func (dm *SDifyManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DifyCreateInput) (*jsonutils.JSONDict, error) {
	data, err := dm.ValidateCreateData(ctx, userCred, ownerId, query, &input)
	if err != nil {
		return nil, err
	}
	return data.JSON(data), nil
}

func (dm *SDifyManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.DifyListInput) (*sqlchemy.SQuery, error) {
	q, err := dm.SLLMBaseManager.ListItemFilter(ctx, q, userCred, input.LLMBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "VirtualResourceBaseManager.ListItemFilter")
	}
	if len(input.DifyModel) > 0 {
		modelObj, err := GetDifyModelManager().FetchByIdOrName(ctx, userCred, input.DifyModel)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GetDifyModelManager().KeywordPlural(), input.DifyModel)
			} else {
				return nil, errors.Wrap(err, "DifyModelManager.FetchByIdOrName")
			}
		}
		q = q.Equals("dify_model_id", modelObj.GetId())
	}
	return q, nil
}

func (dify *SDify) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return dify.StartDeleteTask(ctx, userCred, "")
}

func (dify *SDify) GetDifyModel(modelId string) (*SDifyModel, error) {
	if len(modelId) == 0 {
		modelId = dify.DifyModelId
	}
	model, err := GetDifyModelManager().FetchById(modelId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch DifyModel")
	}
	return model.(*SDifyModel), nil
}

func (dify *SDify) GetDifyContainers() []*computeapi.PodContainerCreateInput {
	keys := []string{
		api.DIFY_POSTGRES_KEY,
		api.DIFY_REDIS_KEY,
		api.DIFY_API_KEY,
		api.DIFY_WORKER_KEY,
		api.DIFY_WORKER_BEAT_KEY,
		api.DIFY_PLUGIN_KEY,
		api.DIFY_SANDBOX_KEY,
		api.DIFY_SSRF_KEY,
		api.DIFY_WEB_KEY,
		api.DIFY_NGINX_KEY,
		api.DIFY_WEAVIATE_KEY,
	}

	var containers []*computeapi.PodContainerCreateInput
	for _, key := range keys {
		if c, err := dify.getDifyContainerByContainerKey(key); err == nil {
			containers = append(containers, c)
		}
	}
	return containers
}

func (dify *SDify) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, input api.DifyCreateInput, parentTaskId string) error {
	dify.SetStatus(ctx, userCred, commonapi.STATUS_CREATING, "")
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "DifyCreateTask", dify, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(params)
	}()
	if err != nil {
		dify.SetStatus(ctx, userCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
		return err
	}
	return nil
}

func (dify *SDify) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	dify.SetStatus(ctx, userCred, api.LLM_STATUS_START_DELETE, "StartDeleteTask")
	task, err := taskman.TaskManager.NewTask(ctx, "DifyDeleteTask", dify, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (dify *SDify) ServerCreate(ctx context.Context, userCred mcclient.TokenCredential, input *api.DifyCreateInput) (string, error) {
	model, err := dify.GetDifyModel(dify.DifyModelId)
	if nil != err {
		return "", errors.Wrap(err, "GetDifyModel")
	}

	// set AutoStart to true
	input.AutoStart = true
	data, err := GetDifyPodCreateInput(ctx, userCred, input, dify, model, "")
	if nil != err {
		return "", errors.Wrap(err, "GetDifyPodCreateInput")
	}
	log.Infoln("PodCreateInput Data: ", jsonutils.Marshal(data).String())

	s := auth.GetSession(ctx, userCred, "")
	resp, err := compute.Servers.Create(s, jsonutils.Marshal(data))
	if nil != err {
		return "", errors.Wrap(err, "Servers.Create")
	}
	id, err := resp.GetString("id")
	if nil != err {
		return "", errors.Wrap(err, "resp.GetString")
	}

	return id, nil
}

// func (llm *SDify) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
// 	instanceId, isBound, err := llm.IsBoundToInstance()
// 	if err != nil {
// 		return errors.Wrap(err, "IsBoundToInstance")
// 	}
// 	if isBound {
// 		return httperrors.NewBadRequestError("llm is bound to instance %s", instanceId)
// 	}
// 	return nil
// }

func (dify *SDify) getDifyContainerByContainerKey(containerKey string) (*computeapi.PodContainerCreateInput, error) {
	model, err := dify.GetDifyModel("")
	if nil != err {
		return nil, err
	}
	container, err := getDifyContainersManager().GetContainer(dify.GetName(), containerKey, model)
	if nil != err {
		return nil, err
	}
	container.AlwaysRestart = true // always restart to solve dependency issue
	return container, nil
}

// func (dify *SDify) ContainerCreate(ctx context.Context, userCred mcclient.TokenCredential, containerKey string) (string, error) {
// 	model, err := dify.GetDifyModel("")
// 	if nil != err {
// 		return "", errors.Wrap(err, "GetDifyModel")
// 	}
// 	// get input
// 	input, err := getDifyContainersManager().GetContainer(dify.GetName(), containerKey, model)
// 	if nil != err {
// 		return "", errors.Wrap(err, "GetContainer")
// 	}

// 	// create on pod
// 	params := &computeapi.ContainerCreateInput{
// 		Spec: input.ContainerSpec,
// 	}
// 	s := auth.GetSession(ctx, userCred, "")

// 	return "", nil
// }

func getDifyContainersManager() *DifyContainersManager {
	return &DifyContainersManager{}
}
