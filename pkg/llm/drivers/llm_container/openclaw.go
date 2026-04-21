package llm_container

import (
	"context"
	"encoding/base64"
	"fmt"
	"hash/fnv"

	"yunion.io/x/jsonutils"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// coollabsio/openclaw docker-compose: openclaw (main) + browser (CDP sidecar for /browser/)
// See: https://github.com/coollabsio/openclaw/blob/main/docker-compose.yml
const (
	// openclawContainerName  = "openclaw"
	// browserContainerName   = "browser"
	// openclawBrowserImage   = "registry.cn-beijing.aliyuncs.com/cloudpods/openclaw-browser:latest"
	openclawDataDir  = "/data"
	desktopConfigDir = "/config"
	homeDir          = "/home/"
	// openclawBrowserCDPPort = "9222"
)

func openclawFixed9DigitPassword(llmId string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(llmId))
	x := h.Sum64()

	// 固定 9 位，字母数字混合，并保证至少包含 1 个字母
	const alpha = "abcdefghijklmnopqrstuvwxyz"
	const base36 = "0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]byte, 9)
	out[0] = alpha[x%26]
	for i := 1; i < len(out); i++ {
		// xorshift64* 生成稳定伪随机序列
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		x *= 2685821657736338717
		out[i] = base36[x%36]
	}
	return string(out)
}

func appendCredentialEnvs(envs []*commonapi.ContainerKeyValue, cred *api.LLMSpecCredential) []*commonapi.ContainerKeyValue {
	if cred == nil {
		return envs
	}
	for _, key := range cred.ExportKeys {
		envs = append(envs, &commonapi.ContainerKeyValue{
			Key: key,
			ValueFrom: &commonapi.ContainerValueSource{
				Credential: &commonapi.ContainerValueSourceCredential{
					Id:  cred.Id,
					Key: key,
				},
			},
		})
	}
	return envs
}

func init() {
	models.RegisterLLMContainerDriver(newOpenClaw())
}

type openclaw struct {
	baseDriver
}

func newOpenClaw() models.ILLMContainerDriver {
	return &openclaw{baseDriver: newBaseDriver(api.LLM_CONTAINER_OPENCLAW)}
}

func (c *openclaw) GetSpec(sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMSpec == nil {
		return nil
	}
	return sku.LLMSpec.OpenClaw
}

// mergeOpenClaw merges llm and sku OpenClaw specs; llm takes priority, use sku when llm field is unset (nil or empty).
func mergeOpenClaw(llm, sku *api.LLMSpecOpenClaw) *api.LLMSpecOpenClaw {
	if llm == nil {
		if sku == nil {
			return nil
		}
		return copyOpenClaw(sku)
	}
	if sku == nil {
		return copyOpenClaw(llm)
	}
	out := &api.LLMSpecOpenClaw{}
	if len(llm.Providers) > 0 {
		out.Providers = make([]*api.LLMSpecOpenClawProvider, len(llm.Providers))
		copy(out.Providers, llm.Providers)
	} else if len(sku.Providers) > 0 {
		out.Providers = make([]*api.LLMSpecOpenClawProvider, len(sku.Providers))
		copy(out.Providers, sku.Providers)
	}
	if len(llm.Channels) > 0 {
		out.Channels = make([]*api.LLMSpecOpenClawChannel, len(llm.Channels))
		copy(out.Channels, llm.Channels)
	} else if len(sku.Channels) > 0 {
		out.Channels = make([]*api.LLMSpecOpenClawChannel, len(sku.Channels))
		copy(out.Channels, sku.Channels)
	}
	if llm.WorkspaceTemplates != nil && (llm.WorkspaceTemplates.AgentsMD != "" || llm.WorkspaceTemplates.SoulMD != "" || llm.WorkspaceTemplates.UserMD != "") {
		out.WorkspaceTemplates = &api.LLMSpecOpenClawWorkspaceTemplates{
			AgentsMD: llm.WorkspaceTemplates.AgentsMD,
			SoulMD:   llm.WorkspaceTemplates.SoulMD,
			UserMD:   llm.WorkspaceTemplates.UserMD,
		}
	} else if sku.WorkspaceTemplates != nil {
		out.WorkspaceTemplates = &api.LLMSpecOpenClawWorkspaceTemplates{
			AgentsMD: sku.WorkspaceTemplates.AgentsMD,
			SoulMD:   sku.WorkspaceTemplates.SoulMD,
			UserMD:   sku.WorkspaceTemplates.UserMD,
		}
	}
	return out
}

func copyOpenClaw(s *api.LLMSpecOpenClaw) *api.LLMSpecOpenClaw {
	if s == nil {
		return nil
	}
	out := &api.LLMSpecOpenClaw{}
	if len(s.Providers) > 0 {
		out.Providers = make([]*api.LLMSpecOpenClawProvider, len(s.Providers))
		copy(out.Providers, s.Providers)
	}
	if len(s.Channels) > 0 {
		out.Channels = make([]*api.LLMSpecOpenClawChannel, len(s.Channels))
		copy(out.Channels, s.Channels)
	}
	if s.WorkspaceTemplates != nil {
		out.WorkspaceTemplates = &api.LLMSpecOpenClawWorkspaceTemplates{
			AgentsMD: s.WorkspaceTemplates.AgentsMD,
			SoulMD:   s.WorkspaceTemplates.SoulMD,
			UserMD:   s.WorkspaceTemplates.UserMD,
		}
	}
	return out
}

func (c *openclaw) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMSpec == nil {
		return llm.LLMSpec.OpenClaw
	}
	var llmOC *api.LLMSpecOpenClaw
	if llm != nil && llm.LLMSpec != nil {
		llmOC = llm.LLMSpec.OpenClaw
	}
	return mergeOpenClaw(llmOC, sku.LLMSpec.OpenClaw)
}

func (c *openclaw) StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	// lc, err := llm.GetLLMContainer()
	// if err != nil {
	// 	return errors.Wrap(err, "get llm container")
	// }
	// // 启动 openclaw gateway
	// cmd := fmt.Sprintf("/app/scripts/entrypoint-gui.sh")
	// _, err = exec(ctx, lc.CmpId, cmd, 30)
	// if err != nil {
	// 	return errors.Wrap(err, "exec start openclaw gateway")
	// }
	return nil
}

func (c *openclaw) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	// Multi-container: use GetContainerSpecs
	return nil
}

// func (c *openclaw) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
// 	diskIndex := 0

// 	// 1. Browser sidecar: CDP on 9222, persistent /config, shm 2g
// 	browserVols := []*commonapi.ContainerVolumeMount{
// 		{
// 			Disk: &commonapi.ContainerVolumeMountDisk{
// 				Index:        &diskIndex,
// 				SubDirectory: browserStorageDir,
// 			},
// 			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
// 			MountPath: browserConfigDir,
// 		},
// 	}
// 	browserSpec := computeapi.ContainerSpec{
// 		ContainerSpec: commonapi.ContainerSpec{
// 			Image:         openclawBrowserImage,
// 			EnableLxcfs:   true,
// 			AlwaysRestart: true,
// 			ShmSizeMB:     2048, // 2g for Chrome
// 			Envs: []*commonapi.ContainerKeyValue{
// 				{Key: "PUID", Value: "1000"},
// 				{Key: "PGID", Value: "1000"},
// 				{Key: "TZ", Value: "Etc/UTC"},
// 				{Key: "CHROME_CLI", Value: "--remote-debugging-port=" + openclawBrowserCDPPort},
// 			},
// 		},
// 		VolumeMounts: browserVols,
// 	}

// 	// 2. OpenClaw main: nginx :8080 -> gateway :18789, /data, depends on browser
// 	openclawVols := []*commonapi.ContainerVolumeMount{
// 		{
// 			Disk: &commonapi.ContainerVolumeMountDisk{
// 				Index:        &diskIndex,
// 				SubDirectory: "data",
// 			},
// 			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
// 			MountPath: openclawDataDir,
// 		},
// 	}
// 	openclawSpec := computeapi.ContainerSpec{
// 		ContainerSpec: commonapi.ContainerSpec{
// 			Image:             image.ToContainerImage(),
// 			ImageCredentialId: image.CredentialId,
// 			EnableLxcfs:       true,
// 			AlwaysRestart:     true,
// 			DependsOn:         []string{fmt.Sprintf("%s-%s", llm.GetName(), browserContainerName)},
// 			Envs: []*commonapi.ContainerKeyValue{
// 				// Provider
// 				{Key: "MOONSHOT_API_KEY", Value: "sk-9taa32DcGGQliadQTEcZfpMUL9LCAnZVfyE6hKWPUMWEofJ8"},
// 				{Key: "OPENCLAW_PRIMARY_MODEL", Value: "moonshot/kimi-k2.5"},
// 				// Auth
// 				{Key: "AUTH_USERNAME", Value: "admin"},
// 				{Key: "AUTH_PASSWORD", Value: "admin@123"},
// 				{Key: "OPENCLAW_GATEWAY_TOKEN", Value: "90d42cfc7a925201a27b61ce9b6403693629d2a18094a596"},
// 				// Browser sidecar
// 				{Key: "BROWSER_CDP_URL", Value: "http://localhost" + ":" + openclawBrowserCDPPort},
// 				{Key: "BROWSER_DEFAULT_PROFILE", Value: "openclaw"},
// 				{Key: "BROWSER_EVALUATE_ENABLED", Value: "true"},
// 			},
// 		},
// 		VolumeMounts: openclawVols,
// 	}

// 	return []*computeapi.PodContainerCreateInput{
// 		{Name: fmt.Sprintf("%s-%s", llm.GetName(), browserContainerName), ContainerSpec: browserSpec},
// 		{Name: fmt.Sprintf("%s-%s", llm.GetName(), openclawContainerName), ContainerSpec: openclawSpec},
// 	}
// }

func (c *openclaw) getOpenClawBaseConfig(llm *models.SLLM) *api.OpenClawConfig {
	return &api.OpenClawConfig{
		Browser: &api.OpenClawConfigBrowser{
			Enabled:        true,
			DefaultProfile: "openclaw",
			SSRFPolicy: map[string]interface{}{
				"dangerouslyAllowPrivateNetwork": true,
			},
			Headless:  false,
			NoSandbox: false,
		},
		Agents: &api.OpenClawConfigAgents{
			"defaults": &api.OpenClawConfigAgent{
				// TODO: 支持从 llm spec 里面自动选择
				ImageModel: &api.OpenClawConfigAgentModel{
					Primary: "moonshot/kimi-k2.5",
				},
			},
		},
	}
}

func (c *openclaw) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	diskIndex := 0

	openclawVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "config",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: desktopConfigDir,
		},
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
			MountPath: openclawDataDir,
		},
		{
			Type: commonapi.CONTAINER_VOLUME_MOUNT_TYPE_TEXT,
			Text: &commonapi.ContainerVolumeMountText{
				Content: jsonutils.Marshal(c.getOpenClawBaseConfig(llm)).PrettyString(),
			},
			MountPath: api.LLM_OPENCLAW_CUSTOM_CONFIG_FILE,
		},
	}
	openclawInner := desktopWebtopImageBaseContainerSpec(image)
	openclawInner.Envs = desktopWebtopCommonEnvs(llm.GetId())
	openclawInner.Envs = append(openclawInner.Envs,
		&commonapi.ContainerKeyValue{Key: string(api.LLM_OPENCLAW_CUSTOM_CONFIG), Value: api.LLM_OPENCLAW_CUSTOM_CONFIG_FILE},
		&commonapi.ContainerKeyValue{Key: "OPENCLAW_GATEWAY_TOKEN", Value: llm.GetId()},
		&commonapi.ContainerKeyValue{Key: "OPENCLAW_GATEWAY_PORT", Value: "18789"},
		&commonapi.ContainerKeyValue{Key: "OPENCLAW_GATEWAY_BIND", Value: "loopback"},
		&commonapi.ContainerKeyValue{Key: "OPENCLAW_STATE_DIR", Value: "/config/.openclaw"},
		&commonapi.ContainerKeyValue{Key: "OPENCLAW_WORKSPACE_DIR", Value: "/config/.openclaw/workspace"},
	)
	openclawSpec := computeapi.ContainerSpec{
		ContainerSpec: openclawInner,
		VolumeMounts:  openclawVols,
		RootFs:        desktopContainerRootFs(&diskIndex),
	}
	// inject credential envs
	// spec := c.GetEffectiveSpec(llm, sku)
	if llm.LLMSpec == nil || llm.LLMSpec.OpenClaw == nil {
		return []*computeapi.PodContainerCreateInput{
			{
				Name:          fmt.Sprintf("%s-%d", llm.GetName(), 0),
				ContainerSpec: openclawSpec,
			},
		}
	}
	spec := llm.LLMSpec.OpenClaw
	for _, provider := range spec.Providers {
		openclawSpec.Envs = appendCredentialEnvs(openclawSpec.Envs, provider.Credential)
	}
	for _, channel := range spec.Channels {
		openclawSpec.Envs = appendCredentialEnvs(openclawSpec.Envs, channel.Credential)
	}

	if sku.LLMSpec != nil && sku.LLMSpec.OpenClaw != nil {
		skuSpec := sku.LLMSpec.OpenClaw
		// inject workspace templates
		if skuSpec.WorkspaceTemplates != nil {
			if skuSpec.WorkspaceTemplates.AgentsMD != "" {
				openclawSpec.Envs = append(openclawSpec.Envs, &commonapi.ContainerKeyValue{
					Key:   string(api.LLM_OPENCLAW_TEMPLATE_AGENTS_MD_B64),
					Value: base64.StdEncoding.EncodeToString([]byte(skuSpec.WorkspaceTemplates.AgentsMD)),
				})
			}
			if skuSpec.WorkspaceTemplates.SoulMD != "" {
				openclawSpec.Envs = append(openclawSpec.Envs, &commonapi.ContainerKeyValue{
					Key:   string(api.LLM_OPENCLAW_TEMPLATE_SOUL_MD_B64),
					Value: base64.StdEncoding.EncodeToString([]byte(skuSpec.WorkspaceTemplates.SoulMD)),
				})
			}
			if skuSpec.WorkspaceTemplates.UserMD != "" {
				openclawSpec.Envs = append(openclawSpec.Envs, &commonapi.ContainerKeyValue{
					Key:   string(api.LLM_OPENCLAW_TEMPLATE_USER_MD_B64),
					Value: base64.StdEncoding.EncodeToString([]byte(skuSpec.WorkspaceTemplates.UserMD)),
				})
			}
		}
	}

	return []*computeapi.PodContainerCreateInput{
		{
			Name:          fmt.Sprintf("%s-%d", llm.GetName(), 0),
			ContainerSpec: openclawSpec,
		},
	}
}

func (c *openclaw) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "https", api.LLM_DESKTOP_DEFAULT_PORT)
}

// GetLoginInfo returns OpenClaw web UI login credentials (same defaults as container env).
func (c *openclaw) GetLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (*api.LLMAccessInfo, error) {
	return getDesktopWebUILoginInfo(ctx, llm)
}

func (c *openclaw) GetProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, mdlIds ...string) (map[string]api.LLMInternalInstantMdlInfo, error) {
	return nil, nil
}

func (c *openclaw) DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	return nil, nil
}

func (c *openclaw) GetImageInternalPathMounts(sApp *models.SInstantModel) map[string]string {
	return nil
}

func (c *openclaw) GetSaveDirectories(sApp *models.SInstantModel) (string, []string, error) {
	return "", nil, nil
}

func (c *openclaw) ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error) {
	return nil, nil
}

func (c *openclaw) CheckDuplicateMounts(errStr string, dupIndex int) string {
	return "Duplicate mounts detected"
}

func (c *openclaw) GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string {
	return ""
}

func (c *openclaw) GetDirPostOverlay(dir api.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay {
	return nil
}

func (c *openclaw) PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	return nil
}

func (c *openclaw) InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, dirs []string, mdlIds []string) error {
	return nil
}

func (c *openclaw) UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	return nil
}

func (c *openclaw) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error) {
	return "", nil, nil
}
