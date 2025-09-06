package compute

const (
	DIFY_POSTGRES_KEY    = "postgres"
	DIFY_REDIS_KEY       = "redis"
	DIFY_API_KEY         = "api"
	DIFY_WORKER_KEY      = "worker"
	DIFY_WORKER_BEAT_KEY = "beat"
	DIFY_PLUGIN_KEY      = "plugin"
	DIFY_WEB_KEY         = "web"
	DIFY_SSRF_KEY        = "ssrf"
	DIFY_NGINX_KEY       = "nginx"
)

const (
	POSTGRES_PVC_MOUNT_PATH       = "/var/lib/postgresql/data"
	POSTGRES_MAX_CONNECTIONS      = "100"
	POSTGRES_SHARED_BUFFERS       = "128MB"
	POSTGRES_WORK_MEM             = "4MB"
	POSTGRES_MAINTENANCE_WORK_MEM = "64MB"
	POSTGRES_EFFECTIVE_CACHE_SIZE = "4096MB"
	POSTGRES_USER                 = "postgres"
	POSTGRES_PASSWORD             = "difyai123456"
	POSTGRES_DB                   = "dify"
	PGDATA                        = "/pgdata"
	POSTGRES_UMASK                = 70
	POSTGRES_GMASK                = 70
)

const (
	REDISCLI_AUTH        = "difyai123456"
	REDIS_PVC_MOUNT_PATH = "/data"
)

const (
	API_PVC_MOUNT_PATH              = "/app/api/storage"
	API_MODE                        = "api"
	API_SENTRY_DSN                  = ""
	API_SENTRY_TRACES_SAMPLE_RATE   = "1.0"
	API_SENTRY_PROFILES_SAMPLE_RATE = "1.0"
	API_PLUGIN_REMOTE_INSTALL_HOST  = "localhost"
	API_PLUGIN_REMOTE_INSTALL_PORT  = "5003"
	API_PLUGIN_MAX_PACKAGE_SIZE     = "52428800"
	API_INNER_API_KEY_FOR_PLUGIN    = "QaHbTe77CtuXmsfyhR7+vRjI/+XbV1AaFy691iy+kGDv2Jvy0/eAh8Y1"
)

const (
	WORKER_MODE      = "worker"
	WORKER_BEAT_MODE = "beat"
)

const (
	PLUGIN_DB_DATABASE               = "dify_plugin"
	PLUGIN_SERVER_PORT               = "5002"
	PLUGIN_SERVER_KEY                = "lYkiYYT6owG+71oLerGzA7GXCgOT++6ovaezWAjpCjf+Sjc3ZtU+qUEi"
	PLUGIN_MAX_PACKAGE_SIZE          = "52428800"
	PLUGIN_PPROF_ENABLED             = "false"
	PLUGIN_DIFY_INNER_API_URL        = "http://localhost:5001"
	PLUGIN_DIFY_INNER_API_KEY        = "QaHbTe77CtuXmsfyhR7+vRjI/+XbV1AaFy691iy+kGDv2Jvy0/eAh8Y1"
	PLUGIN_REMOTE_INSTALLING_HOST    = "0.0.0.0"
	PLUGIN_REMOTE_INSTALLING_PORT    = "5003"
	PLUGIN_WORKING_PATH              = "/app/storage/cwd"
	PLUGIN_FORCE_VERIFYING_SIGNATURE = "true"
	PLUGIN_PYTHON_ENV_INIT_TIMEOUT   = "120"
	PLUGIN_MAX_EXECUTION_TIMEOUT     = "600"
	PLUGIN_PIP_MIRROR_URL            = "https://mirrors.aliyun.com/pypi/simple"
	PLUGIN_STORAGE_TYPE              = "local"
	PLUGIN_STORAGE_LOCAL_ROOT        = "/app/storage"
	PLUGIN_INSTALLED_PATH            = "plugin"
	PLUGIN_PACKAGE_CACHE_PATH        = "plugin_packages"
	PLUGIN_MEDIA_CACHE_PATH          = "assets"
)

const (
	WEB_CONSOLE_API_URL                         = "http://localhost:5001"
	WEB_APP_API_URL                             = "http://localhost:3000"
	WEB_SENTRY_DSN                              = ""
	WEB_NEXT_TELEMETRY_DISABLED                 = "0"
	WEB_TEXT_GENERATION_TIMEOUT_MS              = "60000"
	WEB_CSP_WHITELIST                           = ""
	WEB_ALLOW_EMBED                             = "false"
	WEB_ALLOW_UNSAFE_DATA_SCHEME                = "false"
	WEB_MARKETPLACE_API_URL                     = "https://marketplace.dify.ai"
	WEB_MARKETPLACE_URL                         = "https://marketplace.dify.ai"
	WEB_TOP_K_MAX_VALUE                         = ""
	WEB_INDEXING_MAX_SEGMENTATION_TOKENS_LENGTH = ""
	WEB_PM2_INSTANCES                           = "2"
	WEB_LOOP_NODE_MAX_COUNT                     = "100"
	WEB_MAX_TOOLS_NUM                           = "10"
	WEB_MAX_PARALLEL_LIMIT                      = "10"
	WEB_MAX_ITERATIONS_NUM                      = "99"
	WEB_ENABLE_WEBSITE_JINAREADER               = "true"
	WEB_ENABLE_WEBSITE_FIRECRAWL                = "true"
	WEB_ENABLE_WEBSITE_WATERCRAWL               = "true"
)

const (
	SSRF_MOUNT_PATH         = "/etc/squid/"
	SSRF_HTTP_PORT          = "3128"
	SSRF_COREDUMP_DIR       = "/var/spool/squid"
	SSRF_REVERSE_PROXY_PORT = "8194"
	SSRF_SANDBOX_HOST       = "localhost"
	SSRF_SANDBOX_PORT       = "8194"
)

const (
	NGINX_MOUNT_PATH           = "/etc/nginx/conf.d"
	NGINX_SERVER_NAME          = "_"
	NGINX_PORT                 = "80"
	NGINX_WORKER_PROCESSES     = "auto"
	NGINX_CLIENT_MAX_BODY_SIZE = "100M"
	NGINX_KEEPALIVE_TIMEOUT    = "65"
	NGINX_PROXY_READ_TIMEOUT   = "3600s"
	NGINX_PROXY_SEND_TIMEOUT   = "3600s"
)

const (
	DIFY_CREATED                   = "running"
	DIFY_DEPLOY_REDIS_FAILED       = "dify_deploy_redis_failed"
	DIFY_DEPLOY_POSTGRES_FAILED    = "dify_deploy_postgres_failed"
	DIFY_DEPLOY_API_FAILED         = "dify_deploy_api_failed"
	DIFY_DEPLOY_WORKER_FAILED      = "dify_deploy_worker_failed"
	DIFY_DEPLOY_WORKER_BEAT_FAILED = "dify_deploy_worker_beat_failed"
	DIFY_DEPLOY_WEB_FAILED         = "dify_deploy_web_failed"
	DIFY_DEPLOY_PLUGIN_FAILED      = "dify_deploy_plugin_failed"
	DIFY_DEPLOY_SSRF_FAILED        = "dify_deploy_ssrf_failed"
	DIFY_DEPLOY_NGINX_FAILED       = "dify_deploy_nginx_failed"
	DIFY_CREATE_FAILED             = "create_dify_failed"
)

type DifyCreateInput struct {
	ServerCreateInput
}
