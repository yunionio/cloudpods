package models

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

var llmManager *SLLMManager

func init() {
	GetLLMManager()
}

func GetLLMManager() *SLLMManager {
	if llmManager != nil {
		return llmManager
	}
	llmManager = &SLLMManager{
		SLLMBaseManager: NewSLLMBaseManager(
			SLLM{},
			"llms_tbl",
			"llm",
			"llms",
		),
	}
	llmManager.SetVirtualObject(llmManager)
	return llmManager
}

type SLLMManager struct {
	SLLMBaseManager
}

type SLLM struct {
	SLLMBase

	LLMModelId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMImageId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (man *SLLMManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMCreateInput) (*api.LLMCreateInput, error) {
	var err error
	input.LLMBaseCreateInput, err = man.SLLMBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMBaseCreateInput")
	}
	model, err := GetLLMModelManager().FetchByIdOrName(ctx, userCred, input.LLMModelId)
	if err != nil {
		return input, errors.Wrap(err, "fetch LLMModel")
	}
	lModel := model.(*SLLMModel)
	input.LLMModelId = lModel.Id
	input.LLMImageId = lModel.LLMImageId

	return input, nil
}

func (man *SLLMManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.LLMCreateInput) (*jsonutils.JSONDict, error) {
	data, err := man.ValidateCreateData(ctx, userCred, ownerId, query, &input)
	if err != nil {
		return nil, err
	}
	return data.JSON(data), nil
}

func (man *SLLMManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.LLMListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SLLMBaseManager.ListItemFilter(ctx, q, userCred, input.LLMBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "VirtualResourceBaseManager.ListItemFilter")
	}

	if len(input.LLMModel) > 0 {
		modelObj, err := GetLLMModelManager().FetchByIdOrName(ctx, userCred, input.LLMModel)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GetLLMModelManager().KeywordPlural(), input.LLMModel)
			} else {
				return nil, errors.Wrap(err, "LLMModelManager.FetchByIdOrName")
			}
		}
		q = q.Equals("llm_model_id", modelObj.GetId())
	}
	if len(input.LLMImage) > 0 {
		imgObj, err := GetLLMImageManager().FetchByIdOrName(ctx, userCred, input.LLMImage)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GetLLMImageManager().KeywordPlural(), input.LLMImage)
			} else {
				return nil, errors.Wrap(err, "LLMImageManager.FetchByIdOrName")
			}
		}
		q = q.Equals("llm_image_id", imgObj.GetId())
	}

	// if input.Unused != nil {
	// 	instanceQ := GetDesktopInstanceManager().Query().SubQuery()
	// 	if *input.Unused {
	// 		q = q.NotEquals("id", instanceQ.Query(instanceQ.Field("desktop_id")).SubQuery())
	// 	} else {
	// 		q = q.Join(instanceQ, sqlchemy.Equals(q.Field("id"), instanceQ.Field("desktop_id")))
	// 	}
	// }

	return q, nil
}

func (lm *SLLMManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data []jsonutils.JSONObject) {
	parentTaskId, _ := data[0].GetString("parent_task_id")
	err := runBatchCreateTask(ctx, items, userCred, data, "LLMBatchCreateTask", parentTaskId)
	if err != nil {
		for i := range items {
			llm := items[i].(*SLLM)
			llm.SetStatus(ctx, userCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
		}
	}
}

func runBatchCreateTask(
	ctx context.Context,
	items []db.IModel,
	userCred mcclient.TokenCredential,
	data []jsonutils.JSONObject,
	taskName string,
	parentTaskId string,
) error {
	taskItems := make([]db.IStandaloneModel, len(items))
	for i, t := range items {
		taskItems[i] = t.(db.IStandaloneModel)
	}
	params := jsonutils.NewDict()
	params.Set("data", jsonutils.NewArray(data...))

	task, err := taskman.TaskManager.NewParallelTask(ctx, taskName, taskItems, userCred, params, parentTaskId, "")
	if err != nil {
		return errors.Wrapf(err, "NewParallelTask %s", taskName)
	}

	return task.ScheduleRun(nil)
}

func (llm *SLLM) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return llm.StartDeleteTask(ctx, userCred, "")
}

func (llm *SLLM) GetLLMModel(modelId string) (*SLLMModel, error) {
	if len(modelId) == 0 {
		modelId = llm.LLMModelId
	}
	model, err := GetLLMModelManager().FetchById(modelId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch LLMModel")
	}
	return model.(*SLLMModel), nil
}

func (llm *SLLM) GetLargeLanguageModelName() (modelName string, modelTag string, err error) {
	model, err := llm.GetLLMModel("")
	if err != nil {
		return "", "", err
	}
	name := model.LLMModelName
	parts := strings.Split(name, ":")
	modelName = parts[0]
	modelTag = "latest"
	if len(parts) > 1 {
		modelTag = parts[1]
	}
	return
}

func (llm *SLLM) GetLLMImage() (*SLLMImage, error) {
	return llm.getImage(llm.LLMImageId)
}

func (llm *SLLM) GetLLMContainer() (*SLLMContainer, error) {
	return GetLLMContainerManager().FetchByLLMId(llm.Id)
}

func (llm *SLLM) GetLLMContainerDriver() ILLMContainerDriver {
	model, _ := llm.GetLLMModel(llm.LLMModelId)
	return model.GetLLMContainerDriver()
}

func (llm *SLLM) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, input api.LLMCreateInput, parentTaskId string) error {
	llm.SetStatus(ctx, userCred, commonapi.STATUS_CREATING, "")
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "LLMCreateTask", llm, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(params)
	}()
	if err != nil {
		llm.SetStatus(ctx, userCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
		return err
	}
	return nil
}

func (llm *SLLM) StartPullModelTask(ctx context.Context, userCred mcclient.TokenCredential, input *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LLMPullModelTask", llm, userCred, input, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (llm *SLLM) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_DELETE, "StartDeleteTask")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMDeleteTask", llm, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (llm *SLLM) ServerCreate(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMCreateInput) (string, error) {
	model, err := llm.GetLLMModel(llm.LLMModelId)
	if nil != err {
		return "", errors.Wrap(err, "GetLLMModel")
	}
	llmImage, err := llm.GetLLMImage()
	if nil != err {
		return "", errors.Wrap(err, "GetLLMImage")
	}
	// set AutoStart to true
	input.AutoStart = true
	data, err := GetLLMPodCreateInput(ctx, userCred, input, llm, model, llmImage, "")
	if nil != err {
		return "", errors.Wrap(err, "GetPodCreateInput")
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

// func (llm *SLLM) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
// 	instanceId, isBound, err := llm.IsBoundToInstance()
// 	if err != nil {
// 		return errors.Wrap(err, "IsBoundToInstance")
// 	}
// 	if isBound {
// 		return httperrors.NewBadRequestError("llm is bound to instance %s", instanceId)
// 	}
// 	return nil
// }

// func (llm *SLLM) WaitContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.SContainer, error) {
// 	return nil, nil
// }
