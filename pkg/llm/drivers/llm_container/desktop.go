package llm_container

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newDesktop())
}

type desktop struct {
	baseDriver
}

func newDesktop() models.ILLMContainerDriver {
	return &desktop{baseDriver: newBaseDriver(api.LLM_CONTAINER_DESKTOP)}
}

func (d *desktop) GetSpec(sku *models.SLLMSku) interface{} {
	return nil
}

func (d *desktop) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	return nil
}

func desktopUiTitle(cfg *api.LLMImageDesktopConfig) string {
	if cfg != nil && cfg.UiTitle != "" {
		return cfg.UiTitle
	}
	return "Cloudpods Desktop"
}

func appendDesktopExtraEnvs(envs []*commonapi.ContainerKeyValue, extra map[string]string) []*commonapi.ContainerKeyValue {
	if len(extra) == 0 {
		return envs
	}
	for k, v := range extra {
		envs = append(envs, &commonapi.ContainerKeyValue{Key: k, Value: v})
	}
	return envs
}

func (d *desktop) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	diskIndex := 0
	cfg, err := image.GetResolvedDesktopConfig()
	if err != nil {
		// Should not happen for valid desktop images; fall back to defaults.
		cfg, _ = api.ResolveDesktopConfig(image.ImageName, image.ImageLabel, nil)
	}

	desktopInner := desktopWebtopImageBaseContainerSpec(image)
	desktopInner.Envs = desktopWebtopCommonEnvs(llm.GetId(), desktopUiTitle(cfg))
	if cfg != nil {
		desktopInner.Envs = appendDesktopExtraEnvs(desktopInner.Envs, cfg.ExtraEnvs)
	}
	if desktopHasIsolatedGPU(llm, sku, devices) {
		desktopInner.Envs = append(desktopInner.Envs, desktopGPUWaylandEnvs()...)
	}
	desktopSpec := computeapi.ContainerSpec{
		ContainerSpec: desktopInner,
		VolumeMounts:  desktopStandardVolumeMounts(&diskIndex, false, ""),
		RootFs:        desktopContainerRootFs(&diskIndex),
	}
	appendContainerIsolatedDevices(&desktopSpec, llm, sku, devices)
	return []*computeapi.PodContainerCreateInput{
		{
			Name:          fmt.Sprintf("%s-%d", llm.GetName(), 0),
			ContainerSpec: desktopSpec,
		},
	}
}

func (d *desktop) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	protocol := "https"
	port := api.LLM_DESKTOP_DEFAULT_PORT
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMSku")
	}
	if sku != nil && sku.LLMImageId != "" {
		image, err := models.GetLLMImageManager().FetchById(sku.LLMImageId)
		if err != nil {
			return nil, errors.Wrap(err, "FetchLLMImage")
		}
		if img, ok := image.(*models.SLLMImage); ok {
			cfg, err := img.GetResolvedDesktopConfig()
			if err != nil {
				return nil, errors.Wrap(err, "GetResolvedDesktopConfig")
			}
			if cfg != nil {
				if cfg.Protocol != "" {
					protocol = cfg.Protocol
				}
				if cfg.DefaultPort > 0 {
					port = cfg.DefaultPort
				}
			}
		}
	}
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, protocol, port)
}

func (d *desktop) GetLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (*api.LLMAccessInfo, error) {
	return getDesktopWebUILoginInfo(ctx, llm)
}
