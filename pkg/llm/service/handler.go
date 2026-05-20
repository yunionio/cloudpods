package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func handleOllamaRegistryYAML(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	yamlContent := models.GetInstantModelManager().GetOllamaRegistryYAML()
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	appsrv.Send(w, yamlContent)
}

// handleLLMModelSetList: GET /llm_model_sets
func handleLLMModelSetList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "Parse query string %q: %v", r.URL.RawQuery, err)
		return
	}
	input := api.LLMModelSetListInput{}
	if query != nil {
		_ = query.Unmarshal(&input)
	}
	sets, total := models.GetLLMModelSetManager().ListSets(input)
	if sets == nil {
		sets = []api.LLMModelSet{}
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	// Build the envelope as a JSONDict — jsonutils.Marshal omits zero-length
	// slices, so an explicit dict guarantees `llm_model_sets` is always present
	// (the mcclient module's list parser fails with "key not found" otherwise).
	resp := jsonutils.NewDict()
	resp.Set("llm_model_sets", jsonutils.Marshal(sets))
	resp.Set("total", jsonutils.NewInt(int64(total)))
	resp.Set("limit", jsonutils.NewInt(int64(limit)))
	resp.Set("offset", jsonutils.NewInt(int64(input.Offset)))
	appsrv.SendJSON(w, resp)
}

// handleLLMModelSetShow: GET /llm_model_sets/<id>
func handleLLMModelSetShow(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id := lastPathSegment(r.URL.Path)
	set, ok := models.GetLLMModelSetManager().GetSet(id)
	if !ok {
		httperrors.NotFoundError(ctx, w, "model set %s not found", id)
		return
	}
	appsrv.SendStruct(w, api.LLMModelSetShowOutput{LLMModelSet: *set})
}

// handleLLMModelSetSpecs: GET /llm_model_sets/<id>/specs
func handleLLMModelSetSpecs(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// path is .../llm_model_sets/<id>/specs — id is the second-to-last segment.
	segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segs) < 2 {
		httperrors.InvalidInputError(ctx, w, "missing model set id")
		return
	}
	id := segs[len(segs)-2]
	specs, ok := models.GetLLMModelSetManager().ListSpecs(id)
	if !ok {
		httperrors.NotFoundError(ctx, w, "model set %s not found", id)
		return
	}
	if specs == nil {
		specs = []api.LLMModelSpec{}
	}
	resp := jsonutils.NewDict()
	resp.Set("llm_model_specs", jsonutils.Marshal(specs))
	resp.Set("total", jsonutils.NewInt(int64(len(specs))))
	appsrv.SendJSON(w, resp)
}

// handleLLMModelSpecShow: GET /llm_model_specs/<id> — convenience for the
// deployment-create page so it doesn't need to know which set the spec belongs to.
func handleLLMModelSpecShow(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id := lastPathSegment(r.URL.Path)
	spec, _, ok := models.GetLLMModelSetManager().GetSpec(id)
	if !ok {
		httperrors.NotFoundError(ctx, w, "model spec %s not found", id)
		return
	}
	appsrv.SendStruct(w, api.LLMModelSpecShowOutput{LLMModelSpec: *spec})
}

// handleLLMModelSetRefresh: POST /llm_model_sets/refresh — admin only.
// After a successful refresh, returns the freshly loaded catalog in the same
// envelope shape as the list endpoint so the dashboard can re-render in one
// round trip.
func handleLLMModelSetRefresh(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return
	}
	if !userCred.HasSystemAdminPrivilege() {
		httperrors.ForbiddenError(ctx, w, "system admin required")
		return
	}
	if err := models.GetLLMModelSetManager().Refresh(ctx); err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sets, total := models.GetLLMModelSetManager().ListSets(api.LLMModelSetListInput{})
	if sets == nil {
		sets = []api.LLMModelSet{}
	}
	resp := jsonutils.NewDict()
	resp.Set("llm_model_sets", jsonutils.Marshal(sets))
	resp.Set("total", jsonutils.NewInt(int64(total)))
	appsrv.SendJSON(w, resp)
}

func lastPathSegment(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
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

func handleLLMProviderModels(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
	ret, err := models.GetLLMManager().GetProviderModels(ctx, userCred, query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendStruct(w, ret)
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

	app_common.ExportOptionsHandler(app, &options.Options)

	taskman.AddTaskHandler("", app, isSlave)

	app.AddHandler("GET", "/ollama-registry.yaml", handleOllamaRegistryYAML)

	// Catalog endpoints — backed by an in-memory store fed by an external YAML.
	// Two-level structure matches GPUStack: sets (browsable cards) + specs
	// (deployable variants under one set).
	app.AddHandler2("GET", "/llm_model_sets", auth.Authenticate(handleLLMModelSetList), nil, "llm_model_set_list", nil)
	app.AddHandler2("POST", "/llm_model_sets/refresh", auth.Authenticate(handleLLMModelSetRefresh), nil, "llm_model_set_refresh", nil)
	app.AddHandler2("GET", "/llm_model_sets/<id>/specs", auth.Authenticate(handleLLMModelSetSpecs), nil, "llm_model_set_specs", nil)
	app.AddHandler2("GET", "/llm_model_sets/<id>", auth.Authenticate(handleLLMModelSetShow), nil, "llm_model_set_show", nil)
	app.AddHandler2("GET", "/llm_model_specs/<id>", auth.Authenticate(handleLLMModelSpecShow), nil, "llm_model_spec_show", nil)

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
		models.GetLLMDeploymentManager(),
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
