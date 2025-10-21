package models

import (
	"errors"
	"fmt"
	"path"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

type DifyContainerEnv map[string]string

func (envsPtr *DifyContainerEnv) GetContainerEnvs(userCustomizedEnvs *DifyContainerEnv) []*apis.ContainerKeyValue {
	envs := *envsPtr
	if len(envs) == 0 {
		return nil
	}

	ctrEnvs := make([]*apis.ContainerKeyValue, 0, len(envs))
	for key, value := range envs {
		if userCustomizedEnvs != nil && (*userCustomizedEnvs)[key] != "" {
			value = (*userCustomizedEnvs)[key]
		}
		ctrEnvs = append(ctrEnvs, &apis.ContainerKeyValue{
			Key:   key,
			Value: value,
		})
	}
	return ctrEnvs
}

func (envsPtr *DifyContainerEnv) SetContainerEnv(key, value string) error {
	if key == "" {
		return errors.New("environment variable key cannot be empty")
	}

	if envsPtr == nil {
		return errors.New("DifyContainerEnv pointer is nil")
	}

	if *envsPtr == nil {
		*envsPtr = make(DifyContainerEnv)
	}

	(*envsPtr)[key] = value
	return nil
}

func getPVCMount(name, subDir, mountPath string) *apis.ContainerVolumeMount {
	diskIndex := 0
	pvc := &apis.ContainerVolumeMount{
		UniqueName: name + "-data-pvc",
		Type:       apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
		Disk: &apis.ContainerVolumeMountDisk{
			SubDirectory: subDir,
			Index:        &diskIndex,
		},
		MountPath:   mountPath,
		ReadOnly:    false,
		Propagation: apis.MOUNTPROPAGATION_PROPAGATION_PRIVATE,
	}

	return pvc
}

type DifyContainersManager struct {
	UserCustomizedEnvs *DifyContainerEnv
}

func _getRegistryImage(imageId string) string {
	image, err := GetLLMImageManager().FetchById(imageId)
	if err != nil {
		return ""
	}
	return image.(*SLLMImage).ToContainerImage()
}

func (m *DifyContainersManager) GetContainer(name, containerKey string, sku *SDifyModel) (*computeapi.PodContainerCreateInput, error) {
	switch containerKey {
	case api.DIFY_REDIS_KEY:
		return m._getRedisContainer(name, containerKey, _getRegistryImage(sku.RedisImageId)), nil
	case api.DIFY_POSTGRES_KEY:
		return m._getPostgresContainer(name, containerKey, _getRegistryImage(sku.PostgresImageId)), nil
	case api.DIFY_API_KEY:
		return m._getApiContainer(name, containerKey, _getRegistryImage(sku.DifyApiImageId)), nil
	case api.DIFY_WORKER_KEY:
		return m._getWorkerContainer(name, containerKey, _getRegistryImage(sku.DifyApiImageId)), nil
	case api.DIFY_WORKER_BEAT_KEY:
		return m._getWorkerBeatContainer(name, containerKey, _getRegistryImage(sku.DifyApiImageId)), nil
	case api.DIFY_PLUGIN_KEY:
		return m._getPluginContainer(name, containerKey, _getRegistryImage(sku.DifyPluginImageId)), nil
	case api.DIFY_WEB_KEY:
		return m._getWebContainer(name, containerKey, _getRegistryImage(sku.DifyWebImageId)), nil
	case api.DIFY_SSRF_KEY:
		return m._getSsrfContainer(name, containerKey, _getRegistryImage(sku.DifySSRFImageId)), nil
	case api.DIFY_NGINX_KEY:
		return m._getNginxContainer(name, containerKey, _getRegistryImage(sku.NginxImageId)), nil
	case api.DIFY_WEAVIATE_KEY:
		return m._getWeaviateContainer(name, containerKey, _getRegistryImage(sku.DifyWeaviateImageId)), nil
	case api.DIFY_SANDBOX_KEY:
		return m._getSandboxContainer(name, containerKey, _getRegistryImage(sku.DifySandboxImageId)), nil
	default:
		return nil, errors.New("unsupported container key")
	}
}

func (m *DifyContainersManager) _getRedisContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"REDISCLI_AUTH": api.DIFY_REDISCLI_AUTH,
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.DIFY_REDIS_PVC_MOUNT_PATH),
	}

	// set command
	ctr.Args = []string{
		"redis-server", "--requirepass", api.DIFY_REDISCLI_AUTH,
	}

	return ctr
}

func (m *DifyContainersManager) _getPostgresContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"POSTGRES_USER":     api.DIFY_POSTGRES_USER,
		"POSTGRES_PASSWORD": api.DIFY_POSTGRES_PASSWORD,
		"POSTGRES_DB":       api.DIFY_POSTGRES_DB,
		"PGDATA":            path.Join(api.DIFY_POSTGRES_PVC_MOUNT_PATH, api.DIFY_POSTGRES_PGDATA),
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.DIFY_POSTGRES_PVC_MOUNT_PATH),
	}

	// set command for ctr
	ctr.Args = []string{
		"postgres",
		"-c", "max_connections=" + api.DIFY_POSTGRES_MAX_CONNECTIONS,
		"-c", "shared_buffers=" + api.DIFY_POSTGRES_SHARED_BUFFERS,
		"-c", "work_mem=" + api.DIFY_POSTGRES_WORK_MEM,
		"-c", "maintenance_work_mem=" + api.DIFY_POSTGRES_MAINTENANCE_WORK_MEM,
		"-c", "effective_cache_size=" + api.DIFY_POSTGRES_EFFECTIVE_CACHE_SIZE,
	}

	return ctr
}

func (m *DifyContainersManager) _getApiContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"MODE":                        api.DIFY_API_MODE,
		"SENTRY_DSN":                  api.DIFY_API_SENTRY_DSN,
		"SENTRY_TRACES_SAMPLE_RATE":   api.DIFY_API_SENTRY_TRACES_SAMPLE_RATE,
		"SENTRY_PROFILES_SAMPLE_RATE": api.DIFY_API_SENTRY_PROFILES_SAMPLE_RATE,
		"PLUGIN_REMOTE_INSTALL_HOST":  api.DIFY_LOCALHOST,
		"PLUGIN_REMOTE_INSTALL_PORT":  api.DIFY_PLUGIN_REMOTE_INSTALLING_PORT,
		"PLUGIN_MAX_PACKAGE_SIZE":     api.DIFY_PLUGIN_MAX_PACKAGE_SIZE,
		"INNER_API_KEY_FOR_PLUGIN":    api.DIFY_API_INNER_KEY,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(m.UserCustomizedEnvs), envs.GetContainerEnvs(m.UserCustomizedEnvs)...)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, "api", api.DIFY_API_PVC_MOUNT_PATH),
	}

	return ctr
}

func (m *DifyContainersManager) _getWorkerContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"MODE":                        api.DIFY_WORKER_MODE,
		"SENTRY_DSN":                  api.DIFY_API_SENTRY_DSN,
		"SENTRY_TRACES_SAMPLE_RATE":   api.DIFY_API_SENTRY_TRACES_SAMPLE_RATE,
		"SENTRY_PROFILES_SAMPLE_RATE": api.DIFY_API_SENTRY_PROFILES_SAMPLE_RATE,
		"PLUGIN_MAX_PACKAGE_SIZE":     api.DIFY_PLUGIN_MAX_PACKAGE_SIZE,
		"INNER_API_KEY_FOR_PLUGIN":    api.DIFY_API_INNER_KEY,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(m.UserCustomizedEnvs), envs.GetContainerEnvs(m.UserCustomizedEnvs)...)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, "api", api.DIFY_API_PVC_MOUNT_PATH),
	}

	return ctr
}

func (m *DifyContainersManager) _getWorkerBeatContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"MODE": api.DIFY_WORKER_BEAT_MODE,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(m.UserCustomizedEnvs), envs.GetContainerEnvs(m.UserCustomizedEnvs)...)

	return ctr
}

func (m *DifyContainersManager) _getPluginContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"DB_DATABASE":                   api.DIFY_PLUGIN_DB_DATABASE,
		"SERVER_PORT":                   api.DIFY_PLUGIN_SERVER_PORT,
		"SERVER_KEY":                    api.DIFY_PLUGIN_SERVER_KEY,
		"MAX_PACKAGE_CACHE_PATH":        api.DIFY_PLUGIN_PACKAGE_CACHE_PATH,
		"PPROF_ENABLED":                 api.DIFY_PLUGIN_PPROF_ENABLED,
		"DIFY_INNER_API_URL":            api.DIFY_PLUGIN_DIFY_INNER_API_URL,
		"DIFY_INNER_API_KEY":            api.DIFY_API_INNER_KEY,
		"PLUGIN_REMOTE_INSTALLING_HOST": api.DIFY_PLUGIN_REMOTE_INSTALLING_HOST,
		"PLUGIN_REMOTE_INSTALLING_PORT": api.DIFY_PLUGIN_REMOTE_INSTALLING_PORT,
		"PLUGIN_WORKING_PATH":           api.DIFY_PLUGIN_WORKING_PATH,
		"FORCE_VERIFYING_SIGNATURE":     api.DIFY_PLUGIN_FORCE_VERIFYING_SIGNATURE,
		"PYTHON_ENV_INIT_TIMEOUT":       api.DIFY_PLUGIN_PYTHON_ENV_INIT_TIMEOUT,
		"PLUGIN_MAX_EXECUTION_TIMEOUT":  api.DIFY_PLUGIN_MAX_EXECUTION_TIMEOUT,
		"PIP_MIRROR_URL":                api.PIP_MIRROR_URL,
		"PLUGIN_STORAGE_TYPE":           api.DIFY_PLUGIN_STORAGE_TYPE,
		"PLUGIN_STORAGE_LOCAL_ROOT":     api.DIFY_PLUGIN_STORAGE_LOCAL_ROOT,
		"PLUGIN_INSTALLED_PATH":         api.DIFY_PLUGIN_INSTALLED_PATH,
		"PLUGIN_PACKAGE_CACHE_PATH":     api.DIFY_PLUGIN_PACKAGE_CACHE_PATH,
		"PLUGIN_MEDIA_CACHE_PATH":       api.DIFY_PLUGIN_MEDIA_CACHE_PATH,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(m.UserCustomizedEnvs), envs.GetContainerEnvs(m.UserCustomizedEnvs)...)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.DIFY_PLUGIN_STORAGE_LOCAL_ROOT),
	}

	return ctr
}

func (m *DifyContainersManager) _getWebContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"HOSTNAME":                                "", // set HOSTNAME to empty, to avoid Error: getaddrinfo ENOTFOUND
		"CONSOLE_API_URL":                         api.DIFY_WEB_CONSOLE_API_URL,
		"APP_API_URL":                             api.DIFY_WEB_APP_API_URL,
		"SENTRY_DSN":                              api.DIFY_WEB_SENTRY_DSN,
		"NEXT_TELEMETRY_DISABLED":                 api.DIFY_WEB_NEXT_TELEMETRY_DISABLED,
		"TEXT_GENERATION_TIMEOUT_MS":              api.DIFY_WEB_TEXT_GENERATION_TIMEOUT_MS,
		"CSP_WHITELIST":                           api.DIFY_WEB_CSP_WHITELIST,
		"ALLOW_EMBED":                             api.DIFY_WEB_ALLOW_EMBED,
		"ALLOW_UNSAFE_DATA_SCHEME":                api.DIFY_WEB_ALLOW_UNSAFE_DATA_SCHEME,
		"MARKETPLACE_API_URL":                     api.DIFY_WEB_MARKETPLACE_API_URL,
		"MARKETPLACE_URL":                         api.DIFY_WEB_MARKETPLACE_URL,
		"TOP_K_MAX_VALUE":                         api.DIFY_WEB_TOP_K_MAX_VALUE,
		"INDEXING_MAX_SEGMENTATION_TOKENS_LENGTH": api.DIFY_WEB_INDEXING_MAX_SEGMENTATION_TOKENS_LENGTH,
		"PM2_INSTANCES":                           api.DIFY_WEB_PM2_INSTANCES,
		"LOOP_NODE_MAX_COUNT":                     api.DIFY_WEB_LOOP_NODE_MAX_COUNT,
		"MAX_TOOLS_NUM":                           api.DIFY_WEB_MAX_TOOLS_NUM,
		"MAX_PARALLEL_LIMIT":                      api.DIFY_WEB_MAX_PARALLEL_LIMIT,
		"MAX_ITERATIONS_NUM":                      api.DIFY_WEB_MAX_ITERATIONS_NUM,
		"ENABLE_WEBSITE_JINAREADER":               api.DIFY_WEB_ENABLE_WEBSITE_JINAREADER,
		"ENABLE_WEBSITE_FIRECRAWL":                api.DIFY_WEB_ENABLE_WEBSITE_FIRECRAWL,
		"ENABLE_WEBSITE_WATERCRAWL":               api.DIFY_WEB_ENABLE_WEBSITE_WATERCRAWL,
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	return ctr
}

func (m *DifyContainersManager) _getSsrfContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		// "VISIBLE_HOSTNAME":   "localhost", // set VISIBLE_HOSTNAME to localhost, to avoid rDNS test failed
		"HTTP_PORT":          api.DIFY_SSRF_HTTP_PORT,
		"COREDUMP_DIR":       api.DIFY_SSRF_COREDUMP_DIR,
		"REVERSE_PROXY_PORT": api.DIFY_SANDBOX_PORT,
		"SANDBOX_HOST":       api.DIFY_LOCALHOST,
		"SANDBOX_PORT":       api.DIFY_SANDBOX_PORT,
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	// set PVC to store data
	// ctr.VolumeMounts = []*apis.ContainerVolumeMount{
	// 	getPVCMount(key, key, api.SSRF_MOUNT_PATH),
	// }

	// generate entrypoint
	entrypointSH := fmt.Sprintf(api.DIFY_SSRF_ENTRYPINT_SHELL, api.DIFY_SSRF_SQUID_CONFIGURATION_FILE)
	ctr.Command = []string{
		"/bin/sh", "-c", entrypointSH,
	}

	return ctr
}

func (m *DifyContainersManager) _getNginxContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"NGINX_SERVER_NAME":          api.DIFY_NGINX_SERVER_NAME,
		"NGINX_PORT":                 api.DIFY_NGINX_PORT,
		"NGINX_WORKER_PROCESSES":     api.DIFY_NGINX_WORKER_PROCESSES,
		"NGINX_CLIENT_MAX_BODY_SIZE": api.DIFY_NGINX_CLIENT_MAX_BODY_SIZE,
		"NGINX_KEEPALIVE_TIMEOUT":    api.DIFY_NGINX_KEEPALIVE_TIMEOUT,
		"NGINX_PROXY_READ_TIMEOUT":   api.DIFY_NGINX_PROXY_READ_TIMEOUT,
		"NGINX_PROXY_SEND_TIMEOUT":   api.DIFY_NGINX_PROXY_SEND_TIMEOUT,
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.DIFY_NGINX_MOUNT_PATH),
	}

	// generate entrypoint
	entrypointSH := fmt.Sprintf(api.DIFY_NGINX_ENTRYPINT_SHELL, api.DIFY_NGINX_NGINX_CONF_FILE, api.DIFY_NGINX_PROXY_CONF_FILE, api.DIFY_NGINX_DEFAULT_CONF_FILE)
	ctr.Command = []string{
		"/bin/sh", "-c", entrypointSH,
	}

	return ctr
}

func (m *DifyContainersManager) _getWeaviateContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"PERSISTENCE_DATA_PATH":                   api.DIFY_WEAVIATE_PERSISTENCE_DATA_PATH,
		"QUERY_DEFAULTS_LIMIT":                    api.DIFY_WEAVIATE_QUERY_DEFAULTS_LIMIT,
		"AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED": api.DIFY_WEAVIATE_AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED,
		"DEFAULT_VECTORIZER_MODULE":               api.DIFY_WEAVIATE_DEFAULT_VECTORIZER_MODULE,
		"CLUSTER_HOSTNAME":                        api.DIFY_WEAVIATE_CLUSTER_HOSTNAME,
		"AUTHENTICATION_APIKEY_ENABLED":           api.DIFY_WEAVIATE_AUTHENTICATION_APIKEY_ENABLED,
		"AUTHENTICATION_APIKEY_ALLOWED_KEYS":      api.DIFY_WEAVIATE_AUTHENTICATION_APIKEY_ALLOWED_KEYS,
		"AUTHENTICATION_APIKEY_USERS":             api.DIFY_WEAVIATE_AUTHENTICATION_APIKEY_USERS,
		"AUTHORIZATION_ADMINLIST_ENABLED":         api.DIFY_WEAVIATE_AUTHORIZATION_ADMINLIST_ENABLED,
		"AUTHORIZATION_ADMINLIST_USERS":           api.DIFY_WEAVIATE_AUTHORIZATION_ADMINLIST_USERS,
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.DIFY_WEAVIATE_PERSISTENCE_DATA_PATH),
	}

	return ctr
}

func (m *DifyContainersManager) _getSandboxContainer(name, key, image string) *computeapi.PodContainerCreateInput {
	// set name and image
	ctr := &computeapi.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = image

	// set container environments
	envs := &DifyContainerEnv{
		"API_KEY":        api.DIFY_SANDBOX_API_KEY,
		"GIN_MODE":       api.DIFY_SANDBOX_GIN_MODE,
		"WORKER_TIMEOUT": api.DIFY_SANDBOX_WORKER_TIMEOUT,
		"ENABLE_NETWORK": api.DIFY_SANDBOX_ENABLE_NETWORK,
		"HTTP_PROXY":     api.DIFY_SANDBOX_HTTP_PROXY,
		"HTTPS_PROXY":    api.DIFY_SANDBOX_HTTPS_PROXY,
		"SANDBOX_PORT":   api.DIFY_SANDBOX_PORT,
		"PIP_MIRROR_URL": api.PIP_MIRROR_URL,
	}
	ctr.Envs = envs.GetContainerEnvs(m.UserCustomizedEnvs)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key+"conf", key+"conf", api.DIFY_SANDBOX_CONF_MOUNT_PATH),
		getPVCMount(key+"dep", key+"dep", api.DIFY_SANDBOX_DEP_MOUNT_PATH),
	}

	// set command
	writeConfigCommand := fmt.Sprintf(api.DIFY_SANDBOX_WRITE_CONF_SHELL, api.DIFY_SANDBOX_CONF_FILE, api.DIFY_SANDBOX_CONF_TEMP_FILE)
	ctr.Command = []string{
		"/bin/sh", "-c", writeConfigCommand,
	}

	return ctr
}
