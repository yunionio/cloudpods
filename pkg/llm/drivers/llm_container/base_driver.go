package llm_container

import (
	"context"

	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type baseDriver struct {
	drvType api.LLMContainerType
}

func newBaseDriver(drvType api.LLMContainerType) baseDriver {
	return baseDriver{drvType: drvType}
}

func (b *baseDriver) GetType() api.LLMContainerType {
	return b.drvType
}

func (b *baseDriver) GetPrimaryImageId(sku *models.SLLMSku) string {
	return sku.LLMImageId
}

func (b *baseDriver) GetPrimaryContainer(ctx context.Context, llm *models.SLLM, containers []*computeapi.PodContainerDesc) (*computeapi.PodContainerDesc, error) {
	return containers[0], nil
}

func (b *baseDriver) GetMountedModels(sku *models.SLLMSku) []string {
	return sku.MountedModels
}

func (b *baseDriver) StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	return nil
}

func (b *baseDriver) ValidateLLMSkuCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	imgObj, err := validators.ValidateModel(ctx, userCred, models.GetLLMImageManager(), &input.LLMImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "validate image_id %s", input.LLMImageId)
	}
	llmImage := imgObj.(*models.SLLMImage)
	if llmImage.LLMType != input.LLMType {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type %s", input.LLMImageId, input.LLMType)
	}
	input.LLMImageId = llmImage.Id
	if input.MountedModels != nil {
		for i, mdl := range input.MountedModels {
			instMdl, err := models.GetInstantModelManager().FetchByIdOrName(ctx, userCred, mdl)
			if err != nil {
				return nil, errors.Wrapf(err, "validate mounted model %s", mdl)
			}
			instantModle := instMdl.(*models.SInstantModel)
			if !api.IsLLMInstantModelCompatible(api.LLMContainerType(instantModle.LlmType), api.LLMContainerType(input.LLMType)) {
				return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "mounted model %s is not of type %s", mdl, input.LLMType)
			}
			input.MountedModels[i] = instantModle.GetId()
		}
	}
	return input, nil
}

func (b *baseDriver) ValidateLLMSkuUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	llmImageId := input.LLMImageId
	if llmImageId != "" {
		imgObj, err := validators.ValidateModel(ctx, userCred, models.GetLLMImageManager(), &llmImageId)
		if err != nil {
			return nil, errors.Wrapf(err, "validate image_id %s", llmImageId)
		}
		llmImage := imgObj.(*models.SLLMImage)
		if llmImage.LLMType != sku.LLMType {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type %s", llmImageId, sku.LLMType)
		}
		input.LLMImageId = llmImage.Id
	}

	mountedModels := input.MountedModels
	if input.MountedModels != nil {
		mountedModels = make([]string, len(input.MountedModels))
		for i, mdl := range input.MountedModels {
			instMdl, err := models.GetInstantModelManager().FetchByIdOrName(ctx, userCred, mdl)
			if err != nil {
				return nil, errors.Wrapf(err, "validate mounted model %s", mdl)
			}
			instantModle := instMdl.(*models.SInstantModel)
			if !api.IsLLMInstantModelCompatible(api.LLMContainerType(instantModle.LlmType), api.LLMContainerType(sku.LLMType)) {
				return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "mounted model %s is not of type %s", mdl, sku.LLMType)
			}
			mountedModels[i] = instantModle.GetId()
		}
	}
	input.MountedModels = mountedModels
	return input, nil
}

func MatchContainerToUpdateByName(ctr *computeapi.SContainer, podCtrs []*computeapi.PodContainerCreateInput) (*computeapi.PodContainerCreateInput, error) {
	ctrName := ctr.Name
	for _, podCtr := range podCtrs {
		if podCtr.Name == ctrName {
			return podCtr, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "container %s not found", ctrName)
}

func (b *baseDriver) MatchContainerToUpdate(ctr *computeapi.SContainer, podCtrs []*computeapi.PodContainerCreateInput) (*computeapi.PodContainerCreateInput, error) {
	if len(podCtrs) == 1 {
		return podCtrs[0], nil
	}
	return MatchContainerToUpdateByName(ctr, podCtrs)
}

func (b *baseDriver) ValidateLLMCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSpec) (*api.LLMSpec, error) {
	return input, nil
}

func (b *baseDriver) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	return input, nil
}
