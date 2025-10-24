package llm

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

var (
	DIFY_SECRET_KEY                                  string
	DIFY_API_INNER_KEY                               string
	DIFY_PLUGIN_SERVER_KEY                           string
	DIFY_WEAVIATE_AUTHENTICATION_APIKEY_ALLOWED_KEYS string
)

func init() {
	skBytes := make([]byte, 32)
	rand.Read(skBytes)
	DIFY_SECRET_KEY = "sk-" + hex.EncodeToString(skBytes)

	innerKeyBytes := make([]byte, 32)
	rand.Read(innerKeyBytes)
	DIFY_API_INNER_KEY = base64.StdEncoding.EncodeToString(innerKeyBytes)

	pluginKeyBytes := make([]byte, 32)
	rand.Read(pluginKeyBytes)
	DIFY_PLUGIN_SERVER_KEY = base64.StdEncoding.EncodeToString(pluginKeyBytes)

	weaviateKeyBytes := make([]byte, 32)
	rand.Read(weaviateKeyBytes)
	DIFY_WEAVIATE_AUTHENTICATION_APIKEY_ALLOWED_KEYS = base64.URLEncoding.EncodeToString(weaviateKeyBytes)
}

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
	DIFY_WEAVIATE_KEY    = "weaviate"
	DIFY_SANDBOX_KEY     = "sandbox"
)

const (
	DIFY_POSTGRES_IMAGE = "/postgres:15-alpine"
	DIFY_REDIS_IMAGE    = "/redis:6-alpine"
	DIFY_NGINX_IMAGE    = "/nginx:latest"
	DIFY_API_IMAGE      = "/langgenius/dify-api:1.7.2"
	DIFY_PLUGIN_IMAGE   = "/langgenius/dify-plugin-daemon:0.2.0-local"
	DIFY_WEB_IMAGE      = "/langgenius/dify-web:1.7.2"
	DIFY_SANDBOX_IMAGE  = "/langgenius/dify-sandbox:0.2.12"
	DIFY_SSRF_IMAGE     = "/ubuntu/squid:latest"
	DIFY_WEAVIATE_IAMGE = "/semitechnologies/weaviate:1.19.0"
)

const (
	DIFY_LOCALHOST = "localhost"
	PIP_MIRROR_URL = "https://mirrors.aliyun.com/pypi/simple"
)

const (
	DIFY_POSTGRES_PVC_MOUNT_PATH       = "/var/lib/postgresql/data"
	DIFY_POSTGRES_MAX_CONNECTIONS      = "100"
	DIFY_POSTGRES_SHARED_BUFFERS       = "128MB"
	DIFY_POSTGRES_WORK_MEM             = "4MB"
	DIFY_POSTGRES_MAINTENANCE_WORK_MEM = "64MB"
	DIFY_POSTGRES_EFFECTIVE_CACHE_SIZE = "4096MB"
	DIFY_POSTGRES_USER                 = "postgres"
	DIFY_POSTGRES_PASSWORD             = "difyai123456"
	DIFY_POSTGRES_DB                   = "dify"
	DIFY_POSTGRES_PGDATA               = "/pgdata"
	DIFY_POSTGRES_PORT                 = "5432"
)

const (
	DIFY_REDISCLI_AUTH        = "difyai123456"
	DIFY_REDIS_PVC_MOUNT_PATH = "/data"
	DIFY_REDIS_PORT           = "6379"
)

const (
	DIFY_API_PVC_MOUNT_PATH              = "/app/api/storage"
	DIFY_API_MODE                        = "api"
	DIFY_API_SENTRY_DSN                  = ""
	DIFY_API_SENTRY_TRACES_SAMPLE_RATE   = "1.0"
	DIFY_API_SENTRY_PROFILES_SAMPLE_RATE = "1.0"
)

const (
	DIFY_WORKER_MODE      = "worker"
	DIFY_WORKER_BEAT_MODE = "beat"
)

const (
	DIFY_PLUGIN_DB_DATABASE               = "dify_plugin"
	DIFY_PLUGIN_SERVER_PORT               = "5002"
	DIFY_PLUGIN_MAX_PACKAGE_SIZE          = "52428800"
	DIFY_PLUGIN_PPROF_ENABLED             = "false"
	DIFY_PLUGIN_DIFY_INNER_API_URL        = "http://localhost:5001"
	DIFY_PLUGIN_REMOTE_INSTALLING_HOST    = "0.0.0.0"
	DIFY_PLUGIN_REMOTE_INSTALLING_PORT    = "5003"
	DIFY_PLUGIN_WORKING_PATH              = "/app/storage/cwd"
	DIFY_PLUGIN_FORCE_VERIFYING_SIGNATURE = "true"
	DIFY_PLUGIN_PYTHON_ENV_INIT_TIMEOUT   = "120"
	DIFY_PLUGIN_MAX_EXECUTION_TIMEOUT     = "600"
	DIFY_PLUGIN_STORAGE_TYPE              = "local"
	DIFY_PLUGIN_STORAGE_LOCAL_ROOT        = "/app/storage"
	DIFY_PLUGIN_INSTALLED_PATH            = "plugin"
	DIFY_PLUGIN_PACKAGE_CACHE_PATH        = "plugin_packages"
	DIFY_PLUGIN_MEDIA_CACHE_PATH          = "assets"
)

const (
	DIFY_WEB_CONSOLE_API_URL                         = ""
	DIFY_WEB_APP_API_URL                             = ""
	DIFY_WEB_SENTRY_DSN                              = ""
	DIFY_WEB_NEXT_TELEMETRY_DISABLED                 = "0"
	DIFY_WEB_TEXT_GENERATION_TIMEOUT_MS              = "60000"
	DIFY_WEB_CSP_WHITELIST                           = ""
	DIFY_WEB_ALLOW_EMBED                             = "false"
	DIFY_WEB_ALLOW_UNSAFE_DATA_SCHEME                = "false"
	DIFY_WEB_MARKETPLACE_API_URL                     = "https://marketplace.dify.ai"
	DIFY_WEB_MARKETPLACE_URL                         = "https://marketplace.dify.ai"
	DIFY_WEB_TOP_K_MAX_VALUE                         = "10"
	DIFY_WEB_INDEXING_MAX_SEGMENTATION_TOKENS_LENGTH = "4000"
	DIFY_WEB_PM2_INSTANCES                           = "2"
	DIFY_WEB_LOOP_NODE_MAX_COUNT                     = "100"
	DIFY_WEB_MAX_TOOLS_NUM                           = "10"
	DIFY_WEB_MAX_PARALLEL_LIMIT                      = "10"
	DIFY_WEB_MAX_ITERATIONS_NUM                      = "99"
	DIFY_WEB_ENABLE_WEBSITE_JINAREADER               = "true"
	DIFY_WEB_ENABLE_WEBSITE_FIRECRAWL                = "true"
	DIFY_WEB_ENABLE_WEBSITE_WATERCRAWL               = "true"
)

const (
	DIFY_SSRF_MOUNT_PATH   = "/etc/squid/"
	DIFY_SSRF_HTTP_PORT    = "3128"
	DIFY_SSRF_COREDUMP_DIR = "/var/spool/squid"
)

const (
	DIFY_NGINX_MOUNT_PATH           = "/etc/nginx/conf.d"
	DIFY_NGINX_SERVER_NAME          = "_"
	DIFY_NGINX_PORT                 = "80"
	DIFY_NGINX_WORKER_PROCESSES     = "auto"
	DIFY_NGINX_CLIENT_MAX_BODY_SIZE = "100M"
	DIFY_NGINX_KEEPALIVE_TIMEOUT    = "65"
	DIFY_NGINX_PROXY_READ_TIMEOUT   = "3600s"
	DIFY_NGINX_PROXY_SEND_TIMEOUT   = "3600s"
)

const (
	DIFY_WEAVIATE_PERSISTENCE_DATA_PATH                   = "/var/lib/weaviate"
	DIFY_WEAVIATE_QUERY_DEFAULTS_LIMIT                    = "25"
	DIFY_WEAVIATE_AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED = "true"
	DIFY_WEAVIATE_DEFAULT_VECTORIZER_MODULE               = "none"
	DIFY_WEAVIATE_CLUSTER_HOSTNAME                        = "node1"
	DIFY_WEAVIATE_AUTHENTICATION_APIKEY_ENABLED           = "true"
	DIFY_WEAVIATE_AUTHENTICATION_APIKEY_USERS             = "hello@dify.ai"
	DIFY_WEAVIATE_AUTHORIZATION_ADMINLIST_ENABLED         = "true"
	DIFY_WEAVIATE_AUTHORIZATION_ADMINLIST_USERS           = "hello@dify.ai"
)

const (
	DIFY_SANDBOX_CONF_MOUNT_PATH = "/conf"
	DIFY_SANDBOX_DEP_MOUNT_PATH  = "/dependencies"
	DIFY_SANDBOX_API_KEY         = "dify-sandbox"
	DIFY_SANDBOX_GIN_MODE        = "release"
	DIFY_SANDBOX_WORKER_TIMEOUT  = "15"
	DIFY_SANDBOX_ENABLE_NETWORK  = "true"
	DIFY_SANDBOX_HTTP_PROXY      = "http://" + DIFY_LOCALHOST + ":" + DIFY_SSRF_HTTP_PORT
	DIFY_SANDBOX_HTTPS_PROXY     = "http://" + DIFY_LOCALHOST + ":" + DIFY_SSRF_HTTP_PORT
	DIFY_SANDBOX_PORT            = "8194"
)

const (
	DIFY_DEPLOY_REDIS_FAILED       = "dify_deploy_redis_failed"
	DIFY_DEPLOY_POSTGRES_FAILED    = "dify_deploy_postgres_failed"
	DIFY_DEPLOY_API_FAILED         = "dify_deploy_api_failed"
	DIFY_DEPLOY_WORKER_FAILED      = "dify_deploy_worker_failed"
	DIFY_DEPLOY_WORKER_BEAT_FAILED = "dify_deploy_worker_beat_failed"
	DIFY_DEPLOY_WEB_FAILED         = "dify_deploy_web_failed"
	DIFY_DEPLOY_PLUGIN_FAILED      = "dify_deploy_plugin_failed"
	DIFY_DEPLOY_SANDBOX_FAILED     = "dify_deploy_sandbox_failed"
	DIFY_DEPLOY_SSRF_FAILED        = "dify_deploy_ssrf_failed"
	DIFY_DEPLOY_NGINX_FAILED       = "dify_deploy_nginx_failed"
	DIFY_DEPLOY_WEAVIATE_FAILED    = "dify_deploy_weaviate_failed"
	DIFY_CREATE_FAILED             = "create_dify_failed"
	DIFY_CREATED                   = "running"
)
