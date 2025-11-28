package models

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	llmutils "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	LLMSkuId   string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMImageId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`

	// 秒装应用配额（可安装的总容量限制）
	InstantModelQuotaGb int `list:"user" update:"user" create:"optional" default:"0" nullable:"false"`
}

func (man *SLLMManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMCreateInput) (*api.LLMCreateInput, error) {
	var err error
	input.LLMBaseCreateInput, err = man.SLLMBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMBaseCreateInput")
	}
	sku, err := GetLLMSkuManager().FetchByIdOrName(ctx, userCred, input.LLMSkuId)
	if err != nil {
		return input, errors.Wrap(err, "fetch LLMSku")
	}
	lSku := sku.(*SLLMSku)
	input.LLMSkuId = lSku.Id
	input.LLMImageId = lSku.LLMImageId

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

	if len(input.LLMSku) > 0 {
		skuObj, err := GetLLMSkuManager().FetchByIdOrName(ctx, userCred, input.LLMSku)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GetLLMSkuManager().KeywordPlural(), input.LLMSku)
			} else {
				return nil, errors.Wrap(err, "GetLLMSkuManager.FetchByIdOrName")
			}
		}
		q = q.Equals("llm_sku_id", skuObj.GetId())
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

func (llm *SLLM) GetLLMSku(skuId string) (*SLLMSku, error) {
	if len(skuId) == 0 {
		skuId = llm.LLMSkuId
	}
	sku, err := GetLLMSkuManager().FetchById(skuId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch LLMSku")
	}
	return sku.(*SLLMSku), nil
}

func (llm *SLLM) GetLargeLanguageModelName(name string) (modelName string, modelTag string, err error) {
	if name == "" {
		sku, err := llm.GetLLMSku("")
		if err != nil {
			return "", "", err
		}
		name = sku.LLMModelName
	}
	parts := strings.Split(name, ":")
	modelName = parts[0]
	modelTag = "latest"
	if len(parts) == 2 {
		modelTag = parts[1]
	}
	return
}

func (llm *SLLM) GetLLMImage() (*SLLMImage, error) {
	return llm.getImage(llm.LLMImageId)
}

func (llm *SLLM) GetLLMSContainer(ctx context.Context) (*computeapi.SContainer, error) {
	llmCtr, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMContainer")
	}
	return llmCtr.GetSContainer(ctx)
}

func (llm *SLLM) GetLLMContainer() (*SLLMContainer, error) {
	return GetLLMContainerManager().FetchByLLMId(llm.Id)
}

func (llm *SLLM) GetLLMContainerDriver() ILLMContainerDriver {
	sku, _ := llm.GetLLMSku(llm.LLMSkuId)
	return sku.GetLLMContainerDriver()
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

func (llm *SLLM) ServerCreate(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, input *api.LLMCreateInput) (string, error) {
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if nil != err {
		return "", errors.Wrap(err, "GetLLMSku")
	}
	llmImage, err := llm.GetLLMImage()
	if nil != err {
		return "", errors.Wrap(err, "GetLLMImage")
	}

	data, err := GetLLMPodCreateInput(ctx, userCred, input, llm, sku, llmImage, "")
	if nil != err {
		return "", errors.Wrap(err, "GetPodCreateInput")
	}
	log.Infoln("PodCreateInput Data: ", jsonutils.Marshal(data).String())

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

func (llm *SLLM) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// can't start while it's already running
	if utils.IsInStringArray(llm.Status, computeapi.VM_RUNNING_STATUS) {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "llm id: %s status: %s", llm.Id, llm.Status)
	}

	if err := llm.StartStartTask(ctx, userCred, ""); err != nil {
		return nil, errors.Wrap(err, "StartStartTask")
	}
	return jsonutils.Marshal(nil), nil
}

func (llm *SLLM) StartStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LLMStartTask", llm, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (llm *SLLM) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if llm.Status == computeapi.VM_READY {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "llm id: %s status: %s", llm.Id, llm.Status)
	}
	llm.SetStatus(ctx, userCred, computeapi.VM_START_STOP, "perform stop")
	err := llm.StartLLMStopTask(ctx, userCred, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartStopTask")
	}
	return nil, nil
}

func (llm *SLLM) StartLLMStopTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LLMStopTask", llm, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return errors.Wrap(err, "ScheduleRun")
	}
	return nil
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

func (llm *SLLM) WaitContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.SContainer, error) {
	llmCtr, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMContainer")
	}
	return llmutils.WaitContainerStatus(ctx, llmCtr.CmpId, targetStatus, timeoutSecs)
}

func (llm *SLLM) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data api.LLMSyncStatusInput) (jsonutils.JSONObject, error) {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_SYNCSTATUS, "perform syncstatus")
	err := llm.StartSyncStatusTask(ctx, userCred, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartSyncStatusTask")
	}
	return nil, nil
}

func (llm *SLLM) StartSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LLMSyncStatusTask", llm, userCred, nil, parentTaskId, "")
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return errors.Wrap(err, "ScheduleRun")
	}
	return nil
}
