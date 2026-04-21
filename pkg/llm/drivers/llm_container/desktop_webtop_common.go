package llm_container

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
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
func desktopWebtopCommonEnvs(llmId string) []*commonapi.ContainerKeyValue {
	httpAuthUsername := "admin"
	httpAuthPassword := openclawFixed9DigitPassword(llmId)
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
		{Key: "SELKIES_UI_TITLE", Value: "Cloudpods Desktop"},
		{Key: "SELKIES_UI_SHOW_LOGO", Value: "False"},
		{Key: "SELKIES_UI_SIDEBAR_SHOW_APPS", Value: "False"},
		{Key: "SELKIES_UI_SIDEBAR_SHOW_GAMEPADS", Value: "False"},
	}
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
