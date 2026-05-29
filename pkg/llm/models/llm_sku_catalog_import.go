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

func resolveLLMSkuImport(input *api.LLMSkuCreateInput) (*api.InstantModelImportInput, error) {
	if input == nil || input.ModelSpec == nil {
		return nil, nil
	}
	return buildLLMSkuImportFromModelSpec(input)
}

func buildLLMSkuImportFromModelSpec(input *api.LLMSkuCreateInput) (*api.InstantModelImportInput, error) {
	if input == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "empty sku input")
	}
	if input.ModelSpec == nil {
		return nil, nil
	}
	importInput := *input.ModelSpec
	if strings.TrimSpace(importInput.ModelName) == "" {
		return nil, errors.Wrap(httperrors.ErrMissingParameter, "model_spec.model_name is required")
	}
	if strings.TrimSpace(importInput.ModelTag) == "" {
		return nil, errors.Wrap(httperrors.ErrMissingParameter, "model_spec.model_tag is required")
	}
	if importInput.LlmType == "" {
		importInput.LlmType = api.LLMContainerType(input.LLMType)
	}
	if input.LLMType != "" && importInput.LlmType != "" && string(importInput.LlmType) != input.LLMType {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "model_spec.llm_type %q does not match llm_type %q", importInput.LlmType, input.LLMType)
	}

	source := strings.ToLower(strings.TrimSpace(importInput.Source))
	repoID := strings.TrimSpace(importInput.RepoId)
	if source == "" {
		if repoID == "" {
			repoID = strings.TrimSpace(importInput.ModelName)
		}
		source = normalizeInstantModelSource(source, repoID)
	}
	switch source {
	case api.InstantModelSourceHuggingFace:
		if repoID == "" {
			repoID = strings.TrimSpace(importInput.ModelName)
		}
		revision := strings.TrimSpace(importInput.Revision)
		if revision == "" {
			revision = strings.TrimSpace(importInput.ModelTag)
		}
		importInput.Source = api.InstantModelSourceHuggingFace
		importInput.RepoId = repoID
		importInput.Revision = revision
		importInput.ModelName = repoID
		importInput.ModelTag = revision
		input.Source = api.LLM_MODEL_SOURCE_HUGGINGFACE
		input.HuggingfaceRepoId = repoID
	case "ollama":
		importInput.Source = source
		input.Source = source
	default:
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "unsupported model_spec source %q", importInput.Source)
	}

	input.ModelSpec = &importInput
	return &importInput, nil
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
