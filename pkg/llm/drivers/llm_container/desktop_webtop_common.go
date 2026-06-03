package llm_container

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

// desktopWebtopImageBaseContainerSpec 返回 OpenClaw / Hermes 等桌面栈共用的镜像与运行时字段（不含 Envs）。
func desktopWebtopImageBaseContainerSpec(image *models.SLLMImage) commonapi.ContainerSpec {
	return commonapi.ContainerSpec{
		Image:             image.ToContainerImage(),
		ImageCredentialId: image.CredentialId,
		EnableLxcfs:       true,
		AlwaysRestart:     true,
		ShmSizeMB:         2048,
		DisableNoNewPrivs: true,
	}
}

// desktopWebtopCommonEnvs 返回 webtop / Selkies 等桌面容器共用的环境变量（时区、locale、admin 登录、Homebrew、侧边栏等）。
// uiTitle 为空时使用 "Cloudpods Desktop"。
// selkiesSidebarShow 为 true 时启用 Selkies 侧边栏 Apps / Gamepads（desktop 类型 LLM 使用）。
func desktopWebtopCommonEnvs(llmId string, uiTitle string, selkiesSidebarShow bool) []*commonapi.ContainerKeyValue {
	httpAuthUsername := "admin"
	httpAuthPassword := openclawFixed9DigitPassword(llmId)
	if uiTitle == "" {
		uiTitle = "Cloudpods Desktop"
	}
	sidebarShowApps := "False"
	sidebarShowGamepads := "False"
	if selkiesSidebarShow {
		sidebarShowApps = "True"
		sidebarShowGamepads = "True"
	}
	return []*commonapi.ContainerKeyValue{
		{Key: "TZ", Value: "Asia/Shanghai"},
		{Key: "PUID", Value: "1000"},
		{Key: "PGID", Value: "1000"},
		{Key: "LC_ALL", Value: "zh_CN.UTF-8"},
		{Key: string(api.LLM_DESKTOP_AUTH_USERNAME), Value: httpAuthUsername},
		{Key: string(api.LLM_DESKTOP_CUSTOM_USER), Value: httpAuthUsername},
		{Key: string(api.LLM_DESKTOP_AUTH_PASSWORD), Value: httpAuthPassword},
		{Key: string(api.LLM_DESKTOP_PASSWORD), Value: httpAuthPassword},
		{Key: "HOMEBREW_PREFIX", Value: "/home/linuxbrew/.linuxbrew"},
		{Key: "HOMEBREW_CELLAR", Value: "/home/linuxbrew/.linuxbrew/Cellar"},
		{Key: "HOMEBREW_REPOSITORY", Value: "/home/linuxbrew/.linuxbrew/Homebrew"},
		{Key: "SELKIES_UI_TITLE", Value: uiTitle},
		{Key: "SELKIES_UI_SHOW_LOGO", Value: "False"},
		{Key: "SELKIES_UI_SIDEBAR_SHOW_APPS", Value: sidebarShowApps},
		{Key: "SELKIES_UI_SIDEBAR_SHOW_GAMEPADS", Value: sidebarShowGamepads},
	}
}

func desktopStandardVolumeMounts(diskIndex *int, includeData bool, dataMountPath string) []*commonapi.ContainerVolumeMount {
	vols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        diskIndex,
				SubDirectory: "config",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: desktopConfigDir,
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        diskIndex,
				SubDirectory: "home",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: homeDir,
		},
	}
	if includeData && dataMountPath != "" {
		vols = append(vols, &commonapi.ContainerVolumeMount{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        diskIndex,
				SubDirectory: "data",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: dataMountPath,
		})
	}
	return vols
}

func desktopContainerRootFs(diskIndex *int) *commonapi.ContainerRootfs {
	return &commonapi.ContainerRootfs{
		Type: commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
		Disk: &commonapi.ContainerVolumeMountDisk{
			Index:        diskIndex,
			SubDirectory: "rootfs",
		},
		Persistent: false,
	}
}

const desktopDefaultDRINode = "/dev/dri/renderD128"

func desktopHasIsolatedGPU(llm *models.SLLM, sku *models.SLLMSku, devices []computeapi.SIsolatedDevice) bool {
	if len(devices) > 0 {
		return true
	}
	effDevs := models.GetEffectiveDevices(llm, sku)
	return effDevs != nil && len(*effDevs) > 0
}

func desktopGPUWaylandEnvs() []*commonapi.ContainerKeyValue {
	return []*commonapi.ContainerKeyValue{
		{Key: "PIXELFLUX_WAYLAND", Value: "true"},
		{Key: "DRINODE", Value: desktopDefaultDRINode},
		{Key: "DRI_NODE", Value: desktopDefaultDRINode},
	}
}

func appendContainerIsolatedDevices(spec *computeapi.ContainerSpec, llm *models.SLLM, sku *models.SLLMSku, devices []computeapi.SIsolatedDevice) {
	effDevs := models.GetEffectiveDevices(llm, sku)
	if len(devices) == 0 && effDevs != nil && len(*effDevs) > 0 {
		for i := range *effDevs {
			index := i
			spec.Devices = append(spec.Devices, &computeapi.ContainerDevice{
				Type: commonapi.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &computeapi.ContainerIsolatedDevice{
					Index: &index,
				},
			})
		}
		return
	}
	for i := range devices {
		spec.Devices = append(spec.Devices, &computeapi.ContainerDevice{
			Type: commonapi.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
			IsolatedDevice: &computeapi.ContainerIsolatedDevice{
				Id: devices[i].Id,
			},
		})
	}
}

// getDesktopWebUILoginInfo 从已部署容器 Spec 中解析桌面 Web UI 登录信息与 gateway token（若存在）。
func getDesktopWebUILoginInfo(ctx context.Context, llm *models.SLLM) (*api.LLMAccessInfo, error) {
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
		if env.Key == string(api.LLM_DESKTOP_AUTH_USERNAME) {
			username = env.Value
		}
		if env.Key == string(api.LLM_DESKTOP_AUTH_PASSWORD) {
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
