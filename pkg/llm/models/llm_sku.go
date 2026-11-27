package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	schedulerapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	llmscheduler "yunion.io/x/onecloud/pkg/llm/scheduler"
	"yunion.io/x/onecloud/pkg/llm/utils/vram"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	schedulermodules "yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	mcclientoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetLLMSkuManager()
}

var llmSkuManager *SLLMSkuManager

func GetLLMSkuManager() *SLLMSkuManager {
	if llmSkuManager != nil {
		return llmSkuManager
	}
	llmSkuManager = &SLLMSkuManager{
		SLLMSkuBaseManager: NewSLLMSkuBaseManager(
			SLLMSku{},
			"llm_skus_tbl",
			"llm_sku",
			"llm_skus",
		),
	}
	llmSkuManager.SetVirtualObject(llmSkuManager)
	return llmSkuManager
}

type SLLMSkuManager struct {
	SLLMSkuBaseManager
	SMountedModelsResourceManager
}

type SLLMSku struct {
	SLLMSkuBase
	SMountedModelsResource

	// primary image id of primary container
	LLMImageId string       `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	LLMType    string       `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required"`
	LLMSpec    *api.LLMSpec `json:"llm_spec" length:"long" list:"user" create:"optional" update:"user"`

	// Model source
	Source              string   `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	HuggingfaceRepoId   string   `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	HuggingfaceFilename string   `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	ModelScopeModelId   string   `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	ModelScopeFilePath  string   `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	LocalPath           string   `width:"512" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	PreferHosts         []string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional" update:"user"`
	Categories          []string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional" update:"user"`
	BackendVersion      string   `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	BackendParameters   []string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
}

func (man *SLLMSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.LLMSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SLLMSkuBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SLLMSkuBaseManager.ListItemFilter")
	}
	if len(input.LLMType) > 0 {
		q = q.Equals("llm_type", input.LLMType)
	}
	if len(input.LLMTypes) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("llm_type"), input.LLMTypes))
	}
	q, err = man.SMountedModelsResourceManager.ListItemFilter(ctx, q, userCred, input.MountedModelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMountedAppsResourceManager")
	}
	if len(input.Source) > 0 {
		q = q.Equals("source", input.Source)
	}
	if len(input.Categories) > 0 {
		q = q.Contains("categories", fmt.Sprintf("%q", input.Categories))
	}
	return q, nil
}

func (manager *SLLMSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMSkuDetails {
	skuIds := []string{}
	imageIds := []string{}
	templateIds := []string{}

	skus := []SLLMSku{}
	jsonutils.Update(&skus, objs)
	virows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for _, sku := range skus {
		skuIds = append(skuIds, sku.Id)
		if imgId := sku.GetLLMImageId(); imgId != "" {
			imageIds = append(imageIds, imgId)
		}
		if sku.Volumes != nil && len(*sku.Volumes) > 0 && len((*sku.Volumes)[0].TemplateId) > 0 {
			templateIds = append(templateIds, (*sku.Volumes)[0].TemplateId)
		}
	}

	q := GetLLMManager().Query().In("llm_sku_id", skuIds).GroupBy("llm_sku_id")
	q = q.AppendField(q.Field("llm_sku_id"))
	q = q.AppendField(sqlchemy.COUNT("llm_capacity"))
	details := []struct {
		LLMSkuId    string
		LLMCapacity int
	}{}
	q.All(&details)
	res := make([]api.LLMSkuDetails, len(objs))
	mountedModelIds := make([]string, 0)
	for i, sku := range skus {
		res[i].SharableVirtualResourceDetails = virows[i]
		res[i].LLMType = sku.LLMType
		res[i].LLMSpec = sku.LLMSpec
		res[i].Source = sku.Source
		res[i].HuggingfaceRepoId = sku.HuggingfaceRepoId
		res[i].HuggingfaceFilename = sku.HuggingfaceFilename
		res[i].ModelScopeModelId = sku.ModelScopeModelId
		res[i].ModelScopeFilePath = sku.ModelScopeFilePath
		res[i].LocalPath = sku.LocalPath
		res[i].PreferHosts = GetSkuPreferHosts(&sku)
		res[i].Categories = sku.Categories
		res[i].BackendVersion = sku.BackendVersion
		res[i].BackendParameters = sku.BackendParameters
		for _, v := range details {
			if v.LLMSkuId == sku.Id {
				res[i].LLMCapacity = v.LLMCapacity
				break
			}
		}
		if modelIds := sku.GetMountedModels(); len(modelIds) > 0 {
			mountedModelIds = append(mountedModelIds, modelIds...)
		}
	}

	// fetch mounted models
	if len(mountedModelIds) > 0 {
		instModels := make(map[string]SInstantModel)
		err := db.FetchModelObjectsByIds(GetInstantModelManager(), "id", mountedModelIds, &instModels)
		if err != nil {
			log.Errorf("FetchModelObjectsByIds InstantModelManager fail %s", err)
		} else {
			for i, sku := range skus {
				modelIds := sku.GetMountedModels()
				res[i].VramClaimMb = EstimateVramClaimMbFromInstantModels(sku.LLMType, modelIds, instModels, effectiveMaxModelLen(&sku))
				if len(modelIds) > 0 {
					res[i].MountedModelDetails = make([]api.MountedModelInfo, 0)
					for _, modelId := range modelIds {
						if instModel, ok := instModels[modelId]; ok {
							info := api.MountedModelInfo{
								Id:       instModel.Id,
								ModelId:  instModel.ModelId,
								FullName: instModel.ModelName + ":" + instModel.ModelTag,
							}
							res[i].MountedModelDetails = append(res[i].MountedModelDetails, info)
						}
					}
				}
			}
		}
	} else {
		for i := range skus {
			res[i].VramClaimMb = 0
		}
	}
	{
		images := make(map[string]SLLMImage)
		err := db.FetchModelObjectsByIds(GetLLMImageManager(), "id", imageIds, &images)
		if err == nil {
			for i, sku := range skus {
				if imgId := sku.GetLLMImageId(); imgId != "" {
					if image, ok := images[imgId]; ok {
						res[i].Image = image.Name
						res[i].ImageLabel = image.ImageLabel
						res[i].ImageName = image.ImageName
						res[i].AppName = image.AppName
					}
				}
			}
		} else {
			log.Errorf("FetchModelObjectsByIds LLMImageManager fail %s", err)
		}
	}

	if len(templateIds) > 0 {
		templates, err := fetchTemplates(ctx, userCred, templateIds)
		if err == nil {
			for i, sku := range skus {
				if sku.Volumes == nil || len(*sku.Volumes) == 0 {
					continue
				}
				if templ, ok := templates[(*sku.Volumes)[0].TemplateId]; ok {
					res[i].Template = templ.Name
				}
			}
		} else {
			log.Errorf("fail to retrive image info %s", err)
		}
	}

	return res
}

func boolPtr(v bool) *bool {
	return &v
}

func skuCgroupLimitEnabled(enabled *bool) bool {
	if enabled == nil {
		return true
	}
	return *enabled
}

func applySkuCgroupLimitDefaults(input *api.LLMSkuCreateInput) {
	if input == nil {
		return
	}
	def := !api.IsLLMInferenceType(input.LLMType)
	if input.EnableCgroupCpu == nil {
		input.EnableCgroupCpu = boolPtr(def)
	}
	if input.EnableCgroupMemory == nil {
		input.EnableCgroupMemory = boolPtr(def)
	}
}

func (man *SLLMSkuManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	var err error
	input.LLMSKuBaseCreateInput, err = man.SLLMSkuBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.LLMSKuBaseCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLLMSkuBaseManager.ValidateCreateData")
	}
	if !api.IsLLMContainerType(input.LLMType) && input.LLMType != string(api.LLM_CONTAINER_DIFY) {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_type must be one of "+strings.Join(api.LLM_CONTAINER_TYPES.List(), ","))
	}
	if err := api.ValidateLLMModelCategories(input.Categories); err != nil {
		return input, errors.Wrap(httperrors.ErrInputParameter, err.Error())
	}
	input.Categories = api.NormalizeLLMModelCategories(input.Categories)
	if err := validateLLMModelCategoryForBackend(input.LLMType, input.Categories); err != nil {
		return input, err
	}
	if !api.IsLLMOptionalDiskType(input.LLMType) && (input.Volumes == nil || input.Volumes.IsZero()) {
		return input, errors.Wrapf(httperrors.ErrMissingParameter,
			"volumes (data disk) is required for llm_type %q", input.LLMType)
	}
	applySkuCgroupLimitDefaults(input)

	drv, err := GetLLMContainerDriverWithError(api.LLMContainerType(input.LLMType))
	if err != nil {
		return input, errors.Wrap(err, "get container driver")
	}
	input, err = drv.ValidateLLMSkuCreateData(ctx, userCred, input)
	if err != nil {
		return input, errors.Wrap(err, "validate create input")
	}
	if isLocalPathSkuCreate(input) {
		if err := ValidateLocalPathSkuCreate(input); err != nil {
			return input, err
		}
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(input.Devices); err != nil {
			return input, err
		}
		resolved, err := resolvePreferHosts(ctx, userCred, input.PreferHosts)
		if err != nil {
			return input, err
		}
		input.PreferHosts = resolved
	}
	importInput, err := resolveLLMSkuImport(input)
	if err != nil {
		return input, err
	}
	if importInput != nil {
		input.Status = api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL
	} else {
		input.Status = api.STATUS_READY
	}
	return input, nil
}

// validateLLMModelCategoryForBackend rejects category/backend combinations that
// cannot be enabled by the container drivers.
func validateLLMModelCategoryForBackend(llmType string, cats []string) error {
	if llmType == string(api.LLM_CONTAINER_OLLAMA) || llmType == "" {
		return nil
	}
	if api.IsLLMEmbeddingCategory(cats) && api.IsLLMRerankerCategory(cats) {
		return errors.Wrap(httperrors.ErrInputParameter,
			"a model cannot be both embedding and reranker")
	}
	return nil
}

func (sku *SLLMSku) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := sku.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return errors.Wrap(err, "SSharableVirtualResourceBase.CustomizeCreate")
	}
	sku.Categories = api.NormalizeLLMModelCategories(sku.Categories)
	return nil
}

func (sku *SLLMSku) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	sku.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.LLMSkuCreateInput{}
	if data == nil || data.Unmarshal(&input) != nil || input.ModelSpec == nil {
		return
	}
	if err := sku.StartCreateTask(ctx, userCred, data); err != nil {
		log.Errorf("SLLMSku.PostCreate start task failed: %s", err)
		sku.SetStatus(ctx, userCred, api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED, err.Error())
	}
}

func (sku *SLLMSku) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	params, _ := data.(*jsonutils.JSONDict)
	if params == nil {
		params = jsonutils.NewDict()
	}
	importInput := api.InstantModelImportInput{}
	if err := params.Unmarshal(&importInput, "import_input"); err != nil {
		createdInput := api.LLMSkuCreateInput{}
		if err := params.Unmarshal(&createdInput); err != nil {
			return errors.Wrap(err, "unmarshal LLMSkuCreateInput")
		}
		derived, err := resolveLLMSkuImport(&createdInput)
		if err != nil {
			return errors.Wrap(err, "resolve sku import")
		}
		if derived == nil {
			return errors.Wrap(httperrors.ErrInputParameter, "missing model import input")
		}
		importInput = *derived
		params.Set("import_input", jsonutils.Marshal(importInput))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMSkuCreateTask", sku, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMSkuCreateTask")
	}
	return task.ScheduleRun(nil)
}

// GetLLMImageId returns the primary image id for this SKU. Delegates to driver.
func (sku *SLLMSku) GetLLMImageId() string {
	return sku.GetLLMContainerDriver().GetPrimaryImageId(sku)
}

// GetMountedModels returns mounted model ids (from Ollama or Vllm spec). Delegates to instant-model driver; returns nil for drivers that do not support instant models (e.g. Dify).
func (sku *SLLMSku) GetMountedModels() []string {
	drv, err := GetLLMContainerInstantModelDriver(api.LLMContainerType(sku.LLMType))
	if err != nil {
		return nil
	}
	return drv.GetMountedModels(sku)
}

// EstimateVramClaimMb returns heuristic VRAM (MiB) from the largest mounted
// InstantModel's weight_size_bytes, including KV reserve for effective
// max_model_len. 0 means unknown / no mounted weights.
func (sku *SLLMSku) EstimateVramClaimMb() int {
	if sku == nil {
		return 0
	}
	return EstimateVramClaimMbFromMountedModels(sku.LLMType, sku.GetMountedModels(), effectiveMaxModelLen(sku))
}

// EstimateModelVramMb returns weight+framework VRAM (MiB) without KV reserve.
// Used as the numerator for auto gpu-memory-utilization; HAMI slice sizing
// continues to use EstimateVramClaimMb (WithContext).
func (sku *SLLMSku) EstimateModelVramMb() int {
	if sku == nil {
		return 0
	}
	return EstimateModelVramMbFromMountedModels(sku.LLMType, sku.GetMountedModels())
}

// isolatedDeviceVramPlan selects the GPUStack whole-card claim (no KV) and the
// configured gpu-memory-utilization, while HAMI keeps the KV-inclusive claim.
func (sku *SLLMSku) isolatedDeviceVramPlan() isolatedDeviceVramPlan {
	if sku == nil {
		return isolatedDeviceVramPlan{}
	}
	return isolatedDeviceVramPlan{
		HamiClaimMb:          sku.EstimateVramClaimMb(),
		WholeCardClaimMb:     sku.EstimateModelVramMb(),
		WholeCardUtilization: skuGpuMemoryUtilization(sku),
	}
}

// effectiveMaxModelLen returns SKU-configured max-model-len, or the platform default.
func effectiveMaxModelLen(sku *SLLMSku) int {
	if sku == nil {
		return api.LLM_DEFAULT_CONTEXT_TOKENS
	}
	keys := []string{"max-model-len"}
	if val, ok := tokenLimitFromBackendParameters(sku.BackendParameters, keys); ok && val > 0 {
		return int(val)
	}
	switch api.LLMContainerType(sku.LLMType) {
	case api.LLM_CONTAINER_VLLM:
		if sku.LLMSpec != nil && sku.LLMSpec.Vllm != nil {
			for _, arg := range sku.LLMSpec.Vllm.CustomizedArgs {
				if arg == nil || !runtimeArgKeyIn(arg.Key, keys) {
					continue
				}
				if val, ok := parseTokenLimitValue(arg.Value); ok && val > 0 {
					return int(val)
				}
			}
		}
	case api.LLM_CONTAINER_SGLANG:
		if val, ok := sglangRuntimeIntArg(sku, keys); ok && val > 0 {
			return int(val)
		}
	}
	return api.LLM_DEFAULT_CONTEXT_TOKENS
}

func maxMountedInstantModelWeightBytes(modelIds []string) int64 {
	var maxWeight int64
	for _, id := range modelIds {
		if strings.TrimSpace(id) == "" {
			continue
		}
		obj, err := GetInstantModelManager().FetchById(id)
		if err != nil {
			continue
		}
		if w := obj.(*SInstantModel).WeightSizeBytes; w > maxWeight {
			maxWeight = w
		}
	}
	return maxWeight
}

// EstimateModelVramMbFromMountedModels estimates model VRAM without KV reserve.
func EstimateModelVramMbFromMountedModels(llmType string, modelIds []string) int {
	return vram.EstimateClaimMb(maxMountedInstantModelWeightBytes(modelIds), llmType)
}

// EstimateVramClaimMbFromMountedModels estimates VRAM from InstantModel ids.
func EstimateVramClaimMbFromMountedModels(llmType string, modelIds []string, maxModelLen int) int {
	return vram.EstimateClaimMbWithContext(maxMountedInstantModelWeightBytes(modelIds), llmType, maxModelLen)
}

// EstimateVramClaimMbFromInstantModels uses already-fetched InstantModel rows.
func EstimateVramClaimMbFromInstantModels(llmType string, modelIds []string, models map[string]SInstantModel, maxModelLen int) int {
	var maxWeight int64
	for _, id := range modelIds {
		if m, ok := models[id]; ok && m.WeightSizeBytes > maxWeight {
			maxWeight = m.WeightSizeBytes
		}
	}
	return vram.EstimateClaimMbWithContext(maxWeight, llmType, maxModelLen)
}

func (sku *SLLMSku) GetLLMContainerDriver() ILLMContainerDriver {
	return GetLLMContainerDriver(api.LLMContainerType(sku.LLMType))
}

// buildSchedulableIsolatedDevices builds IsolatedDeviceConfig list for forecast
// using the same path as pod create (GPU/NPU + SharingMode + MemoryMb).
func buildSchedulableIsolatedDevices(devices *api.Devices, plan isolatedDeviceVramPlan) ([]*computeapi.IsolatedDeviceConfig, int, error) {
	isoDevs, err := buildIsolatedDeviceConfigs(devices, plan)
	if err != nil {
		return nil, 0, err
	}
	perDevMinMb := 0
	for _, d := range isoDevs {
		if d.MemoryMb > perDevMinMb {
			perDevMinMb = d.MemoryMb
		}
	}
	return isoDevs, perDevMinMb, nil
}

func skuForecastDisks(volumes *api.Volumes) []*computeapi.DiskConfig {
	if volumes == nil || volumes.IsZero() {
		return nil
	}
	disks := make([]*computeapi.DiskConfig, 0, len(*volumes))
	for idx, volume := range *volumes {
		disks = append(disks, &computeapi.DiskConfig{
			DiskType: "data",
			SizeMb:   volume.SizeMB,
			Index:    idx,
		})
	}
	return disks
}

// schedulableCheckMissingVram is true when neither InstantModel weights nor a
// per-device memory_mb produced a VRAM floor. Local-path SKUs mount a host
// directory and have no InstantModel weight; exclusive whole cards do not need
// that estimate and should continue to host forecast.
func schedulableCheckMissingVram(sku *SLLMSku, vramClaimMb, perDevMinMb int) bool {
	if vramClaimMb > 0 || perDevMinMb > 0 {
		return false
	}
	return !SkuHasLocalHostPathModel(sku)
}

// PerformSchedulableCheck is POST /llm_skus/{id}/schedulable-check.
// Specs are read from the SKU (cpu/memory/volumes/devices). SKUs without
// GPUs still run forecast for CPU/memory/disk. Isolated-device VRAM checks
// apply only when devices are configured.
func (sku *SLLMSku) PerformSchedulableCheck(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSchedulableCheckInput,
) (*api.LLMSchedulableCheckOutput, error) {
	gpuCount := 0
	if sku.Devices != nil {
		gpuCount = len(*sku.Devices)
	}
	out := &api.LLMSchedulableCheckOutput{
		GpuCount: gpuCount,
	}
	if input.Nnodes == 1 {
		out.Reason = "nnodes=1 is not distributed inference; omit nnodes for auto planning"
		return out, nil
	}
	if input.Nnodes < 0 {
		out.Reason = "nnodes must be >= 0"
		return out, nil
	}
	if input.PipelineParallelSize < 0 {
		out.Reason = "pipeline_parallel_size must be >= 0"
		return out, nil
	}

	parallelMode := api.NormalizeLLMDeploymentParallelMode(input.ParallelMode)
	out.ParallelMode = parallelMode

	// Validate parallel_mode against SKU backend pins and backend support.
	if gpuCount > 0 {
		user := llmscheduler.ParseUserParallelism(sku.BackendParameters)
		if input.PipelineParallelSize > 0 {
			pp := input.PipelineParallelSize
			user.PP = &pp
		}
		if api.LLMContainerType(sku.LLMType) == api.LLM_CONTAINER_SGLANG && parallelMode == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP {
			out.Schedulable = false
			out.Reason = "SGLang 不支持数据并行(DP)，请使用 parallel_mode=pp"
			return out, nil
		}
		if parallelMode == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP {
			if user.PP != nil && *user.PP >= 2 {
				out.Schedulable = false
				out.Reason = fmt.Sprintf("parallel_mode=dp 与 SKU backend_parameters 中的 --pipeline-parallel-size %d 冲突，请移除其一", *user.PP)
				return out, nil
			}
		}
		if parallelMode == api.LLM_DEPLOYMENT_PARALLEL_MODE_PP {
			if user.DP != nil && *user.DP >= 2 {
				out.Schedulable = false
				out.Reason = fmt.Sprintf("parallel_mode=pp 与 SKU backend_parameters 中的 --data-parallel-size %d 冲突，请移除其一", *user.DP)
				return out, nil
			}
		}
	}

	var isoDevs []*computeapi.IsolatedDeviceConfig
	if gpuCount > 0 {
		plan := sku.isolatedDeviceVramPlan()
		vramClaimMb := plan.HamiClaimMb
		out.VramClaimMb = vramClaimMb

		var err error
		var perDevMinMb int
		isoDevs, perDevMinMb, err = buildSchedulableIsolatedDevices(sku.Devices, plan)
		if err != nil {
			out.Reason = err.Error()
			return out, nil
		}
		out.PerDevMinMb = perDevMinMb
		if schedulableCheckMissingVram(sku, vramClaimMb, perDevMinMb) {
			out.Reason = "Auto VRAM calculation failed — mounted instant models have unknown weight (not yet backfilled)"
			return out, nil
		}
	}

	schedInput := &schedulerapi.ScheduleInput{
		ServerConfig: schedulerapi.ServerConfig{
			ServerConfigs: &computeapi.ServerConfigs{
				Hypervisor:      computeapi.HYPERVISOR_POD,
				Count:           1,
				IsolatedDevices: isoDevs,
				Disks:           skuForecastDisks(sku.Volumes),
			},
			Ncpu:   sku.Cpu,
			Memory: sku.Memory,
		},
	}

	s := auth.GetAdminSession(ctx, "")
	canCreate, raw, err := schedulermodules.SchedManager.DoScheduleForecast(s, schedInput, 1)
	if err != nil {
		return nil, errors.Wrap(err, "scheduler forecast")
	}

	out.Schedulable = canCreate
	out.Reason = "Scheduler forecast completed — see hosts for qualifying candidates"
	if canCreate && gpuCount > 0 {
		// A single host fits; expose the trivial one-node layout explicitly.
		out.DistributedNnodes = 1
		out.GpusPerNode = gpuCount
	}

	candidates, _ := raw.GetArray("candidates")
	out.TotalGpuHosts = len(candidates)
	for _, c := range candidates {
		hostID, _ := c.GetString("host_id")
		hostName, _ := c.GetString("name")
		out.Hosts = append(out.Hosts, api.LLMSchedulableHostInfo{
			HostId:   hostID,
			HostName: hostName,
		})
		if hostID != "" {
			out.QualifiedHosts++
		}
	}

	if canCreate && input.Nnodes < 2 {
		if input.PipelineParallelSize <= 1 {
			return out, nil
		}
		pp := input.PipelineParallelSize
		if err := llmscheduler.ValidateSingleNodePipelineParallel(gpuCount, &pp); err == nil {
			out.TensorParallelSize = gpuCount / pp
			out.PipelineParallelSize = pp
			out.DataParallelSize = 1
			return out, nil
		} else if !input.DistributedInference {
			out.Schedulable = false
			out.Reason = err.Error()
			return out, nil
		}
		// The single-node layout cannot realize this pipeline pin; fall
		// through to the multi-host forecast, which validates the pin per
		// candidate layout.
	}

	if !canCreate {
		var reasons []string
		notAllow, _ := raw.GetArray("not_allow_reasons")
		for _, r := range notAllow {
			if s, _ := r.GetString(); s != "" {
				reasons = append(reasons, s)
			}
		}
		if len(reasons) > 0 {
			out.Reason = fmt.Sprintf("not schedulable: %s", reasons[0])
		} else {
			out.Reason = "not schedulable — no host satisfies all predicates"
		}
		fc, _ := raw.Get("filtered_candidates")
		out.FilteredCandidates = fc
	}

	if (input.DistributedInference || input.Nnodes >= 2) && gpuCount > 0 {
		if input.Nnodes >= 2 {
			// A pinned node count never accepts a single-host fit.
			out.Schedulable = false
			out.DistributedNnodes = 0
			out.GpusPerNode = 0
			out.Reason = "pinned nnodes requires a multi-host layout"
		}
		tmpl, ok := skuDeviceTemplate(sku)
		if ok && strings.TrimSpace(tmpl.Model) != "" {
			hosts, invErr := ListFreeGPUInventoryByModel(ctx, userCred, tmpl.Model)
			if invErr == nil {
				distPlan, planErr := llmscheduler.PlanReplica(llmscheduler.PlanOptions{
					TotalGPUs:             gpuCount,
					DeviceTemplate:        tmpl,
					PreferHosts:           input.PreferHosts,
					PlacementStrategy:     input.PlacementStrategy,
					DistributedInference:  true,
					ForceSingleGPUPerNode: skuForceSingleGPUPerNode(sku),
					ForceNNodes:           input.Nnodes,
					Backend:               sku.LLMType,
					BackendParameters:     sku.BackendParameters,
					PipelineParallelSize:  input.PipelineParallelSize,
					ParallelMode:          parallelMode,
					Hosts:                 hosts,
				})
				if planErr == nil && distPlan != nil && distPlan.Distributed {
					out.Schedulable = true
					out.DistributedNnodes = distPlan.NNodes
					out.GpusPerNode = distPlan.GpusPerNode
					layout := ""
					if distPlan.Parallelism != nil {
						out.TensorParallelSize = distPlan.Parallelism.TP
						out.PipelineParallelSize = distPlan.Parallelism.PP
						out.DataParallelSize = distPlan.Parallelism.DP
						out.DistShape = distPlanShape(distPlan.Parallelism)
						modeLabel := "PP"
						if parallelMode == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP {
							modeLabel = "DP"
						}
						layout = fmt.Sprintf(" (%s: TP=%d / PP=%d / DP=%d)",
							modeLabel, distPlan.Parallelism.TP, distPlan.Parallelism.PP, distPlan.Parallelism.DP)
					}
					if input.Nnodes >= 2 {
						out.Reason = fmt.Sprintf(
							"%d 节点 × %d 卡%s", distPlan.NNodes, distPlan.GpusPerNode, layout,
						)
					} else {
						out.Reason = fmt.Sprintf(
							"单机放不下；多机规划 %d 节点 × %d 卡%s",
							distPlan.NNodes, distPlan.GpusPerNode, layout,
						)
					}
					byHost := make(map[string]llmscheduler.HostGPUInventory, len(hosts))
					for _, h := range hosts {
						byHost[h.HostId] = h
					}
					out.Hosts = make([]api.LLMSchedulableHostInfo, 0, len(distPlan.Members))
					out.QualifiedHosts = 0
					for _, m := range distPlan.Members {
						info := api.LLMSchedulableHostInfo{
							HostId:       m.HostId,
							HostName:     m.HostName,
							HostIp:       m.HostIp,
							GpuAvailable: distPlan.GpusPerNode,
						}
						if h, ok := byHost[m.HostId]; ok {
							if info.HostName == "" {
								info.HostName = h.HostName
							}
							info.HostIp = h.HostIp
							info.BestGpuModel = h.GpuModel
							info.BestGpuVramMb = h.MinVRAMMb
						}
						out.Hosts = append(out.Hosts, info)
						out.QualifiedHosts++
					}
					out.TotalGpuHosts = out.QualifiedHosts
					out.FilteredCandidates = nil
				} else if input.Nnodes >= 2 {
					// The user pinned a node count; a single-host fit is not the
					// requested layout, so surface the planning error instead.
					if planErr != nil {
						out.Reason = planErr.Error()
					} else {
						out.Reason = "pinned nnodes requires a multi-host layout, but the plan is single-host"
					}
					out.Hosts = nil
					out.QualifiedHosts = 0
				} else if input.PipelineParallelSize > 1 {
					// The single-host layout could not realize the pinned PP;
					// surface why the multi-host fallback failed too.
					if planErr != nil {
						out.Schedulable = false
						out.Reason = planErr.Error()
						out.Hosts = nil
						out.QualifiedHosts = 0
					}
				}
			} else if input.Nnodes >= 2 {
				out.Reason = fmt.Sprintf("list free GPU inventory: %v", invErr)
			}
		}
	}

	return out, nil
}

// distPlanShape maps a derived parallel layout to the vLLM multi-node shape
// name exposed by schedulable-check. Single-node and pure DP/PP layouts return
// an empty shape.
func distPlanShape(p *llmscheduler.PlanParallelism) string {
	if p == nil || p.DP <= 1 && p.PP <= 1 {
		return ""
	}
	if p.DP > 1 && p.PP > 1 {
		return string(llmscheduler.ShapeNested)
	}
	if p.DP > 1 {
		return string(llmscheduler.ShapeDPOnly)
	}
	return string(llmscheduler.ShapeMPOnly)
}

func ValidateLLMSkuReadyForUse(sku *SLLMSku) error {
	if sku == nil {
		return errors.Wrap(httperrors.ErrInputParameter, "empty llm_sku")
	}
	if sku.Status != api.STATUS_READY {
		return errors.Wrapf(httperrors.ErrInvalidStatus, "llm_sku %s status is %s", sku.Name, sku.Status)
	}
	return nil
}

func (sku *SLLMSku) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSkuUpdateInput) (api.LLMSkuUpdateInput, error) {
	var err error
	input.LLMSkuBaseUpdateInput, err = sku.SLLMSkuBase.ValidateUpdateData(ctx, userCred, query, input.LLMSkuBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate LLMSkuBaseUpdateInput")
	}
	if !api.IsLLMOptionalDiskType(sku.LLMType) && (input.Volumes == nil || input.Volumes.IsZero()) {
		return input, errors.Wrapf(httperrors.ErrMissingParameter,
			"volumes (data disk) is required for llm_type %q", sku.LLMType)
	}

	if SkuHasLocalHostPathModel(sku) && input.Devices != nil {
		if err := ValidateLocalPathHamiDevicesRequireMemoryMb(input.Devices); err != nil {
			return input, err
		}
	}

	if err := validateLocalPathSkuUpdatePreferHosts(ctx, userCred, sku, &input); err != nil {
		return input, err
	}

	if input.Categories != nil {
		if err := api.ValidateLLMModelCategories(*input.Categories); err != nil {
			return input, errors.Wrap(httperrors.ErrInputParameter, err.Error())
		}
		normalized := api.NormalizeLLMModelCategories(*input.Categories)
		if err := validateLLMModelCategoryForBackend(sku.LLMType, normalized); err != nil {
			return input, err
		}
		input.Categories = &normalized
	}

	if sku.LLMSpec == nil {
		return input, nil
	}
	drv := sku.GetLLMContainerDriver()
	updateInput, err := drv.ValidateLLMSkuUpdateData(ctx, userCred, sku, &input)
	if err != nil {
		return input, errors.Wrap(err, "validate update spec")
	}
	return *updateInput, nil
}

func (sku *SLLMSku) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	count, err := GetLLMManager().Query().Equals("llm_sku_id", sku.Id).CountWithError()
	if err != nil {
		return errors.Wrap(err, "fetch llm")
	}
	if count > 0 {
		return errors.Wrap(errors.ErrNotSupported, "This sku is currently in use by LLM")
	}
	return nil
}

func fetchTemplates(ctx context.Context, userCred mcclient.TokenCredential, templateIds []string) (map[string]imageapi.ImageDetails, error) {
	s := auth.GetSession(ctx, userCred, "")
	params := mcclientoptions.BaseListOptions{}
	params.Id = templateIds
	limit := len(templateIds)
	params.Limit = &limit
	params.Scope = "maxallowed"
	results, err := imagemodules.Images.List(s, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "Images.List")
	}
	templates := make(map[string]imageapi.ImageDetails)
	for i := range results.Data {
		tmpl := imageapi.ImageDetails{}
		err := results.Data[i].Unmarshal(&tmpl)
		if err == nil {
			templates[tmpl.Id] = tmpl
		}
	}
	return templates, nil
}
