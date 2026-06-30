package models

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func isLocalPathSkuCreate(input *api.LLMSkuCreateInput) bool {
	if input == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(input.Source), api.LLM_MODEL_SOURCE_LOCAL_PATH)
}

// ValidateLocalPathSkuCreate validates SKU create requests that mount an on-host model directory.
func ValidateLocalPathSkuCreate(input *api.LLMSkuCreateInput) error {
	if input == nil {
		return errors.Wrap(httperrors.ErrInputParameter, "empty sku input")
	}
	llmType := strings.TrimSpace(input.LLMType)
	if llmType != string(api.LLM_CONTAINER_VLLM) && llmType != string(api.LLM_CONTAINER_SGLANG) {
		return errors.Wrapf(httperrors.ErrInputParameter, "local_path import supports vllm and sglang only, got %q", llmType)
	}
	localPath := strings.TrimSpace(input.LocalPath)
	if localPath == "" {
		return errors.Wrap(httperrors.ErrMissingParameter, "local_path is required for local_path source")
	}
	if !strings.HasPrefix(localPath, "/") {
		return errors.Wrap(httperrors.ErrInputParameter, "local_path must be an absolute path")
	}
	if input.ModelSpec != nil {
		return errors.Wrap(httperrors.ErrInputParameter, "model_spec is not allowed for local_path import")
	}
	if input.HostPaths == nil || input.HostPaths.IsZero() {
		return errors.Wrap(httperrors.ErrMissingParameter, "host_paths is required for local_path source")
	}
	if !hostPathsHasContainerMount(*input.HostPaths, 0) {
		return errors.Wrap(httperrors.ErrInputParameter, "host_paths must include a mount for container index 0")
	}
	if len(normalizePreferHostInputs(input.PreferHosts)) == 0 {
		return errors.Wrap(httperrors.ErrMissingParameter, "prefer_hosts is required for local_path source")
	}
	input.Source = api.LLM_MODEL_SOURCE_LOCAL_PATH
	input.LocalPath = localPath
	return nil
}

func hostPathsHasContainerMount(paths api.HostPaths, containerIndex int) bool {
	key := fmt.Sprintf("%d", containerIndex)
	for _, hp := range paths {
		if hp.IsZero() {
			continue
		}
		if hp.Containers == nil {
			continue
		}
		rel, ok := hp.Containers[key]
		if !ok || rel == nil {
			continue
		}
		if strings.TrimSpace(rel.MountPath) != "" {
			return true
		}
	}
	return false
}

// SkuHasLocalHostPathModel reports whether sku carries a host-mounted local model (no InstantModel import).
func SkuHasLocalHostPathModel(sku *SLLMSku) bool {
	if sku == nil {
		return false
	}
	if strings.TrimSpace(sku.Source) != api.LLM_MODEL_SOURCE_LOCAL_PATH {
		return false
	}
	if strings.TrimSpace(sku.LocalPath) == "" {
		return false
	}
	if sku.HostPaths == nil || sku.HostPaths.IsZero() {
		return false
	}
	return hostPathsHasContainerMount(*sku.HostPaths, 0)
}
