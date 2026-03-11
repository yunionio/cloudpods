package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func handleOllamaRegistryYAML(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	yamlContent := models.GetInstantModelManager().GetOllamaRegistryYAML()
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	appsrv.Send(w, yamlContent)
}

func AddAvailableNetworkHandler(prefix string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/available-network", prefix), auth.Authenticate(handleLLMAvailableNetwork), nil, "get_llm_available_network", nil)
}

func handleLLMAvailableNetwork(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "Parse query string %q: %v", r.URL.RawQuery, err)
		return
	}
	ret, err := models.GetLLMManager().GetAvailableNetwork(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	wrapped := jsonutils.NewDict()
	if ret != nil {
		wrapped.Add(ret, "llm")
	}
	appsrv.SendJSON(w, wrapped)
}

func handleDefaultChatStream(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	body, err := jsonutils.Parse(bodyBytes)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "invalid body: %v", err)
		return
	}
	var input api.LLMMCPAgentRequestInput
	if body.Contains(models.GetMCPAgentManager().Keyword()) {
		agentObj, _ := body.Get(models.GetMCPAgentManager().Keyword())
		if agentObj != nil {
			body = agentObj
		}
	}
	if err := body.Unmarshal(&input); err != nil {
		httperrors.InvalidInputError(ctx, w, "invalid input: %v", err)
		return
	}
	defaultAgent, err := models.GetMCPAgentManager().GetDefaultAgent(ctx, userCred)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if defaultAgent == nil {
		httperrors.NotFoundError(ctx, w, "no default MCP agent set (set one agent with default_agent=true)")
		return
	}
	query := jsonutils.NewDict()
	_, err = defaultAgent.PerformChatStream(ctx, userCred, query, input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
}

func handleDefaultMcpTools(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	result, err := models.GetMCPAgentManager().GetDefaultMcpServerTools(ctx, userCred)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, result)
}

func InitHandlers(app *appsrv.Application, isSlave bool) {
	db.InitAllManagers()
	db.RegistUserCredCacheUpdater()

	taskman.AddTaskHandler("", app, isSlave)

	app.AddHandler("GET", "/ollama-registry.yaml", handleOllamaRegistryYAML)

	AddAvailableNetworkHandler(models.GetLLMManager().KeywordPlural(), app)

	// 默认 Agent 聊天流：优先于 dispatcher 注册，避免被 performClassAction 的 sendJSON 覆盖。
	// 注册两种路径：default-chat-stream（apigateway 转发用）与 default/chat-stream（climc 直连 region 时用，否则会被当作 resid=default 的 perform 导致 404）
	defaultChatStream := app.AddHandler2("POST", "/mcp_agents/default-chat-stream", auth.Authenticate(handleDefaultChatStream), nil, "default_chat_stream", nil)
	defaultChatStream.SetProcessTimeout(time.Hour * 4).SetWorkerManager(models.GetMCPAgentWorkerManager())
	defaultChatStreamSlash := app.AddHandler2("POST", "/mcp_agents/default/chat-stream", auth.Authenticate(handleDefaultChatStream), nil, "default_chat_stream_slash", nil)
	defaultChatStreamSlash.SetProcessTimeout(time.Hour * 4).SetWorkerManager(models.GetMCPAgentWorkerManager())

	// 默认 MCP 服务器 tools：仅使用 options.MCPServerURL，不依赖 mcp_agent 条目
	app.AddHandler2("GET", "/mcp_agents/default-mcp-tools", auth.Authenticate(handleDefaultMcpTools), nil, "default_mcp_tools", nil)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		taskman.ArchivedTaskManager,

		db.SharedResourceManager,
		db.UserCacheManager,
		db.TenantCacheManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,

		models.GetLLMImageManager(),
		models.GetLLMSkuManager(),
		// models.GetDifySkuManager(),
		models.GetVolumeManager(),
		models.GetLLMBackupManager(),
		models.GetAccessInfoManager(),
		models.GetLLMContainerManager(),
		models.GetLLMManager(),
		// models.GetDifyManager(),
		models.GetInstantModelManager(),
		models.GetLLMInstantModelManager(),
		models.GetMCPAgentManager(),
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler, isSlave)
	}
}
