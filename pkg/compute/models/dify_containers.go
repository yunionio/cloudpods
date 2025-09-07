package models

import (
	"errors"
	"fmt"
	"path"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type DifyContainerEnv map[string]string

func (envsPtr *DifyContainerEnv) GetContainerEnvs() []*apis.ContainerKeyValue {
	envs := *envsPtr
	if len(envs) == 0 {
		return nil
	}

	ctrEnvs := make([]*apis.ContainerKeyValue, 0, len(envs))
	for key, value := range envs {
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

type DifyContainersManager struct{}

func (m *DifyContainersManager) GetContainer(name, containerKey string) (*api.PodContainerCreateInput, error) {
	switch containerKey {
	case api.DIFY_REDIS_KEY:
		return getRedisContainer(name, containerKey), nil
	case api.DIFY_POSTGRES_KEY:
		return getPostgresContainer(name, containerKey), nil
	case api.DIFY_API_KEY:
		return getApiContainer(name, containerKey), nil
	case api.DIFY_WORKER_KEY:
		return getWorkerContainer(name, containerKey), nil
	case api.DIFY_WORKER_BEAT_KEY:
		return getWorkerBeatContainer(name, containerKey), nil
	case api.DIFY_PLUGIN_KEY:
		return getPluginContainer(name, containerKey), nil
	case api.DIFY_WEB_KEY:
		return getWebContainer(name, containerKey), nil
	case api.DIFY_SSRF_KEY:
		return getSsrfContainer(name, containerKey), nil
	case api.DIFY_NGINX_KEY:
		return getNginxContainer(name, containerKey), nil
	case api.DIFY_WEAVIATE_KEY:
		return getWeaviateContainer(name, containerKey), nil
	case api.DIFY_SANDBOX_KEY:
		return getSandboxContainer(name, containerKey), nil
	default:
		return nil, errors.New("unsupported container key")
	}
}

func getRedisContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_REDIS_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"REDISCLI_AUTH": api.REDISCLI_AUTH,
	}
	ctr.Envs = envs.GetContainerEnvs()

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.REDIS_PVC_MOUNT_PATH),
	}

	// set command
	ctr.Args = []string{
		"redis-server", "--requirepass", api.REDISCLI_AUTH,
	}

	return ctr
}

func getPostgresContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_POSTGRES_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"POSTGRES_USER":     api.POSTGRES_USER,
		"POSTGRES_PASSWORD": api.POSTGRES_PASSWORD,
		"POSTGRES_DB":       api.POSTGRES_DB,
		"PGDATA":            path.Join(api.POSTGRES_PVC_MOUNT_PATH, api.PGDATA),
	}
	ctr.Envs = envs.GetContainerEnvs()

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.POSTGRES_PVC_MOUNT_PATH),
	}

	// set command for ctr
	ctr.Args = []string{
		"postgres",
		"-c", "max_connections=" + api.POSTGRES_MAX_CONNECTIONS,
		"-c", "shared_buffers=" + api.POSTGRES_SHARED_BUFFERS,
		"-c", "work_mem=" + api.POSTGRES_WORK_MEM,
		"-c", "maintenance_work_mem=" + api.POSTGRES_MAINTENANCE_WORK_MEM,
		"-c", "effective_cache_size=" + api.POSTGRES_EFFECTIVE_CACHE_SIZE,
	}

	return ctr
}

func getApiContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_API_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"MODE":                        api.API_MODE,
		"SENTRY_DSN":                  api.API_SENTRY_DSN,
		"SENTRY_TRACES_SAMPLE_RATE":   api.API_SENTRY_TRACES_SAMPLE_RATE,
		"SENTRY_PROFILES_SAMPLE_RATE": api.API_SENTRY_PROFILES_SAMPLE_RATE,
		"PLUGIN_REMOTE_INSTALL_HOST":  api.API_PLUGIN_REMOTE_INSTALL_HOST,
		"PLUGIN_REMOTE_INSTALL_PORT":  api.API_PLUGIN_REMOTE_INSTALL_PORT,
		"PLUGIN_MAX_PACKAGE_SIZE":     api.API_PLUGIN_MAX_PACKAGE_SIZE,
		"INNER_API_KEY_FOR_PLUGIN":    api.API_INNER_API_KEY_FOR_PLUGIN,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(), envs.GetContainerEnvs()...)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, "api", api.API_PVC_MOUNT_PATH),
	}

	return ctr
}

func getWorkerContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_API_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"MODE":                        api.WORKER_MODE,
		"SENTRY_DSN":                  api.API_SENTRY_DSN,
		"SENTRY_TRACES_SAMPLE_RATE":   api.API_SENTRY_TRACES_SAMPLE_RATE,
		"SENTRY_PROFILES_SAMPLE_RATE": api.API_SENTRY_PROFILES_SAMPLE_RATE,
		"PLUGIN_MAX_PACKAGE_SIZE":     api.API_PLUGIN_MAX_PACKAGE_SIZE,
		"INNER_API_KEY_FOR_PLUGIN":    api.API_INNER_API_KEY_FOR_PLUGIN,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(), envs.GetContainerEnvs()...)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, "api", api.API_PVC_MOUNT_PATH),
	}

	return ctr
}

func getWorkerBeatContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_API_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"MODE": api.WORKER_BEAT_MODE,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(), envs.GetContainerEnvs()...)

	return ctr
}

func getPluginContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_PLUGIN_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"DB_DATABASE":                   api.PLUGIN_DB_DATABASE,
		"SERVER_PORT":                   api.PLUGIN_SERVER_PORT,
		"SERVER_KEY":                    api.PLUGIN_SERVER_KEY,
		"MAX_PACKAGE_CACHE_PATH":        api.PLUGIN_PACKAGE_CACHE_PATH,
		"PPROF_ENABLED":                 api.PLUGIN_PPROF_ENABLED,
		"DIFY_INNER_API_URL":            api.PLUGIN_DIFY_INNER_API_URL,
		"DIFY_INNER_API_KEY":            api.PLUGIN_DIFY_INNER_API_KEY,
		"PLUGIN_REMOTE_INSTALLING_HOST": api.PLUGIN_REMOTE_INSTALLING_HOST,
		"PLUGIN_REMOTE_INSTALLING_PORT": api.PLUGIN_REMOTE_INSTALLING_PORT,
		"PLUGIN_WORKING_PATH":           api.PLUGIN_WORKING_PATH,
		"FORCE_VERIFYING_SIGNATURE":     api.PLUGIN_FORCE_VERIFYING_SIGNATURE,
		"PYTHON_ENV_INIT_TIMEOUT":       api.PLUGIN_PYTHON_ENV_INIT_TIMEOUT,
		"PLUGIN_MAX_EXECUTION_TIMEOUT":  api.PLUGIN_MAX_EXECUTION_TIMEOUT,
		"PIP_MIRROR_URL":                api.PIP_MIRROR_URL,
		"PLUGIN_STORAGE_TYPE":           api.PLUGIN_STORAGE_TYPE,
		"PLUGIN_STORAGE_LOCAL_ROOT":     api.PLUGIN_STORAGE_LOCAL_ROOT,
		"PLUGIN_INSTALLED_PATH":         api.PLUGIN_INSTALLED_PATH,
		"PLUGIN_PACKAGE_CACHE_PATH":     api.PLUGIN_PACKAGE_CACHE_PATH,
		"PLUGIN_MEDIA_CACHE_PATH":       api.PLUGIN_MEDIA_CACHE_PATH,
	}
	ctr.Envs = append(getSharedApiWorkerEnv(), envs.GetContainerEnvs()...)

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.PLUGIN_STORAGE_LOCAL_ROOT),
	}

	return ctr
}

func getWebContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_WEB_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"HOSTNAME":                                "", // set HOSTNAME to empty, to avoid Error: getaddrinfo ENOTFOUND
		"CONSOLE_API_URL":                         api.WEB_CONSOLE_API_URL,
		"APP_API_URL":                             api.WEB_APP_API_URL,
		"SENTRY_DSN":                              api.WEB_SENTRY_DSN,
		"NEXT_TELEMETRY_DISABLED":                 api.WEB_NEXT_TELEMETRY_DISABLED,
		"TEXT_GENERATION_TIMEOUT_MS":              api.WEB_TEXT_GENERATION_TIMEOUT_MS,
		"CSP_WHITELIST":                           api.WEB_CSP_WHITELIST,
		"ALLOW_EMBED":                             api.WEB_ALLOW_EMBED,
		"ALLOW_UNSAFE_DATA_SCHEME":                api.WEB_ALLOW_UNSAFE_DATA_SCHEME,
		"MARKETPLACE_API_URL":                     api.WEB_MARKETPLACE_API_URL,
		"MARKETPLACE_URL":                         api.WEB_MARKETPLACE_URL,
		"TOP_K_MAX_VALUE":                         api.WEB_TOP_K_MAX_VALUE,
		"INDEXING_MAX_SEGMENTATION_TOKENS_LENGTH": api.WEB_INDEXING_MAX_SEGMENTATION_TOKENS_LENGTH,
		"PM2_INSTANCES":                           api.WEB_PM2_INSTANCES,
		"LOOP_NODE_MAX_COUNT":                     api.WEB_LOOP_NODE_MAX_COUNT,
		"MAX_TOOLS_NUM":                           api.WEB_MAX_TOOLS_NUM,
		"MAX_PARALLEL_LIMIT":                      api.WEB_MAX_PARALLEL_LIMIT,
		"MAX_ITERATIONS_NUM":                      api.WEB_MAX_ITERATIONS_NUM,
		"ENABLE_WEBSITE_JINAREADER":               api.WEB_ENABLE_WEBSITE_JINAREADER,
		"ENABLE_WEBSITE_FIRECRAWL":                api.WEB_ENABLE_WEBSITE_FIRECRAWL,
		"ENABLE_WEBSITE_WATERCRAWL":               api.WEB_ENABLE_WEBSITE_WATERCRAWL,
	}
	ctr.Envs = envs.GetContainerEnvs()

	return ctr
}

func getSsrfContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_SSRF_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		// "VISIBLE_HOSTNAME":   "localhost", // set VISIBLE_HOSTNAME to localhost, to avoid rDNS test failed
		"HTTP_PORT":          api.SSRF_HTTP_PORT,
		"COREDUMP_DIR":       api.SSRF_COREDUMP_DIR,
		"REVERSE_PROXY_PORT": api.SSRF_REVERSE_PROXY_PORT,
		"SANDBOX_HOST":       api.SSRF_SANDBOX_HOST,
		"SANDBOX_PORT":       api.SSRF_SANDBOX_PORT,
	}
	ctr.Envs = envs.GetContainerEnvs()

	// set PVC to store data
	// ctr.VolumeMounts = []*apis.ContainerVolumeMount{
	// 	getPVCMount(key, key, api.SSRF_MOUNT_PATH),
	// }

	// generate entrypoint
	entrypointSH := fmt.Sprintf(api.SSRF_ENTRYPINT_SHELL, api.SSRF_SQUID_CONFIGURATION_FILE)
	ctr.Command = []string{
		"/bin/sh", "-c", entrypointSH,
	}

	return ctr
}

func getNginxContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_NGINX_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"NGINX_SERVER_NAME":          api.NGINX_SERVER_NAME,
		"NGINX_PORT":                 api.NGINX_PORT,
		"NGINX_WORKER_PROCESSES":     api.NGINX_WORKER_PROCESSES,
		"NGINX_CLIENT_MAX_BODY_SIZE": api.NGINX_CLIENT_MAX_BODY_SIZE,
		"NGINX_KEEPALIVE_TIMEOUT":    api.NGINX_KEEPALIVE_TIMEOUT,
		"NGINX_PROXY_READ_TIMEOUT":   api.NGINX_PROXY_READ_TIMEOUT,
		"NGINX_PROXY_SEND_TIMEOUT":   api.NGINX_PROXY_SEND_TIMEOUT,
	}
	ctr.Envs = envs.GetContainerEnvs()

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.NGINX_MOUNT_PATH),
	}

	// generate entrypoint
	entrypointSH := fmt.Sprintf(api.NGINX_ENTRYPINT_SHELL, api.NGINX_NGINX_CONF_FILE, api.NGINX_PROXY_CONF_FILE, api.NGINX_DEFAULT_CONF_FILE)
	ctr.Command = []string{
		"/bin/sh", "-c", entrypointSH,
	}

	return ctr
}

func getWeaviateContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_WEAVIATE_IAMGE

	// set container environments
	envs := &DifyContainerEnv{
		"PERSISTENCE_DATA_PATH":                   api.WEAVIATE_PERSISTENCE_DATA_PATH,
		"QUERY_DEFAULTS_LIMIT":                    api.WEAVIATE_QUERY_DEFAULTS_LIMIT,
		"AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED": api.WEAVIATE_AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED,
		"DEFAULT_VECTORIZER_MODULE":               api.WEAVIATE_DEFAULT_VECTORIZER_MODULE,
		"CLUSTER_HOSTNAME":                        api.WEAVIATE_CLUSTER_HOSTNAME,
		"AUTHENTICATION_APIKEY_ENABLED":           api.WEAVIATE_AUTHENTICATION_APIKEY_ENABLED,
		"AUTHENTICATION_APIKEY_ALLOWED_KEYS":      api.WEAVIATE_AUTHENTICATION_APIKEY_ALLOWED_KEYS,
		"AUTHENTICATION_APIKEY_USERS":             api.WEAVIATE_AUTHENTICATION_APIKEY_USERS,
		"AUTHORIZATION_ADMINLIST_ENABLED":         api.WEAVIATE_AUTHORIZATION_ADMINLIST_ENABLED,
		"AUTHORIZATION_ADMINLIST_USERS":           api.WEAVIATE_AUTHORIZATION_ADMINLIST_USERS,
	}
	ctr.Envs = envs.GetContainerEnvs()

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key, key, api.WEAVIATE_PERSISTENCE_DATA_PATH),
	}

	return ctr
}

func getSandboxContainer(name, key string) *api.PodContainerCreateInput {
	// set name and image
	ctr := &api.PodContainerCreateInput{
		Name: name + "-" + key,
	}
	ctr.Image = api.DIFY_IMAGE_REGISTRY + api.DIFY_SANDBOX_IMAGE

	// set container environments
	envs := &DifyContainerEnv{
		"API_KEY":        api.SANDBOX_API_KEY,
		"GIN_MODE":       api.SANDBOX_GIN_MODE,
		"WORKER_TIMEOUT": api.SANDBOX_WORKER_TIMEOUT,
		"ENABLE_NETWORK": api.SANDBOX_ENABLE_NETWORK,
		"HTTP_PROXY":     api.SANDBOX_HTTP_PROXY,
		"HTTPS_PROXY":    api.SANDBOX_HTTPS_PROXY,
		"SANDBOX_PORT":   api.SANDBOX_PORT,
		"PIP_MIRROR_URL": api.PIP_MIRROR_URL,
	}
	ctr.Envs = envs.GetContainerEnvs()

	// set PVC to store data
	ctr.VolumeMounts = []*apis.ContainerVolumeMount{
		getPVCMount(key+"conf", key+"conf", api.SANDBOX_CONF_MOUNT_PATH),
		getPVCMount(key+"dep", key+"dep", api.SANDBOX_DEP_MOUNT_PATH),
	}

	// set command
	writeConfigCommand := fmt.Sprintf(api.SANDBOX_WRITE_CONF_SHELL, api.SANDBOX_CONF_FILE, api.SANDBOX_CONF_TEMP_FILE)
	ctr.Command = []string{
		"/bin/sh", "-c", writeConfigCommand,
	}

	return ctr
}
