package llm_container

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

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
	httpAuthUsername := "admin"
	httpAuthPassword := openclawFixed9DigitPassword(llm.GetId())
	hermesSpec := computeapi.ContainerSpec{
		ContainerSpec: commonapi.ContainerSpec{
			Image:             image.ToContainerImage(),
			ImageCredentialId: image.CredentialId,
			EnableLxcfs:       true,
			AlwaysRestart:     true,
			ShmSizeMB:         2048,
			DisableNoNewPrivs: true,
			Envs: []*commonapi.ContainerKeyValue{
				// Desktop env
				// {Key: "TZ", Value: "Etc/UTC"},
				{Key: "TZ", Value: "Asia/Shanghai"},
				{Key: "PUID", Value: "1000"},
				{Key: "PGID", Value: "1000"},
				{Key: "LC_ALL", Value: "zh_CN.UTF-8"},

				// webtop envs: https://github.com/linuxserver/docker-webtop?tab=readme-ov-file#advanced-configuration
				// {Key: "DISABLE_SUDO", Value: "true"},

				// Provider
				// {Key: "MOONSHOT_API_KEY", Value: "abc"},
				// {Key: "OPENCLAW_PRIMARY_MODEL", Value: "moonshot/kimi-k2.5"},
				// Auth
				{Key: string(api.LLM_OPENCLAW_AUTH_USERNAME), Value: httpAuthUsername},
				{Key: string(api.LLM_OPENCLAW_CUSTOM_USER), Value: httpAuthUsername},
				{Key: string(api.LLM_OPENCLAW_AUTH_PASSWORD), Value: httpAuthPassword},
				{Key: string(api.LLM_OPENCLAW_PASSWORD), Value: httpAuthPassword},
				// Brew env
				{Key: "HOMEBREW_PREFIX", Value: "/home/linuxbrew/.linuxbrew"},
				{Key: "HOMEBREW_CELLAR", Value: "/home/linuxbrew/.linuxbrew/Cellar"},
				{Key: "HOMEBREW_REPOSITORY", Value: "/home/linuxbrew/.linuxbrew/Homebrew"},
				// Selkies env
				{Key: "SELKIES_UI_TITLE", Value: "Cloudpods Desktop"},
				{Key: "SELKIES_UI_SHOW_LOGO", Value: "False"},
				{Key: "SELKIES_UI_SIDEBAR_SHOW_APPS", Value: "False"},
				{Key: "SELKIES_UI_SIDEBAR_SHOW_GAMEPADS", Value: "False"},
				// Hermes env
				models.NewEnv("HERMES_HOME", hermesAgentDataDir),
				{Key: "HERMES_WEB_DIST", Value: "/opt/hermes/hermes_cli/web_dist"},
			},
		},
		VolumeMounts: hermesVols,
		RootFs: &commonapi.ContainerRootfs{
			Type: commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "rootfs",
			},
			Persistent: false,
		},
	}
	// inject credential envs
	// spec := c.GetEffectiveSpec(llm, sku)
	if llm.LLMSpec == nil || llm.LLMSpec.HermesAgent == nil {
		return &computeapi.PodContainerCreateInput{
			Name:          fmt.Sprintf("%s-%d", llm.GetName(), 0),
			ContainerSpec: hermesSpec,
		}
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
	//TODO implement me
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "https", api.LLM_OPENCLAW_DEFAULT_PORT)
}

// GetLoginInfo returns OpenClaw web UI login credentials (same defaults as container env).
func (h *hermesAgent) GetLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (*api.LLMAccessInfo, error) {
	ctr, err := llm.GetLLMSContainer(ctx)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get llm cloud container")
	}
	if ctr.Spec == nil {
		return nil, errors.Wrap(errors.ErrEmpty, "no Spec")
	}
	var (
		username     string
		password     string
		gatewayToken string
	)
	for _, env := range ctr.Spec.Envs {
		if env.Key == string(api.LLM_OPENCLAW_AUTH_USERNAME) {
			username = env.Value
		}
		if env.Key == string(api.LLM_OPENCLAW_AUTH_PASSWORD) {
			password = env.Value
		}
		if env.Key == string(api.LLM_OPENCLAW_GATEWAY_TOKEN) {
			gatewayToken = env.Value
		}
	}
	return &api.LLMAccessInfo{
		Username: username,
		Password: password,
		Extra: map[string]string{
			string(api.LLM_OPENCLAW_GATEWAY_TOKEN): gatewayToken,
		},
	}, nil
}
