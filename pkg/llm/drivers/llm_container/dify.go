package llm_container

import (
	"context"
	"fmt"
	"strconv"

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

func (d *dify) GetPrimaryImageId(sku *models.SLLMSku) string {
	if spec := d.GetSpec(sku); spec != nil {
		s := spec.(*api.LLMSpecDify)
		if s.DifyApiImageId != "" {
			return s.DifyApiImageId
		}
	}
	return ""
}

func (d *dify) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	if input.LLMSpec == nil || input.LLMSpec.Dify == nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "dify SKU requires llm_spec with type dify and image ids")
	}
	if input.MountedModels != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "dify SKU does not support mounted models")
	}
	difySpec := input.LLMSpec.Dify
	for _, imgId := range []*string{&difySpec.PostgresImageId, &difySpec.RedisImageId, &difySpec.NginxImageId, &difySpec.DifyApiImageId, &difySpec.DifyPluginImageId, &difySpec.DifyWebImageId, &difySpec.DifySandboxImageId, &difySpec.DifySSRFImageId, &difySpec.DifyWeaviateImageId} {
		if *imgId == "" {
			continue
		}
		imgObj, err := validators.ValidateModel(ctx, userCred, models.GetLLMImageManager(), imgId)
		if err != nil {
			return nil, errors.Wrapf(err, "validate image_id %s", *imgId)
		}
		img := imgObj.(*models.SLLMImage)
		if img.LLMType != input.LLMType {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type %s", *imgId, input.LLMType)
		}
		*imgId = img.Id
	}
	input.LLMImageId = difySpec.DifyApiImageId
	return input, nil
}

func (d *dify) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	if input.MountedModels != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "dify SKU does not support mounted models")
	}
	if input.LLMSpec == nil || input.LLMSpec.Dify == nil {
		return nil, nil
	}
	currentSpec := d.GetSpec(sku)
	if currentSpec == nil {
		return nil, nil
	}
	updated := *currentSpec.(*api.LLMSpecDify)
	difySpec := input.LLMSpec.Dify
	mergeStr := func(dst *string, src string) {
		if src != "" {
			*dst = src
		}
	}
	mergeStr(&updated.PostgresImageId, difySpec.PostgresImageId)
	mergeStr(&updated.RedisImageId, difySpec.RedisImageId)
	mergeStr(&updated.NginxImageId, difySpec.NginxImageId)
	mergeStr(&updated.DifyApiImageId, difySpec.DifyApiImageId)
	mergeStr(&updated.DifyPluginImageId, difySpec.DifyPluginImageId)
	mergeStr(&updated.DifyWebImageId, difySpec.DifyWebImageId)
	mergeStr(&updated.DifySandboxImageId, difySpec.DifySandboxImageId)
	mergeStr(&updated.DifySSRFImageId, difySpec.DifySSRFImageId)
	mergeStr(&updated.DifyWeaviateImageId, difySpec.DifyWeaviateImageId)
	if len(difySpec.CustomizedEnvs) > 0 {
		updated.CustomizedEnvs = difySpec.CustomizedEnvs
	}
	for _, imgId := range []*string{&updated.PostgresImageId, &updated.RedisImageId, &updated.NginxImageId, &updated.DifyApiImageId, &updated.DifyPluginImageId, &updated.DifyWebImageId, &updated.DifySandboxImageId, &updated.DifySSRFImageId, &updated.DifyWeaviateImageId} {
		if *imgId != "" {
			imgObj, err := validators.ValidateModel(ctx, userCred, models.GetLLMImageManager(), imgId)
			if err != nil {
				return nil, errors.Wrapf(err, "validate image_id %s", *imgId)
			}
			img := imgObj.(*models.SLLMImage)
			if img.LLMType != sku.LLMType {
				return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "image %s is not of type %s", *imgId, sku.LLMType)
			}
			*imgId = img.GetId()
		}
	}
	// if dify_api_image_id is set, use it as the primary image id
	if input.LLMSpec.Dify.DifyApiImageId != "" {
		input.LLMImageId = input.LLMSpec.Dify.DifyApiImageId
	}
	return input, nil
}

// GetContainerSpec is required by ILLMContainerDriver but not used for Dify; pod creation uses GetContainerSpecs. Return the first container so the interface is satisfied.
func (d *dify) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	specs := d.GetContainerSpecs(ctx, llm, image, sku, props, devices, diskId)
	if len(specs) == 0 {
		return nil
	}
	return specs[0]
}

// GetContainerSpecs returns all Dify pod containers (postgres, redis, api, worker, nginx, etc.). SKU-only policy: customized envs come from llm_spec.dify.customized_envs.
func (d *dify) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return models.GetDifyContainersByNameAndSku(llm.GetName(), sku, nil)
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
