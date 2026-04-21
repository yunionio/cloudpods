package llm_container

import (
	"context"
	"fmt"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	hermesAgentDataDir = "/opt/data"
)

func init() {
	models.RegisterLLMContainerDriver(newHermesAgent())
}

type hermesAgent struct {
	baseDriver
}

func newHermesAgent() models.ILLMContainerDriver {
	return &hermesAgent{
		baseDriver: newBaseDriver(api.LLM_CONTAINER_HERMES_AGENT),
	}
}

func (h *hermesAgent) GetSpec(sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMSpec == nil {
		return nil
	}
	return sku.LLMSpec.HermesAgent
}

func (h *hermesAgent) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.HermesAgent != nil {
		return llm.LLMSpec.HermesAgent
	}
	return h.GetSpec(sku)
}

func (h *hermesAgent) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	diskIndex := 0

	hermesVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "home",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: homeDir,
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "data",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: hermesAgentDataDir,
		},
	}
	hermesInner := desktopWebtopImageBaseContainerSpec(image)
	hermesInner.Envs = append(desktopWebtopCommonEnvs(llm.GetId()),
		models.NewEnv("HERMES_HOME", hermesAgentDataDir),
		&commonapi.ContainerKeyValue{Key: "HERMES_WEB_DIST", Value: "/opt/hermes/hermes_cli/web_dist"},
	)
	hermesSpec := computeapi.ContainerSpec{
		ContainerSpec: hermesInner,
		VolumeMounts:  hermesVols,
		RootFs:        desktopContainerRootFs(&diskIndex),
	}
	return &computeapi.PodContainerCreateInput{
		Name:          fmt.Sprintf("%s-%d", llm.GetName(), 0),
		ContainerSpec: hermesSpec,
	}
}

func (h *hermesAgent) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return []*computeapi.PodContainerCreateInput{
		h.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId),
	}
}

func (h *hermesAgent) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "https", api.LLM_DESKTOP_DEFAULT_PORT)
}

// GetLoginInfo returns desktop web UI login credentials (same defaults as container env).
func (h *hermesAgent) GetLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (*api.LLMAccessInfo, error) {
	return getDesktopWebUILoginInfo(ctx, llm)
}
