package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	cloudutil "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/pkg/errors"
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
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
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
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SLLM struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	LLMModelId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMImageId string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`

	SvrId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	LLMIp string `width:"20" charset:"ascii" nullable:"true" list:"user"`
	// Hypervisor     string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	Priority    int `nullable:"false" default:"100" list:"user"`
	BandwidthMb int `nullable:"true" list:"user" create:"admin_optional"`

	LastAppProbe time.Time `nullable:"true" list:"user" create:"admin_optional"`

	// 是否请求同步更新镜像
	SyncImageRequest bool `default:"false" nullable:"false" list:"user" update:"user"`

	VolumeUsedMb int       `nullable:"true" list:"user"`
	VolumeUsedAt time.Time `nullable:"true" list:"user"`

	// 秒装应用配额（可安装的总容量限制）
	// InstantAppQuotaGb int `list:"user" update:"user" create:"optional" default:"0" nullable:"false"`

	DebugMode     bool `default:"false" nullable:"false" list:"user" update:"user"`
	RootfsUnlimit bool `default:"false" nullable:"false" list:"user" update:"user"`
}

func (man *SLLMManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMCreateInput) (*api.LLMCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate VirtualResourceCreateInput")
	}
	model, err := GetLLMModelManager().FetchByIdOrName(ctx, userCred, input.LLMModelId)
	if err != nil {
		return input, errors.Wrap(err, "fetch DesktopModel")
	}
	lModel := model.(*SLLMModel)
	input.LLMModelId = lModel.Id
	input.LLMImageId = lModel.LLMImageId

	if len(input.PreferHost) > 0 {
		s := auth.GetSession(ctx, userCred, "")
		hostJson, err := compute.Hosts.Get(s, input.PreferHost, nil)
		if err != nil {
			return input, errors.Wrap(err, "get host")
		}
		hostDetails := computeapi.HostDetails{}
		if err := hostJson.Unmarshal(&hostDetails); err != nil {
			return input, errors.Wrap(err, "unmarshal hostDetails")
		}
		if hostDetails.Enabled == nil || !*hostDetails.Enabled {
			return input, errors.Wrap(errors.ErrInvalidStatus, "not enabled")
		}
		if hostDetails.HostStatus != computeapi.HOST_ONLINE {
			return input, errors.Wrap(errors.ErrInvalidStatus, "not online")
		}
		if hostDetails.HostType != computeapi.HOST_TYPE_CONTAINER {
			return input, errors.Wrapf(httperrors.ErrNotAcceptable, "host_type %s not supported", hostDetails.HostType)
		}
		input.PreferHost = hostDetails.Id
	}

	return input, nil
}

func (man *SLLMManager) BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.LLMCreateInput) (*jsonutils.JSONDict, error) {
	data, err := man.ValidateCreateData(ctx, userCred, ownerId, query, &input)
	if err != nil {
		return nil, err
	}
	return data.JSON(data), nil
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

func (llm *SLLM) ServerCreate(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMCreateInput) (string, error) {
	model, err := llm.GetLLMModel(llm.LLMModelId)
	if nil != err {
		return "", errors.Wrap(err, "GetLLMModel")
	}
	llmImage, err := llm.GetLLMImage()
	if nil != err {
		return "", errors.Wrap(err, "GetLLMImage")
	}
	data, err := GetPodCreateInput(ctx, userCred, input, llm, model, llmImage, "")
	if nil != err {
		return "", errors.Wrap(err, "GetPodCreateInput")
	}

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

func (llm *SLLM) getImage(imageId string) (*SLLMImage, error) {
	image, err := GetLLMImageManager().FetchById(imageId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch LLMImage")
	}
	return image.(*SLLMImage), nil
}

func (llm *SLLM) GetLLMImage() (*SLLMImage, error) {
	return llm.getImage(llm.LLMImageId)
}

func (llm *SLLM) GetServer(ctx context.Context) (*computeapi.ServerDetails, error) {
	return cloudutil.GetServer(ctx, llm.SvrId)
}

func (llm *SLLM) GetLLMContainerDriver() ILLMContainerDriver {
	model, _ := llm.GetLLMModel(llm.LLMModelId)
	return model.GetLLMContainerDriver()
}

func (llm *SLLM) WaitDelete(ctx context.Context, userCred mcclient.TokenCredential, timeoutSecs int) error {
	return cloudutil.WaitDelete[computeapi.ServerDetails](ctx, &compute.Servers, llm.SvrId, timeoutSecs)
}

func (llm *SLLM) WaitServerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.ServerDetails, error) {
	return cloudutil.WaitServerStatus(ctx, llm.SvrId, targetStatus, timeoutSecs)
}

// func (llm *SLLM) GetLLMContainerDriver()

// func (llm *SLLM) WaitContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.SContainer, error) {
// 	return nil, nil
// }
