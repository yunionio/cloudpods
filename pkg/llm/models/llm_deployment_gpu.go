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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	autoGpuMemoryUtilizationSafetyFactor = 1.10
	autoGpuMemoryUtilizationMin          = 0.05
	autoGpuMemoryUtilizationMax          = 0.95
	instantModelDynamicVramRatio         = 0.15
	instantModelFixedVramMB              = 500
)

func boolPtrValue(v *bool) bool {
	return v != nil && *v
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

func calculateAutoGpuMemoryUtilizationForModelSize(modelSizeMB int64, gpuMemoryMB int64, tensorParallelSize int) (float64, error) {
	if modelSizeMB <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "mounted model size is empty")
	}
	tp := normalizeTensorParallelSize(tensorParallelSize)
	// Tensor parallel shards model weights, but runtime/KV/framework overhead is
	// still charged per GPU here to avoid underestimating heterogeneous multi-GPU deployments.
	perGPURequiredMB := float64(modelSizeMB)/float64(tp) +
		float64(modelSizeMB)*instantModelDynamicVramRatio +
		instantModelFixedVramMB
	return calculateAutoGpuMemoryUtilizationFromPerGPURequired(perGPURequiredMB, gpuMemoryMB)
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
	if !boolPtrValue(deploy.AutoGpuMemoryUtilization) {
		return nil, nil
	}
	tensorParallelSize := 1
	if sku.Devices != nil && len(*sku.Devices) > 0 {
		tensorParallelSize = len(*sku.Devices)
	}
	modelSizeMB, err := maxMountedModelSizeMB(sku)
	if err != nil {
		return nil, err
	}
	gpuMemoryMB, err := minGpuMemoryMB(ctx, userCred, sku.Devices)
	if err != nil {
		return nil, err
	}
	utilization, err := calculateAutoGpuMemoryUtilizationForModelSize(modelSizeMB, gpuMemoryMB, tensorParallelSize)
	if err != nil {
		return nil, err
	}
	return buildGpuMemoryUtilizationLLMSpec(sku.LLMType, utilization)
}

func maxMountedModelSizeMB(sku *SLLMSku) (int64, error) {
	modelIds := sku.GetMountedModels()
	if len(modelIds) == 0 {
		return 0, httperrors.NewInputParameterError("auto_gpu_memory_utilization requires mounted models: configure mounted_models on the LLM SKU")
	}
	var maxSize int64
	for _, modelId := range modelIds {
		obj, err := GetInstantModelManager().FetchById(modelId)
		if err != nil {
			return 0, errors.Wrapf(err, "fetch InstantModel %s", modelId)
		}
		sizeMB := int64(obj.(*SInstantModel).GetActualSizeMb())
		if sizeMB > maxSize {
			maxSize = sizeMB
		}
	}
	if maxSize <= 0 {
		return 0, errors.Wrap(httperrors.ErrInputParameter, "mounted model size is empty")
	}
	return maxSize, nil
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
