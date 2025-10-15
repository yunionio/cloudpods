package models

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	cloudutil "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
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

func GetServerIdsByHost(ctx context.Context, userCred mcclient.TokenCredential, hostId string) ([]string, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	params := computeoptions.ServerListOptions{}
	params.Scope = "maxallowed"
	params.Host = hostId
	params.Field = []string{"id"}
	limit := 1024
	params.Limit = &limit
	offset := 0
	total := -1
	idList := stringutils2.NewSortedStrings(nil)
	for total < 0 || offset < total {
		params.Offset = &offset
		results, err := compute.Servers.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrap(err, "Servers.List")
		}
		total = results.Total
		for i := range results.Data {
			idStr, _ := results.Data[i].GetString("id")
			if len(idStr) > 0 {
				idList = idList.Append(idStr)
			}
		}
		offset += len(results.Data)
	}
	return idList, nil
}

func (man *SLLMManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.LLMListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return q, errors.Wrap(err, "VirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	if len(input.Host) > 0 {
		serverIds, err := GetServerIdsByHost(ctx, userCred, input.Host)
		if err != nil {
			return nil, errors.Wrap(err, "GetServerIdsByHost")
		}
		q = q.In("svr_id", serverIds)
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
	if len(input.LLMStatus) > 0 {
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		params := computeoptions.ServerListOptions{}
		params.Scope = "maxallowed"
		params.Status = input.LLMStatus
		params.Field = []string{"guest_id"}
		limit := 1024
		params.Limit = &limit
		offset := 0
		total := -1
		idList := stringutils2.NewSortedStrings(nil)
		for total < 0 || offset < total {
			params.Offset = &offset
			results, err := compute.Containers.List(s, jsonutils.Marshal(params))
			if err != nil {
				return nil, errors.Wrap(err, "Containers.List")
			}
			total = results.Total
			for i := range results.Data {
				idStr, _ := results.Data[i].GetString("guest_id")
				if len(idStr) > 0 {
					idList = idList.Append(idStr)
				}
			}
			offset += len(results.Data)
		}
		q = q.In("svr_id", idList)
	}

	if input.NoVolume != nil {
		volumeQ := GetVolumeManager().Query("llm_id").SubQuery()
		q = q.LeftJoin(volumeQ, sqlchemy.Equals(q.Field("id"), volumeQ.Field("llm_id")))
		if *input.NoVolume {
			q = q.Filter(sqlchemy.IsNull(volumeQ.Field("llm_id")))
		} else {
			q = q.Filter(sqlchemy.IsNotNull(volumeQ.Field("llm_id")))
		}
	}
	if len(input.VolumeId) > 0 {
		volumeObj, err := GetVolumeManager().FetchByIdOrName(ctx, userCred, input.VolumeId)
		if err != nil {
			return nil, errors.Wrap(err, "VolumeManager.FetchByIdOrName")
		}
		vq := GetVolumeManager().Query().SubQuery()
		q = q.Join(vq, sqlchemy.Equals(q.Field("id"), vq.Field("llm_id")))
		q = q.Filter(sqlchemy.Equals(vq.Field("id"), volumeObj.GetId()))
	}

	accessQ := GetAccessInfoManager().Query().SubQuery()
	if input.ListenPort > 0 {
		q = q.Join(accessQ, sqlchemy.Equals(q.Field("id"), accessQ.Field("llm_id")))
		q = q.Filter(sqlchemy.Equals(accessQ.Field("listen_port"), input.ListenPort))
	}

	if len(input.PublicIp) > 0 {
		s := auth.GetSession(ctx, userCred, "")
		hostInput := computeapi.HostListInput{
			PublicIp: []string{input.PublicIp},
		}
		hostInput.Field = []string{"id"}
		hosts, err := compute.Hosts.List(s, jsonutils.Marshal(hostInput))
		if err != nil {
			return nil, errors.Wrap(err, "Hosts.List")
		}
		if len(hosts.Data) == 0 {
			return nil, httperrors.NewNotFoundError("Not found host by public_ip %s", input.PublicIp)
		}
		hostIds := []string{}
		for i := range hosts.Data {
			idStr, _ := hosts.Data[i].GetString("id")
			if len(idStr) > 0 {
				hostIds = append(hostIds, idStr)
			}
		}
		if len(hostIds) > 0 {
			serverIds, err := GetServerIdsByHost(ctx, userCred, hostIds[0])
			if err != nil {
				return nil, errors.Wrap(err, "GetServerIdsByHost")
			}
			q = q.In("svr_id", serverIds)
		}
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

func (llm *SLLM) GetServer(ctx context.Context) (*computeapi.ServerDetails, error) {
	return cloudutil.GetServer(ctx, llm.SvrId)
}

func (llm *SLLM) GetVolume() (*SVolume, error) {
	volume := &SVolume{}
	err := GetVolumeManager().Query().Equals("llm_id", llm.Id).First(volume)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrap(errors.ErrNotFound, "query volume")
		}
		return nil, errors.Wrap(err, "FetchVolume")
	}
	volume.SetModelManager(GetVolumeManager(), volume)
	return volume, nil
}

// 取消自动删除
func (llm *SLLM) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (llm *SLLM) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return llm.SVirtualResourceBase.Delete(ctx, userCred)
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
	data, err := GetPodCreateInput(ctx, userCred, input, llm, model, llmImage, "")
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

func (llm *SLLM) ServerDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(llm.SvrId) == 0 {
		return nil
	}
	s := auth.GetSession(ctx, userCred, "")
	server, err := llm.GetServer(ctx)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return nil
		} else {
			return errors.Wrap(err, "GetServer")
		}
	}
	if server.DisableDelete != nil && *server.DisableDelete {
		// update to allow delete
		_, err = compute.Servers.Update(s, llm.SvrId, jsonutils.Marshal(map[string]interface{}{"disable_delete": false}))
		if err != nil {
			return errors.Wrap(err, "update server to delete")
		}
	}
	_, err = compute.Servers.DeleteWithParam(s, llm.SvrId, jsonutils.Marshal(map[string]interface{}{
		"override_pending_delete": true,
	}), nil)
	if err != nil {
		return errors.Wrap(err, "delete server err:")
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

func (llm *SLLM) WaitDelete(ctx context.Context, userCred mcclient.TokenCredential, timeoutSecs int) error {
	return cloudutil.WaitDelete[computeapi.ServerDetails](ctx, &compute.Servers, llm.SvrId, timeoutSecs)
}

func (llm *SLLM) WaitServerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.ServerDetails, error) {
	return cloudutil.WaitServerStatus(ctx, llm.SvrId, targetStatus, timeoutSecs)
}

func (llm *SLLM) getImage(imageId string) (*SLLMImage, error) {
	image, err := GetLLMImageManager().FetchById(imageId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch LLMImage")
	}
	return image.(*SLLMImage), nil
}

// func (llm *SLLM) GetLLMContainerDriver()

// func (llm *SLLM) WaitContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.SContainer, error) {
// 	return nil, nil
// }
