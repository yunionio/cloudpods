package models

import (
	"context"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func resolveLLMSkuCatalogImport(input *api.LLMSkuCreateInput) (*api.InstantModelImportInput, error) {
	if input == nil || strings.TrimSpace(input.LLMModelSpecId) == "" {
		return nil, nil
	}
	spec, setName, ok := GetLLMModelSetManager().GetSpec(input.LLMModelSpecId)
	if !ok {
		return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "llm_model_spec %s not found", input.LLMModelSpecId)
	}
	set, ok := GetLLMModelSetManager().GetSet(setName)
	if !ok {
		return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "llm_model_set %s not found", setName)
	}
	return buildLLMSkuCatalogImport(input, set, spec)
}

func catalogBackendToLLMType(backend string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "":
		return "", true
	case "vllm":
		return string(api.LLM_CONTAINER_VLLM), true
	case "sglang":
		return string(api.LLM_CONTAINER_SGLANG), true
	case "ollama":
		return string(api.LLM_CONTAINER_OLLAMA), true
	default:
		return "", false
	}
}

func buildLLMSkuCatalogImport(input *api.LLMSkuCreateInput, set *api.LLMModelSet, spec *api.LLMModelSpec) (*api.InstantModelImportInput, error) {
	if input == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "empty sku input")
	}
	if spec == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "empty catalog spec")
	}
	if !strings.EqualFold(spec.Source, api.LLM_MODEL_SOURCE_HUGGINGFACE) {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "unsupported catalog source %q", spec.Source)
	}
	if strings.TrimSpace(spec.HuggingfaceRepoId) == "" {
		return nil, errors.Wrap(httperrors.ErrMissingParameter, "huggingface_repo_id is required")
	}
	if expectedType, ok := catalogBackendToLLMType(spec.Backend); !ok {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "unsupported catalog backend %q", spec.Backend)
	} else if expectedType != "" && input.LLMType != "" && expectedType != input.LLMType {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "catalog backend %q requires llm_type %q", spec.Backend, expectedType)
	}

	input.LLMModelSpecId = spec.SpecId
	input.Source = spec.Source
	input.HuggingfaceRepoId = spec.HuggingfaceRepoId
	input.HuggingfaceFilename = spec.HuggingfaceFilename
	input.ModelScopeModelId = spec.ModelScopeModelId
	input.ModelScopeFilePath = spec.ModelScopeFilePath
	input.LocalPath = spec.LocalPath
	input.BackendVersion = spec.BackendVersion
	input.BackendParameters = append([]string{}, spec.BackendParameters...)
	if set != nil {
		input.Categories = append([]string{}, set.Categories...)
	}

	revision := defaultHuggingFaceRevision
	return &api.InstantModelImportInput{
		ModelName: spec.HuggingfaceRepoId,
		ModelTag:  revision,
		LlmType:   api.LLMContainerType(input.LLMType),
		Source:    api.InstantModelSourceHuggingFace,
		RepoId:    spec.HuggingfaceRepoId,
		Revision:  revision,
	}, nil
}

func appendMountedModelIds(existing []string, ids ...string) []string {
	out := append([]string{}, existing...)
	seen := make(map[string]struct{}, len(out)+len(ids))
	for _, id := range out {
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, id)
		seen[id] = struct{}{}
	}
	return out
}

func EnableInstantModelForUse(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	obj, err := GetInstantModelManager().FetchById(id)
	if err != nil {
		return errors.Wrapf(err, "fetch InstantModel %s", id)
	}
	im := obj.(*SInstantModel)
	if im.Enabled.IsTrue() {
		return nil
	}
	_, err = db.Update(im, func() error {
		im.SetEnabled(true)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	return nil
}

func (sku *SLLMSku) AttachMountedModel(ctx context.Context, userCred mcclient.TokenCredential, instantModelId string) error {
	_, err := db.Update(sku, func() error {
		sku.MountedModels = appendMountedModelIds(sku.MountedModels, instantModelId)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	return nil
}
