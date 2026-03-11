package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/apis/notify"
	notifyapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	llmutils "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
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

	// LLMSpec overrides/extends sku LLMSpec when building container; merged with sku.LLMSpec (llm priority).
	LLMSpec *api.LLMSpec `json:"llm_spec,omitempty" length:"long" list:"user" create:"optional" update:"user"`
}

// CustomizeCreate saves Dify customized envs from create input when present.
func (llm *SLLM) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := llm.SLLMBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
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
	input.LLMImageId = lSku.GetLLMImageId()

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
	if len(input.LLMType) > 0 {
		skuQ := GetLLMSkuManager().Query().SubQuery()
		q = q.Join(skuQ, sqlchemy.Equals(q.Field("llm_sku_id"), skuQ.Field("id")))
		q = q.Filter(sqlchemy.Equals(skuQ.Field("llm_type"), input.LLMType))
	}

	return q, nil
}

func (man *SLLMManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMListDetails {
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	llms := []SLLM{}
	jsonutils.Update(&llms, objs)
	res := make([]api.LLMListDetails, len(objs))
	for i := 0; i < len(res); i++ {
		res[i].VirtualResourceDetails = virtRows[i]
	}

	ids := make([]string, len(llms))
	skuIds := make([]string, len(llms))
	imgIds := make([]string, len(llms))
	serverIds := []string{}
	networkIds := []string{}
	for idx, llm := range llms {
		ids[idx] = llm.Id
		skuIds[idx] = llm.LLMSkuId
		imgIds[idx] = llm.LLMImageId
		if !utils.IsInArray(llm.CmpId, serverIds) {
			serverIds = append(serverIds, llm.CmpId)
		}
		if len(llm.NetworkId) > 0 {
			networkIds = append(networkIds, llm.NetworkId)
		}
		mountedModelInfo, _ := llm.FetchMountedModelInfo()
		res[idx].MountedModels = mountedModelInfo
		res[idx].NetworkType = llm.NetworkType
		res[idx].NetworkId = llm.NetworkId
	}

	// fetch volume
	volumeQ := GetVolumeManager().Query().In("llm_Id", ids)
	volumes := []SVolume{}
	db.FetchModelObjects(GetVolumeManager(), volumeQ, &volumes)
	for _, volume := range volumes {
		for i, id := range ids {
			if id == volume.LLMId {
				res[i].Volume = api.Volume{
					Id:          volume.CmpId,
					Name:        volume.Name,
					TemplateId:  volume.TemplateId,
					StorageType: volume.StorageType,
					SizeMB:      volume.SizeMB,
				}
			}
		}
	}

	// fetch sku
	skus := make(map[string]SLLMSku)
	err := db.FetchModelObjectsByIds(GetLLMSkuManager(), "id", skuIds, &skus)
	if err == nil {
		for i := range llms {
			if sku, ok := skus[llms[i].LLMSkuId]; ok {
				res[i].LLMSku = sku.Name
				res[i].LLMType = sku.LLMType
				res[i].VcpuCount = sku.Cpu
				res[i].VmemSizeMb = sku.Memory
				res[i].Devices = sku.Devices
				if llms[i].BandwidthMb != 0 {
					res[i].EffectBandwidthMbps = llms[i].BandwidthMb
				} else {
					res[i].EffectBandwidthMbps = sku.Bandwidth
				}
			}
		}
	} else {
		log.Errorf("FetchModelObjectsByIds LLMSkuManager fail %s", err)
	}

	// fetch image
	images := make(map[string]SLLMImage)
	err = db.FetchModelObjectsByIds(GetLLMImageManager(), "id", imgIds, &images)
	if err == nil {
		for i := range llms {
			if image, ok := images[llms[i].LLMImageId]; ok {
				res[i].LLMImage = image.Name
				res[i].LLMImageLable = image.ImageLabel
				res[i].LLMImageName = image.ImageName
			}
		}
	} else {
		log.Errorf("FetchModelObjectsByIds GetLLMImageManager fail %s", err)
	}

	// fetch network
	if len(networkIds) > 0 {
		networks, err := fetchNetworks(ctx, userCred, networkIds)
		if err == nil {
			for i, llm := range llms {
				if net, ok := networks[llm.NetworkId]; ok {
					res[i].Network = net.Name
				}
			}
		} else {
			log.Errorf("fail to retrieve network info %s", err)
		}
	}

	// fetch host
	if len(serverIds) > 0 {
		// allow query cmp server
		serverMap := make(map[string]computeapi.ServerDetails)
		s := auth.GetAdminSession(ctx, options.Options.Region)
		params := computeoptions.ServerListOptions{}
		limit := 1000
		params.Limit = &limit
		details := true
		params.Details = &details
		params.Scope = "maxallowed"
		offset := 0
		for offset < len(serverIds) {
			lastIdx := offset + limit
			if lastIdx > len(serverIds) {
				lastIdx = len(serverIds)
			}
			params.Id = serverIds[offset:lastIdx]
			results, err := compute.Servers.List(s, jsonutils.Marshal(params))
			if err != nil {
				log.Errorf("query servers fails %s", err)
				break
			} else {
				offset = lastIdx
				for i := range results.Data {
					guest := computeapi.ServerDetails{}
					err := results.Data[i].Unmarshal(&guest)
					if err == nil {
						serverMap[guest.Id] = guest
					}
				}
			}
		}

		for i := range llms {
			llmStatus := api.LLM_STATUS_UNKNOWN
			llm := llms[i]
			if guest, ok := serverMap[llm.CmpId]; ok {
				// find guest
				if len(guest.Containers) == 0 {
					llmStatus = api.LLM_LLM_STATUS_NO_CONTAINER
				} else {
					llmCtr := guest.Containers[0]
					if llmCtr == nil {
						llmStatus = api.LLM_LLM_STATUS_NO_CONTAINER
					} else {
						llmStatus = llmCtr.Status
					}
				}

				res[i].Server = guest.Name
				res[i].StartTime = guest.LastStartAt
				res[i].Host = guest.Host
				res[i].HostId = guest.HostId
				res[i].HostAccessIp = guest.HostAccessIp
				res[i].HostEIP = guest.HostEIP
				res[i].Zone = guest.Zone
				res[i].ZoneId = guest.ZoneId

				adbMappedPort := -1
				// for j := range res[i].AccessInfo {
				// 	res[i].AccessInfo[j].DesktopIp = guest.IPs
				// 	res[i].AccessInfo[j].ServerIp = guest.HostAccessIp
				// 	res[i].AccessInfo[j].PublicIp = guest.HostEIP
				// 	/*if res[i].AccessInfo[j].ListenPort == api.DESKTOP_ADB_PORT {
				// 		adbMappedPort = res[i].AccessInfo[j].AccessPort
				// 	}*/
				// }

				if adbMappedPort >= 0 {
					res[i].AdbAccess = fmt.Sprintf("%s:%d", guest.HostAccessIp, adbMappedPort)
					if len(res[i].HostEIP) > 0 {
						res[i].AdbPublic = fmt.Sprintf("%s:%d", guest.HostEIP, adbMappedPort)
					}
				}
			} else {
				llmStatus = api.LLM_LLM_STATUS_NO_SERVER
			}
			res[i].LLMStatus = llmStatus
		}
	}

	return res
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
		return "", "", errors.Wrap(errors.ErrInvalidStatus, "model name is empty")
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

func (llm *SLLM) SyncLLMContainer(ctx context.Context, userCred mcclient.TokenCredential, server *computeapi.ServerDetails) (*SLLMContainer, error) {
	curCtr, _ := llm.GetLLMContainer()
	if curCtr != nil {
		return curCtr, nil
	}
	drv := llm.GetLLMContainerDriver()
	ctr, err := drv.GetPrimaryContainer(ctx, llm, server.Containers)
	if err != nil {
		return nil, errors.Wrap(err, "GetPrimaryContainer")
	}
	llmCtr, err := GetLLMContainerManager().CreateOnLLM(ctx, userCred, llm.GetOwnerId(), llm, ctr.Id, ctr.Name)
	if nil != err {
		return nil, errors.Wrapf(err, "create llm container on llm %s", ctr.Id)
	}
	return llmCtr, nil
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

	_, err := llm.GetVolume()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrapf(errors.ErrNotSupported, "llm id: %s missing volume", llm.Id)
		}
		return nil, errors.Wrap(err, "GetVolume")
	}
	taskinput := &api.LLMRestartTaskInput{
		LLMId:     llm.Id,
		LLMStatus: api.LLM_STATUS_READY,
	}
	_, err = llm.StartRestartTask(ctx, userCred, taskinput, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRestartTask")
	}
	return nil, nil
}

func (d *SLLM) StartStartTaskInternal(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	d.SetStatus(ctx, userCred, computeapi.VM_STARTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMStartTask", d, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
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

func (llm *SLLM) ValidateRestartInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMRestartInput) (*api.LLMRestartTaskInput, error) {
	if len(llm.CmpId) == 0 {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "empty cmp_id")
	}

	srv, err := llm.GetServer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetServer")
	}

	if (llm.Status != api.LLM_STATUS_READY && llm.Status != api.LLM_STATUS_RUNNING) || (srv.Status != computeapi.VM_READY && !utils.IsInArray(srv.Status, computeapi.VM_RUNNING_STATUS)) {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "invalid llm status %s", llm.Status)
	}

	return &api.LLMRestartTaskInput{}, nil
}

func (llm *SLLM) PerformRestart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LLMRestartInput) (jsonutils.JSONObject, error) {
	taskInput, err := llm.ValidateRestartInput(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateRestartInput")
	}
	_, err = llm.StartRestartTask(ctx, userCred, taskInput, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRestartTask")
	}
	return nil, nil
}

func (llm *SLLM) StartRestartTask(ctx context.Context, userCred mcclient.TokenCredential, params *api.LLMRestartTaskInput, parentTaskId string) (*taskman.STask, error) {
	key := "perform_restart"
	if params.ResetDataDisk {
		key = "perform_reset"
	}
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_RESTART, key)
	taskName := "LLMRestartTask"
	if params.ResetDataDisk {
		taskName = "LLMResetTask"
	}
	params.LLMId = llm.Id
	task, err := taskman.TaskManager.NewTask(ctx, taskName, llm, userCred, jsonutils.Marshal(params).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}
	if err := task.ScheduleRun(nil); err != nil {
		return nil, errors.Wrap(err, "ScheduleRun")
	}
	return task, nil
}

func (llm *SLLM) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.LLMRestartInput) (jsonutils.JSONObject, error) {
	taskInput, err := llm.ValidateRestartInput(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateRestartInput")
	}
	_, err = llm.StartResetTask(ctx, userCred, taskInput, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRestartTask")
	}
	return nil, nil
}

func (llm *SLLM) StartResetTask(ctx context.Context, userCred mcclient.TokenCredential, params *api.LLMRestartTaskInput, parentTaskId string) (*taskman.STask, error) {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_RESTART, "perform_reset")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMResetTask", llm, userCred, jsonutils.Marshal(params).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "NewTask")
	}
	if err := task.ScheduleRun(nil); err != nil {
		return nil, errors.Wrap(err, "ScheduleRun")
	}
	return task, nil
}

func (llm *SLLM) NotifyRequest(ctx context.Context, userCred mcclient.TokenCredential, action notify.SAction, model jsonutils.JSONObject, success bool) {
	obj := func(ctx context.Context, details *jsonutils.JSONDict) {}
	if model != nil {
		obj = func(ctx context.Context, details *jsonutils.JSONDict) {
			details.Set("customize_details", model)
		}
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:                 llm,
		Action:              action,
		ObjDetailsDecorator: obj,
		IsFail:              !success,
		ResourceType:        notifyapi.TOPIC_RESOURCE_LLM,
	})
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

func (llm *SLLM) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if err := llm.SLLMBase.ValidateDeleteCondition(ctx, info); err != nil {
		return err
	}
	// Check for associated MCPAgents
	cnt, err := GetMCPAgentManager().Query().Equals("llm_id", llm.Id).CountWithError()
	if err != nil {
		return errors.Wrap(err, "GetMCPAgentManager().Query().CountWithError")
	}
	if cnt > 0 {
		return httperrors.NewConflictError("LLM is being used by %d MCPAgents", cnt)
	}
	return nil
}

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

func (llm *SLLM) GetLLMUrl(ctx context.Context, userCred mcclient.TokenCredential) (string, error) {
	if llm.CmpId == "" {
		return "", nil
	}
	return llm.GetLLMContainerDriver().GetLLMUrl(ctx, userCred, llm)
}

func (llm *SLLM) GetDetailsUrl(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	accessUrl, err := llm.GetLLMUrl(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMUrl")
	}
	output := jsonutils.NewDict()
	output.Set("access_url", jsonutils.NewString(accessUrl))
	return output, nil
}

func (llm *SLLM) GetDetailsLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.LLMLoginInfo, error) {
	if llm.CmpId == "" {
		return nil, nil
	}
	output := new(api.LLMLoginInfo)
	loginUrl, err := llm.GetLLMUrl(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMUrl")
	}
	output.LoginUrl = loginUrl
	drv := llm.GetLLMContainerDriver()
	if loginInfoDrv, ok := drv.(ILLMContainerLoginInfo); ok {
		info, err := loginInfoDrv.GetLoginInfo(ctx, userCred, llm)
		if err != nil {
			return nil, errors.Wrap(err, "GetLoginInfo")
		}
		if info.Username != "" {
			output.Username = info.Username
		}
		if info.Password != "" {
			output.Password = info.Password
		}
		if len(info.Extra) > 0 {
			output.Extra = info.Extra
		}
	}
	return output, nil
}

func fetchNetworks(ctx context.Context, userCred mcclient.TokenCredential, networkIds []string) (map[string]computeapi.NetworkDetails, error) {
	s := auth.GetSession(ctx, userCred, "")
	params := computeoptions.ServerListOptions{}
	params.Id = networkIds
	limit := len(networkIds)
	params.Limit = &limit
	params.Scope = "maxallowed"
	results, err := compute.Networks.List(s, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "Networks.List")
	}
	networks := make(map[string]computeapi.NetworkDetails)
	for i := range results.Data {
		net := computeapi.NetworkDetails{}
		err := results.Data[i].Unmarshal(&net)
		if err == nil {
			networks[net.Id] = net
		}
	}
	return networks, nil
}

func (man *SLLMManager) GetAvailableNetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	s := auth.GetSession(ctx, userCred, "")

	ret := jsonutils.NewDict()

	q := jsonutils.NewDict()
	if query != nil {
		q.Update(query)
	}
	q.Set("server_type", jsonutils.NewString(string(computeapi.NETWORK_TYPE_HOSTLOCAL)))
	q.Set("is_auto_alloc", jsonutils.NewBool(true))
	q.Set("status", jsonutils.NewString(computeapi.NETWORK_STATUS_AVAILABLE))
	q.Set("limit", jsonutils.NewInt(1))

	result, err := compute.Networks.List(s, q)
	if err == nil && result.Total > 0 {
		ret.Add(jsonutils.NewInt(int64(result.Total)), "auto_alloc_network_hostlocal_count")
	}

	q.Set("server_type", jsonutils.NewString(string(computeapi.NETWORK_TYPE_GUEST)))
	q.Set("vpc_id", jsonutils.NewString(computeapi.DEFAULT_VPC_ID))

	resultGuest, err := compute.Networks.List(s, q)
	if err == nil && resultGuest.Total > 0 {
		ret.Add(jsonutils.NewInt(int64(resultGuest.Total)), "auto_alloc_network_guest_count")
	}

	return ret, nil
}

func (llm *SLLM) StartBindVolumeTask(ctx context.Context, userCred mcclient.TokenCredential, volumeId string, autoStart bool, parenentTaskId string) (*taskman.STask, error) {
	llm.SetStatus(ctx, userCred, api.LLM_STATUS_START_BIND, "perform bind volume")
	params := api.LLMVolumeInput{
		LLMId:     llm.Id,
		VolumeId:  volumeId,
		AutoStart: autoStart,
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMAttachTask", llm, userCred, jsonutils.Marshal(params).(*jsonutils.JSONDict), parenentTaskId, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return nil, errors.Wrap(err, "ScheduleRun")
	}
	return task, nil
}

func (llm *SLLM) ChangeServerNetworkConfig(ctx context.Context, bandwidth int, whitePrefixes []string, noSync bool) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	params := baseoptions.BaseListOptions{}
	params.Scope = "max"
	limit := 0
	params.Limit = &limit
	serverNicObjs, err := compute.Servernetworks.ListDescendent(s, llm.CmpId, jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "compute.Servernetworks.ListDescendent")
	} else if len(serverNicObjs.Data) == 0 {
		return errors.Wrap(httperrors.ErrEmptyRequest, "compute.Servernetworks.ListDescendent")
	}
	gns := computeapi.GuestnetworkDetails{}
	err = serverNicObjs.Data[0].Unmarshal(&gns)
	if err != nil {
		return errors.Wrap(err, "Unmarshal GuestnetworkDetails")
	}
	if gns.BwLimit != bandwidth {
		// need to change bandwidth
		params := computeapi.ServerChangeBandwidthInput{}
		params.Mac = gns.MacAddr
		params.Index = 0
		params.Bandwidth = bandwidth
		params.NoSync = &noSync
		_, err := compute.Servers.PerformAction(s, llm.CmpId, "change-bandwidth", jsonutils.Marshal(params))
		if err != nil {
			return errors.Wrap(err, "compute.Servers.PerformAction change-bandwidth")
		}
	}
	/*if len(adbWhitePrefixes) > 0 {
		for _, pm := range gns.PortMappings {
			if pm.Port == apis.PHONE_ADB_PORT {
				// verify adb port remote ips
				remoteIps := stringutils2.NewSortedStrings(pm.RemoteIps)
				remoteIps2 := stringutils2.NewSortedStrings(adbWhitePrefixes)
				if !stringutils2.Equals(remoteIps, remoteIps2) {
					// need to update remote Ips
					params := computeapi.GuestnetworkUpdateInput{}
					for i := range gns.PortMappings {
						npm := gns.PortMappings[i]
						if gns.PortMappings[i].Port == apis.PHONE_ADB_PORT {
							npm.RemoteIps = adbWhitePrefixes
						}
						params.PortMappings = append(params.PortMappings, npm)
					}
					_, err := compute.Servernetworks.Update(s, gns.GuestId, gns.NetworkId, nil, jsonutils.Marshal(params))
					if err != nil {
						return errors.Wrap(err, "Servernetworks.Update")
					}
				}
				break
			}
		}
	}*/
	return nil
}

func (llm *SLLM) PerformNetConfig(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMChangeNetworkInput,
) (jsonutils.JSONObject, error) {
	err := llm.ChangeServerNetworkConfig(ctx, input.BandwidthMb, input.WhitePrefxies, false)
	if err != nil {
		return nil, errors.Wrap(err, "changeServerNetworkConfig")
	}

	if llm.BandwidthMb != input.BandwidthMb {
		_, err := db.Update(llm, func() error {
			llm.BandwidthMb = input.BandwidthMb
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "update")
		}
	}

	return nil, nil
}

func (llm *SLLM) purgeModelList() error {
	return GetLLMInstantModelManager().purgeModelList(llm.Id)
}
