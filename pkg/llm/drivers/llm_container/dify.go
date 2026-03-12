package llm_container

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newDify())
}

type dify struct{}

func newDify() models.ILLMContainerDriver {
	return new(dify)
}

func (d *dify) GetType() api.LLMContainerType {
	return api.LLM_CONTAINER_DIFY
}

func (d *dify) GetSpec(sku *models.SLLMSku) interface{} {
	if sku.LLMSpec == nil {
		return nil
	}
	return sku.LLMSpec.Dify
}

// mergeDifySpecInto merges src into dst. If fillEmpty is true, only fills dst when dst is empty; otherwise overwrites dst when src is non-empty.
func mergeDifySpecInto(dst, src *api.LLMSpecDify, fillEmpty bool) {
	if dst == nil || src == nil {
		return
	}
	mergeStr := func(dstPtr *string, srcVal string) {
		if fillEmpty {
			if *dstPtr == "" && srcVal != "" {
				*dstPtr = srcVal
			}
		} else {
			if srcVal != "" {
				*dstPtr = srcVal
			}
		}
	}
	mergeStr(&dst.PostgresImageId, src.PostgresImageId)
	mergeStr(&dst.RedisImageId, src.RedisImageId)
	mergeStr(&dst.NginxImageId, src.NginxImageId)
	mergeStr(&dst.DifyApiImageId, src.DifyApiImageId)
	mergeStr(&dst.DifyPluginImageId, src.DifyPluginImageId)
	mergeStr(&dst.DifyWebImageId, src.DifyWebImageId)
	mergeStr(&dst.DifySandboxImageId, src.DifySandboxImageId)
	mergeStr(&dst.DifySSRFImageId, src.DifySSRFImageId)
	mergeStr(&dst.DifyWeaviateImageId, src.DifyWeaviateImageId)
	if (fillEmpty && len(dst.CustomizedEnvs) == 0 && len(src.CustomizedEnvs) > 0) || (!fillEmpty && len(src.CustomizedEnvs) > 0) {
		dst.CustomizedEnvs = src.CustomizedEnvs
	}
}

// mergeDify merges llm and sku Dify specs; llm takes priority, use sku when llm is nil or zero.
func mergeDify(llm, sku *api.LLMSpecDify) *api.LLMSpecDify {
	if llm != nil && !llm.IsZero() {
		out := *llm
		if llm.CustomizedEnvs != nil {
			out.CustomizedEnvs = make([]*api.DifyCustomizedEnv, len(llm.CustomizedEnvs))
			copy(out.CustomizedEnvs, llm.CustomizedEnvs)
		}
		return &out
	}
	if sku != nil {
		out := *sku
		if sku.CustomizedEnvs != nil {
			out.CustomizedEnvs = make([]*api.DifyCustomizedEnv, len(sku.CustomizedEnvs))
			copy(out.CustomizedEnvs, sku.CustomizedEnvs)
		}
		return &out
	}
	return nil
}

func (d *dify) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMSpec == nil {
		return nil
	}
	var llmDify *api.LLMSpecDify
	if llm != nil && llm.LLMSpec != nil {
		llmDify = llm.LLMSpec.Dify
	}
	return mergeDify(llmDify, sku.LLMSpec.Dify)
}

func (d *dify) GetPrimaryImageId(sku *models.SLLMSku) string {
	if spec := d.GetSpec(sku); spec != nil {
		s := spec.(*api.LLMSpecDify)
		if s.DifyApiImageId != "" {
			return s.DifyApiImageId
		}
	}
	return ""
}

func (d *dify) GetPrimaryContainer(ctx context.Context, llm *models.SLLM, containers []*computeapi.PodContainerDesc) (*computeapi.PodContainerDesc, error) {
	for _, ctr := range containers {
		if strings.HasSuffix(ctr.Name, api.DIFY_API_KEY) {
			return ctr, nil
		}
	}
	return nil, errors.Error("api container not found")
}

func (d *dify) ValidateLLMSkuCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	if input.LLMSpec == nil || input.LLMSpec.Dify == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "dify SKU requires llm_spec with type dify and image ids")
	}
	if input.MountedModels != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "dify SKU does not support mounted models")
	}

	// Reuse ValidateLLMCreateSpec to normalize/validate LLMSpec.
	spec, err := d.ValidateLLMCreateSpec(ctx, userCred, nil, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	input.LLMSpec = spec
	if input.LLMSpec != nil && input.LLMSpec.Dify != nil {
		input.LLMImageId = input.LLMSpec.Dify.DifyApiImageId
	}
	return input, nil
}

func (d *dify) ValidateLLMSkuUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	if input.MountedModels != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "dify SKU does not support mounted models")
	}
	if input.LLMSpec == nil {
		return input, nil
	}

	// Reuse ValidateLLMUpdateSpec by treating current SKU spec as the \"current llm spec\".
	fakeLLM := &models.SLLM{LLMSpec: sku.LLMSpec}
	spec, err := d.ValidateLLMUpdateSpec(ctx, userCred, fakeLLM, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	input.LLMSpec = spec

	// if dify_api_image_id is set, use it as the primary image id
	if input.LLMSpec != nil && input.LLMSpec.Dify != nil && input.LLMSpec.Dify.DifyApiImageId != "" {
		input.LLMImageId = input.LLMSpec.Dify.DifyApiImageId
	}
	return input, nil
}

// ValidateLLMCreateSpec implements ILLMContainerDriver. Validates image ids and merges empty fields from SKU spec.
func (d *dify) ValidateLLMCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.Dify == nil {
		return input, nil
	}
	difySpec := input.Dify
	// Merge empty fields from SKU so the stored spec is complete
	if sku != nil && sku.LLMSpec != nil && sku.LLMSpec.Dify != nil {
		mergeDifySpecInto(difySpec, sku.LLMSpec.Dify, true)
	}
	// Validate non-empty image ids
	for _, imgId := range []*string{&difySpec.PostgresImageId, &difySpec.RedisImageId, &difySpec.NginxImageId, &difySpec.DifyApiImageId, &difySpec.DifyPluginImageId, &difySpec.DifyWebImageId, &difySpec.DifySandboxImageId, &difySpec.DifySSRFImageId, &difySpec.DifyWeaviateImageId} {
		if *imgId == "" {
			continue
		}
		imgObj, err := validators.ValidateModel(ctx, userCred, models.GetLLMImageManager(), imgId)
		if err != nil {
			return nil, errors.Wrapf(err, "validate image_id %s", *imgId)
		}
		img := imgObj.(*models.SLLMImage)
		if img.LLMType != string(api.LLM_CONTAINER_DIFY) {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type dify", *imgId)
		}
		*imgId = img.Id
	}
	return &api.LLMSpec{Dify: difySpec}, nil
}

// ValidateLLMUpdateSpec implements ILLMContainerDriver. Merges input with current LLM spec (non-empty overwrites); validates image ids.
func (d *dify) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.Dify == nil {
		return input, nil
	}
	// Start from current LLM spec, or SKU spec as base
	var base *api.LLMSpecDify
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.Dify != nil {
		b := *llm.LLMSpec.Dify
		base = &b
		if llm.LLMSpec.Dify.CustomizedEnvs != nil {
			base.CustomizedEnvs = make([]*api.DifyCustomizedEnv, len(llm.LLMSpec.Dify.CustomizedEnvs))
			copy(base.CustomizedEnvs, llm.LLMSpec.Dify.CustomizedEnvs)
		}
	} else if llm != nil {
		sku, err := llm.GetLLMSku(llm.LLMSkuId)
		if err == nil && sku != nil && sku.LLMSpec != nil && sku.LLMSpec.Dify != nil {
			b := *sku.LLMSpec.Dify
			base = &b
			if sku.LLMSpec.Dify.CustomizedEnvs != nil {
				base.CustomizedEnvs = make([]*api.DifyCustomizedEnv, len(sku.LLMSpec.Dify.CustomizedEnvs))
				copy(base.CustomizedEnvs, sku.LLMSpec.Dify.CustomizedEnvs)
			}
		}
	}
	if base == nil {
		base = &api.LLMSpecDify{}
	}
	mergeDifySpecInto(base, input.Dify, false)
	// Validate non-empty image ids
	for _, imgId := range []*string{&base.PostgresImageId, &base.RedisImageId, &base.NginxImageId, &base.DifyApiImageId, &base.DifyPluginImageId, &base.DifyWebImageId, &base.DifySandboxImageId, &base.DifySSRFImageId, &base.DifyWeaviateImageId} {
		if *imgId == "" {
			continue
		}
		imgObj, err := validators.ValidateModel(ctx, userCred, models.GetLLMImageManager(), imgId)
		if err != nil {
			return nil, errors.Wrapf(err, "validate image_id %s", *imgId)
		}
		img := imgObj.(*models.SLLMImage)
		if img.LLMType != string(api.LLM_CONTAINER_DIFY) {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type dify", *imgId)
		}
		*imgId = img.Id
	}
	return &api.LLMSpec{Dify: base}, nil
}

// GetContainerSpec is required by ILLMContainerDriver but not used for Dify; pod creation uses GetContainerSpecs. Return the first container so the interface is satisfied.
func (d *dify) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	specs := d.GetContainerSpecs(ctx, llm, image, sku, props, devices, diskId)
	if len(specs) == 0 {
		return nil
	}
	return specs[0]
}

// GetContainerSpecs returns all Dify pod containers (postgres, redis, api, worker, nginx, etc.). Uses effective spec (llm + sku merged by driver).
func (d *dify) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	spec := d.GetEffectiveSpec(llm, sku)
	if spec == nil {
		return nil
	}
	return models.GetDifyContainersByNameAndSku(llm.GetName(), sku, nil, spec.(*api.LLMSpecDify))
}

func (d *dify) MatchContainerToUpdate(ctr *computeapi.SContainer, podCtrs []*computeapi.PodContainerCreateInput) (*computeapi.PodContainerCreateInput, error) {
	return MatchContainerToUpdateByName(ctr, podCtrs)
}

// StartLLM is a no-op for Dify; all services are started by their container entrypoints.
func (d *dify) StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	return nil
}

// GetLLMUrl returns the Dify access URL (nginx port 80). Same pattern as vLLM/Ollama: guest network uses LLMIp, hostlocal uses host IP.
func (d *dify) GetLLMUrl(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (string, error) {
	server, err := llm.GetServer(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get server")
	}
	port := 80
	if p, err := strconv.Atoi(api.DIFY_NGINX_PORT); err == nil {
		port = p
	}
	networkType := llm.NetworkType
	if networkType == string(computeapi.NETWORK_TYPE_GUEST) {
		if len(llm.LLMIp) == 0 {
			return "", errors.Error("LLM IP is empty for guest network")
		}
		return fmt.Sprintf("http://%s:%d", llm.LLMIp, port), nil
	}
	if len(server.HostAccessIp) == 0 {
		return "", errors.Error("host access IP is empty")
	}
	return fmt.Sprintf("http://%s:%d", server.HostAccessIp, port), nil
}
