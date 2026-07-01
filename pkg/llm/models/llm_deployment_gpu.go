package models

import (
	"context"
	"math"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/llm/utils/vram"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	autoGpuMemoryUtilizationSafetyFactor = 1.10
	autoGpuMemoryUtilizationMin          = 0.05
	autoGpuMemoryUtilizationMax          = 0.95

	// vLLM derives an unset max-model-len from model config. When auto GPU
	// memory utilization is enabled, inject a conservative cap so the heuristic
	// VRAM estimate is not paired with an unexpectedly large context.
	autoGpuMemoryUtilizationDefaultContextTokens = int64(8192)

	sglangAutoGpuMemoryMetadataReserveMB = 512
)

func boolPtrValue(v *bool) bool {
	return v != nil && *v
}

func deploymentAutoGpuMemoryUtilizationEnabled(auto *bool, llmType string) bool {
	if auto != nil {
		return *auto
	}
	_, ok := gpuMemoryUtilizationRuntimeArgKey(llmType)
	return ok
}

func defaultDeploymentAutoGpuMemoryUtilization(input *api.LLMDeploymentCreateInput, sku *SLLMSku, llmType string) {
	if input == nil || input.GpuMemoryUtilization != nil || input.AutoGpuMemoryUtilization != nil {
		return
	}
	if !deploymentAutoGpuMemoryUtilizationEnabled(nil, llmType) {
		return
	}
	if !skuCanAutoGpuMemoryUtilization(sku) {
		return
	}
	enabled := true
	input.AutoGpuMemoryUtilization = &enabled
}

// disableDeploymentAutoGpuMemoryUtilizationForLocalPath forces auto GPU memory utilization off for host-mounted SKUs.
func disableDeploymentAutoGpuMemoryUtilizationForLocalPath(input *api.LLMDeploymentCreateInput, sku *SLLMSku) {
	if input == nil || sku == nil || !SkuHasLocalHostPathModel(sku) {
		return
	}
	disabled := false
	input.AutoGpuMemoryUtilization = &disabled
}

// skuCanAutoGpuMemoryUtilization reports whether auto GPU memory utilization can be derived for sku.
// Host-mounted local_path SKUs always opt out; use gpu_memory_utilization manually when needed.
func skuCanAutoGpuMemoryUtilization(sku *SLLMSku) bool {
	if sku == nil {
		return true
	}
	return !SkuHasLocalHostPathModel(sku)
}

func validateDeploymentGpuMemoryUtilization(util *float64, auto *bool, llmType string) error {
	needsRuntimeArg := util != nil || boolPtrValue(auto)
	if util != nil {
		if *util <= 0 || *util > 1 {
			return errors.Wrap(httperrors.ErrInputParameter, "gpu_memory_utilization must be > 0 and <= 1")
		}
	}
	if util != nil && boolPtrValue(auto) {
		return errors.Wrap(httperrors.ErrInputParameter, "gpu_memory_utilization and auto_gpu_memory_utilization are mutually exclusive")
	}
	if !needsRuntimeArg {
		return nil
	}
	if _, ok := gpuMemoryUtilizationRuntimeArgKey(llmType); !ok {
		return errors.Wrapf(httperrors.ErrInputParameter, "gpu_memory_utilization is not supported for llm_type %q", llmType)
	}
	return nil
}

func gpuMemoryUtilizationRuntimeArgKey(llmType string) (string, bool) {
	switch api.LLMContainerType(llmType) {
	case api.LLM_CONTAINER_VLLM:
		return "gpu-memory-utilization", true
	case api.LLM_CONTAINER_SGLANG:
		return "mem-fraction-static", true
	default:
		return "", false
	}
}

func calculateAutoGpuMemoryUtilization(requiredVramMB int64, gpuMemoryMB int64, tensorParallelSize int) (float64, error) {
	if requiredVramMB <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "mounted model gpu_memory_required is empty")
	}
	return calculateAutoGpuMemoryUtilizationFromPerGPURequired(
		float64(requiredVramMB)/float64(normalizeTensorParallelSize(tensorParallelSize)),
		gpuMemoryMB,
	)
}

func instantModelEstimatedVramRequirementMB(model *SInstantModel) int64 {
	if model == nil {
		return 0
	}
	return int64(vram.EstimateClaimMb(model.WeightSizeBytes, model.LlmType))
}

func runtimeArgKeyIn(key string, keys []string) bool {
	key = normalizeRuntimeArgKey(key)
	for _, candidate := range keys {
		if key == candidate {
			return true
		}
	}
	return false
}

func runtimeHasExplicitTokenLimit(sku *SLLMSku) bool {
	if sku == nil {
		return false
	}
	switch api.LLMContainerType(sku.LLMType) {
	case api.LLM_CONTAINER_VLLM:
		return runtimeHasExplicitArg(sku, []string{"max-model-len"})
	default:
		return false
	}
}

func runtimeHasExplicitArg(sku *SLLMSku, keys []string) bool {
	if sku == nil {
		return false
	}
	if backendParametersContainRuntimeArg(sku.BackendParameters, keys) {
		return true
	}
	switch api.LLMContainerType(sku.LLMType) {
	case api.LLM_CONTAINER_VLLM:
		if sku.LLMSpec == nil || sku.LLMSpec.Vllm == nil {
			return false
		}
		for _, arg := range sku.LLMSpec.Vllm.CustomizedArgs {
			if arg != nil && runtimeArgKeyIn(arg.Key, keys) {
				return true
			}
		}
	case api.LLM_CONTAINER_SGLANG:
		if sku.LLMSpec == nil || sku.LLMSpec.SGLang == nil {
			return false
		}
		for _, arg := range sku.LLMSpec.SGLang.CustomizedArgs {
			if arg != nil && runtimeArgKeyIn(arg.Key, keys) {
				return true
			}
		}
	}
	return false
}

func backendParametersContainRuntimeArg(items []string, keys []string) bool {
	for i := range items {
		argKey, _, ok := splitBackendParameterFlag(items[i])
		if ok && runtimeArgKeyIn(argKey, keys) {
			return true
		}
	}
	return false
}

func tokenLimitFromBackendParameters(items []string, keys []string) (int64, bool) {
	var tokenLimit int64
	found := false
	for i := range items {
		key, value, ok := splitBackendParameterFlag(items[i])
		if !ok || !runtimeArgKeyIn(key, keys) {
			continue
		}
		if value == "" && i+1 < len(items) {
			next := strings.TrimSpace(items[i+1])
			if next == "-1" || !strings.HasPrefix(next, "-") {
				value = next
			}
		}
		if val, ok := parseTokenLimitValue(value); ok {
			tokenLimit = val
			found = true
		}
	}
	return tokenLimit, found
}

func splitBackendParameterFlag(item string) (string, string, bool) {
	item = strings.TrimSpace(item)
	if item == "" || strings.HasPrefix(item, "-") && !strings.HasPrefix(item, "--") {
		return "", "", false
	}
	item = strings.TrimSpace(strings.TrimPrefix(item, "--"))
	if item == "" {
		return "", "", false
	}
	key := item
	value := ""
	if idx := strings.Index(item, "="); idx >= 0 {
		key = strings.TrimSpace(item[:idx])
		value = item[idx+1:]
	} else if fields := strings.Fields(item); len(fields) > 1 {
		key = fields[0]
		value = strings.TrimSpace(item[len(key):])
	}
	return key, trimArgumentValue(value), true
}

func normalizeRuntimeArgKey(key string) string {
	return strings.TrimPrefix(strings.TrimSpace(key), "--")
}

func trimArgumentValue(value string) string {
	value = strings.TrimSpace(value)
	return strings.Trim(value, `"'`)
}

func parseTokenLimitValue(value string) (int64, bool) {
	value = trimArgumentValue(value)
	if value == "" {
		return 0, false
	}
	if strings.EqualFold(value, "auto") {
		return 0, true
	}
	if val, err := strconv.ParseInt(value, 10, 64); err == nil {
		if val <= 0 {
			return 0, true
		}
		return val, true
	}

	multiplier := float64(1)
	number := value
	switch value[len(value)-1] {
	case 'k':
		multiplier = 1000
		number = value[:len(value)-1]
	case 'K':
		multiplier = 1024
		number = value[:len(value)-1]
	case 'm':
		multiplier = 1000 * 1000
		number = value[:len(value)-1]
	case 'M':
		multiplier = 1024 * 1024
		number = value[:len(value)-1]
	case 'g':
		multiplier = 1000 * 1000 * 1000
		number = value[:len(value)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		number = value[:len(value)-1]
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return int64(math.Ceil(parsed * multiplier)), true
}

func calculateAutoGpuMemoryUtilizationFromPerGPURequired(perGPURequiredMB float64, gpuMemoryMB int64) (float64, error) {
	if gpuMemoryMB <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "gpu memory_mb is empty")
	}
	raw := perGPURequiredMB * autoGpuMemoryUtilizationSafetyFactor / float64(gpuMemoryMB)
	if raw > autoGpuMemoryUtilizationMax {
		return 0, errors.Wrapf(httperrors.ErrInputParameter,
			"model requires %.2f GPU memory utilization, exceeds max %.2f", raw, autoGpuMemoryUtilizationMax)
	}
	if raw < autoGpuMemoryUtilizationMin {
		raw = autoGpuMemoryUtilizationMin
	}
	return math.Ceil(raw*100) / 100, nil
}

func calculateDeploymentAutoGpuMemoryUtilization(sku *SLLMSku, requiredVramMB int64, gpuMemoryMB int64, tensorParallelSize int) (float64, error) {
	if sku != nil && api.LLMContainerType(sku.LLMType) == api.LLM_CONTAINER_SGLANG {
		return calculateSGLangAutoGpuMemoryUtilization(sku, requiredVramMB, gpuMemoryMB, tensorParallelSize)
	}
	return calculateAutoGpuMemoryUtilization(requiredVramMB, gpuMemoryMB, tensorParallelSize)
}

func calculateSGLangAutoGpuMemoryUtilization(sku *SLLMSku, requiredVramMB int64, gpuMemoryMB int64, tensorParallelSize int) (float64, error) {
	modelUtilization, err := calculateAutoGpuMemoryUtilization(requiredVramMB, gpuMemoryMB, tensorParallelSize)
	if err != nil {
		return 0, err
	}
	runtimeUtilization, err := calculateSGLangMemFractionStatic(sku, gpuMemoryMB, tensorParallelSize)
	if err != nil {
		return 0, err
	}
	if runtimeUtilization > modelUtilization {
		return runtimeUtilization, nil
	}
	return modelUtilization, nil
}

func calculateSGLangMemFractionStatic(sku *SLLMSku, gpuMemoryMB int64, tensorParallelSize int) (float64, error) {
	if gpuMemoryMB <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "gpu memory_mb is empty")
	}
	tpSize := int64(normalizeTensorParallelSize(tensorParallelSize))
	if val, ok := sglangRuntimeIntArg(sku, []string{"tp-size", "tensor-parallel-size"}); ok && val > 0 {
		tpSize = val
	}
	ppSize := int64(1)
	if val, ok := sglangRuntimeIntArg(sku, []string{"pp-size", "pipeline-parallel-size"}); ok && val > 0 {
		ppSize = val
	}
	chunkedPrefillSize, cudaGraphMaxBS := sglangRuntimeReserveDefaults(gpuMemoryMB, tpSize)
	if val, ok := sglangRuntimeIntArg(sku, []string{"chunked-prefill-size"}); ok {
		chunkedPrefillSize = val
	}
	if val, ok := sglangRuntimeIntArg(sku, []string{"cuda-graph-max-bs", "cuda-graph-max-bs-decode"}); ok && val > 0 {
		cudaGraphMaxBS = val
	}

	reservedMemMB := float64(sglangAutoGpuMemoryMetadataReserveMB)
	if chunkedPrefillSize > 0 {
		reservedMemMB += float64(maxInt64(chunkedPrefillSize, 2048)) * 1.5
	} else if maxPrefillTokens, ok := sglangRuntimeIntArg(sku, []string{"max-prefill-tokens"}); ok && maxPrefillTokens > 0 {
		reservedMemMB += float64(maxInt64(maxPrefillTokens, 2048)) * 1.5
	} else {
		reservedMemMB += 2048 * 1.5
	}
	reservedMemMB += float64(cudaGraphMaxBS) * 2
	reservedMemMB += float64(tpSize*ppSize) / 8 * 1024
	if gpuMemoryMB > 60*1024 && reservedMemMB < 10*1024 {
		reservedMemMB = 10 * 1024
	}

	utilization := (float64(gpuMemoryMB) - reservedMemMB) / float64(gpuMemoryMB)
	if utilization > autoGpuMemoryUtilizationMax {
		utilization = autoGpuMemoryUtilizationMax
	}
	if utilization < autoGpuMemoryUtilizationMin {
		utilization = autoGpuMemoryUtilizationMin
	}
	return math.Round(utilization*1000) / 1000, nil
}

func sglangRuntimeReserveDefaults(gpuMemoryMB int64, tpSize int64) (int64, int64) {
	if gpuMemoryMB < 20*1024 {
		return 2048, 8
	}
	if gpuMemoryMB < 35*1024 {
		if tpSize < 4 {
			return 2048, 24
		}
		return 2048, 80
	}
	if gpuMemoryMB < 60*1024 {
		if tpSize < 4 {
			return 4096, 32
		}
		return 4096, 160
	}
	if gpuMemoryMB < 160*1024 {
		if tpSize < 4 {
			return 8192, 256
		}
		return 8192, 512
	}
	return 16384, 512
}

func sglangRuntimeIntArg(sku *SLLMSku, keys []string) (int64, bool) {
	if sku == nil {
		return 0, false
	}
	if val, ok := tokenLimitFromBackendParameters(sku.BackendParameters, keys); ok {
		return val, true
	}
	if sku.LLMSpec == nil || sku.LLMSpec.SGLang == nil {
		return 0, false
	}
	for _, arg := range sku.LLMSpec.SGLang.CustomizedArgs {
		if arg == nil || !runtimeArgKeyIn(arg.Key, keys) {
			continue
		}
		if val, ok := parseTokenLimitValue(arg.Value); ok {
			return val, true
		}
	}
	return 0, false
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func normalizeTensorParallelSize(tensorParallelSize int) int {
	if tensorParallelSize <= 0 {
		return 1
	}
	return tensorParallelSize
}

func buildDeploymentGpuMemoryLLMSpec(deploy *SLLMDeployment, sku *SLLMSku) (*api.LLMSpec, error) {
	if deploy == nil || sku == nil || deploy.GpuMemoryUtilization == nil {
		return nil, nil
	}
	return buildGpuMemoryUtilizationLLMSpec(sku.LLMType, *deploy.GpuMemoryUtilization)
}

func buildGpuMemoryUtilizationLLMSpec(llmType string, utilization float64) (*api.LLMSpec, error) {
	key, ok := gpuMemoryUtilizationRuntimeArgKey(llmType)
	if !ok {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "gpu_memory_utilization is not supported for llm_type %q", llmType)
	}
	value := formatGpuMemoryUtilization(utilization)
	switch api.LLMContainerType(llmType) {
	case api.LLM_CONTAINER_VLLM:
		return &api.LLMSpec{
			Vllm: &api.LLMSpecVllm{
				CustomizedArgs: []*api.VllmCustomizedArg{{Key: key, Value: value}},
			},
		}, nil
	case api.LLM_CONTAINER_SGLANG:
		return &api.LLMSpec{
			SGLang: &api.LLMSpecSGLang{
				CustomizedArgs: []*api.SGLangCustomizedArg{{Key: key, Value: value}},
			},
		}, nil
	default:
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "gpu_memory_utilization is not supported for llm_type %q", llmType)
	}
}

func buildAutoGpuMemoryUtilizationLLMSpec(sku *SLLMSku, utilization float64) (*api.LLMSpec, error) {
	if sku == nil {
		return nil, nil
	}
	spec, err := buildGpuMemoryUtilizationLLMSpec(sku.LLMType, utilization)
	if err != nil {
		return nil, err
	}
	switch api.LLMContainerType(sku.LLMType) {
	case api.LLM_CONTAINER_VLLM:
		if spec != nil && spec.Vllm != nil && !runtimeHasExplicitTokenLimit(sku) {
			spec.Vllm.CustomizedArgs = append(spec.Vllm.CustomizedArgs, &api.VllmCustomizedArg{
				Key:   "max-model-len",
				Value: strconv.FormatInt(autoGpuMemoryUtilizationDefaultContextTokens, 10),
			})
		}
	}
	return spec, nil
}

func formatGpuMemoryUtilization(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func BuildDeploymentResolvedGpuMemoryLLMSpec(ctx context.Context, userCred mcclient.TokenCredential, deploy *SLLMDeployment, sku *SLLMSku) (*api.LLMSpec, error) {
	if deploy == nil || sku == nil {
		return nil, nil
	}
	if deploy.GpuMemoryUtilization != nil {
		return buildDeploymentGpuMemoryLLMSpec(deploy, sku)
	}
	if !skuCanAutoGpuMemoryUtilization(sku) {
		return nil, nil
	}
	if !deploymentAutoGpuMemoryUtilizationEnabled(deploy.AutoGpuMemoryUtilization, sku.LLMType) {
		return nil, nil
	}
	tensorParallelSize := 1
	if sku.Devices != nil && len(*sku.Devices) > 0 {
		tensorParallelSize = len(*sku.Devices)
	}
	requiredVramMB, err := maxMountedModelVramRequirementMB(sku)
	if err != nil {
		return nil, err
	}
	gpuMemoryMB, err := minGpuMemoryMB(ctx, userCred, sku.Devices)
	if err != nil {
		return nil, err
	}
	utilization, err := calculateDeploymentAutoGpuMemoryUtilization(sku, requiredVramMB, gpuMemoryMB, tensorParallelSize)
	if err != nil {
		return nil, err
	}
	return buildAutoGpuMemoryUtilizationLLMSpec(sku, utilization)
}

func maxMountedModelVramRequirementMB(sku *SLLMSku) (int64, error) {
	if sku != nil && SkuHasLocalHostPathModel(sku) {
		return 0, httperrors.NewInputParameterError(
			"auto_gpu_memory_utilization is not supported for local_path SKU: set gpu_memory_utilization manually or omit both")
	}
	modelIds := sku.GetMountedModels()
	if len(modelIds) == 0 {
		return 0, httperrors.NewInputParameterError("auto_gpu_memory_utilization requires mounted models: configure mounted_models on the LLM SKU")
	}
	var maxRequiredVramMB int64
	for _, modelId := range modelIds {
		obj, err := GetInstantModelManager().FetchById(modelId)
		if err != nil {
			return 0, errors.Wrapf(err, "fetch InstantModel %s", modelId)
		}
		requiredVramMB := instantModelEstimatedVramRequirementMB(obj.(*SInstantModel))
		if requiredVramMB > maxRequiredVramMB {
			maxRequiredVramMB = requiredVramMB
		}
	}
	if maxRequiredVramMB <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "mounted model vram requirement is empty")
	}
	return maxRequiredVramMB, nil
}

func minGpuMemoryMB(ctx context.Context, userCred mcclient.TokenCredential, devices *api.Devices) (int64, error) {
	if devices == nil || len(*devices) == 0 {
		return 0, httperrors.NewInputParameterError("auto_gpu_memory_utilization requires GPU devices: configure GPU on the LLM SKU")
	}
	var minMemory int64
	for i := range *devices {
		memory, err := fetchMinIsolatedDeviceMemoryMB(ctx, userCred, (*devices)[i])
		if err != nil {
			return 0, err
		}
		if minMemory == 0 || memory < minMemory {
			minMemory = memory
		}
	}
	return minMemory, nil
}

func fetchMinIsolatedDeviceMemoryMB(ctx context.Context, userCred mcclient.TokenCredential, device api.Device) (int64, error) {
	_ = userCred
	session := auth.GetAdminSession(ctx, options.Options.Region)
	params := buildIsolatedDeviceMemoryParams(device)
	results, err := computemodules.IsolatedDevices.List(session, params)
	if err != nil {
		return 0, errors.Wrapf(err, "list isolated devices for %s", isolatedDeviceMemoryFilterDesc(device))
	}
	if len(results.Data) == 0 {
		return 0, errors.Wrapf(httperrors.ErrResourceNotFound,
			"unused isolated device not found for %s", isolatedDeviceMemoryFilterDesc(device))
	}
	memory, err := minIsolatedDeviceMemoryMB(results.Data)
	if err != nil {
		return 0, errors.Wrapf(err, "isolated devices for %s", isolatedDeviceMemoryFilterDesc(device))
	}
	return memory, nil
}

func buildIsolatedDeviceMemoryParams(device api.Device) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Set("unused", jsonutils.JSONTrue)
	params.Set("show_baremetal_isolated_devices", jsonutils.JSONTrue)
	setStringArrayParam(params, "dev_type", device.DevType)
	setStringArrayParam(params, "model", device.Model)
	setStringArrayParam(params, "device_path", device.DevicePath)
	return params
}

func setStringArrayParam(params *jsonutils.JSONDict, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	params.Set(key, jsonutils.NewArray(jsonutils.NewString(value)))
}

func minIsolatedDeviceMemoryMB(rows []jsonutils.JSONObject) (int64, error) {
	var minMemory int64
	for i := range rows {
		if rows[i] == nil || !rows[i].Contains("memory_size") {
			continue
		}
		memory, err := rows[i].Int("memory_size")
		if err != nil {
			return 0, errors.Wrapf(err, "read isolated device memory_size at row %d", i)
		}
		if memory <= 0 {
			continue
		}
		if minMemory == 0 || memory < minMemory {
			minMemory = memory
		}
	}
	if minMemory <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "isolated devices have empty memory_size")
	}
	return minMemory, nil
}

func isolatedDeviceMemoryFilterDesc(device api.Device) string {
	return jsonutils.Marshal(device).String()
}
