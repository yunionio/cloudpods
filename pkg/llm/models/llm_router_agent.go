package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	seclib "yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetLLMRouterAgentManager()
}

var llmRouterAgentManager *SLLMRouterAgentManager

func GetLLMRouterAgentManager() *SLLMRouterAgentManager {
	if llmRouterAgentManager != nil {
		return llmRouterAgentManager
	}
	llmRouterAgentManager = &SLLMRouterAgentManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLLMRouterAgent{},
			"llm_router_agents_tbl",
			"llm_router_agent",
			"llm_router_agents",
		),
	}
	llmRouterAgentManager.SetVirtualObject(llmRouterAgentManager)
	return llmRouterAgentManager
}

type SLLMRouterAgentManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SLLMRouterAgent struct {
	db.SSharableVirtualResourceBase

	LLMId             string               `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	LLMUrl            string               `width:"512" charset:"utf8" nullable:"false" list:"user" create:"required" update:"user"`
	LLMDriver         string               `width:"64" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	Model             string               `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	ApiKey            string               `width:"512" charset:"utf8" nullable:"true" create:"optional" update:"user"`
	DefaultRoute      string               `width:"32" charset:"ascii" nullable:"false" default:"complex" list:"user" create:"optional" update:"user"`
	MaxPromptChars    int                  `nullable:"false" default:"6144" list:"user" create:"optional" update:"user"`
	MaxDecisionTokens int                  `nullable:"false" default:"64" list:"user" create:"optional" update:"user"`
	CandidateMapping  jsonutils.JSONObject `length:"long" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	SimpleDefinition  string               `length:"long" charset:"utf8" nullable:"false" list:"user" create:"optional" update:"user"`
	ComplexDefinition string               `length:"long" charset:"utf8" nullable:"false" list:"user" create:"optional" update:"user"`
	SimpleExamples    jsonutils.JSONObject `length:"long" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	ComplexExamples   jsonutils.JSONObject `length:"long" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
}

func (agent *SLLMRouterAgent) BeforeInsert() {
	if len(agent.Id) == 0 {
		agent.Id = db.DefaultUUIDGenerator()
	}
	agent.encryptApiKey()
	agent.SSharableVirtualResourceBase.BeforeInsert()
}

func (agent *SLLMRouterAgent) BeforeUpdate() {
	agent.encryptApiKey()
}

func (agent *SLLMRouterAgent) encryptApiKey() {
	if len(agent.ApiKey) == 0 {
		return
	}
	if _, err := seclib.DescryptAESBase64(agent.Id, agent.ApiKey); err == nil {
		return
	}
	sec, err := seclib.EncryptAESBase64(agent.Id, agent.ApiKey)
	if err != nil {
		log.Errorf("EncryptAESBase64 fail %s", err)
		return
	}
	agent.ApiKey = sec
}

func (agent *SLLMRouterAgent) GetApiKey() (string, error) {
	if len(agent.ApiKey) == 0 {
		return "", nil
	}
	key, err := seclib.DescryptAESBase64(agent.Id, agent.ApiKey)
	if err == nil {
		return key, nil
	}
	return "", errors.Wrap(err, "decrypt llm router api key")
}

func (man *SLLMRouterAgentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMRouterAgentCreateInput) (*api.LLMRouterAgentCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	if err := resolveRouterAgentLLM(ctx, userCred, query, &input.LLMId, &input.LLMUrl, &input.Model); err != nil {
		return input, err
	}
	input.LLMDriver = normalizeRouterLLMDriver(input.LLMDriver)
	if !api.IsLLMClientType(input.LLMDriver) {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "llm_driver must be one of: %s, got: %s", api.LLM_CLIENT_TYPES.List(), input.LLMDriver)
	}
	input.LLMUrl = strings.TrimSpace(input.LLMUrl)
	input.Model = strings.TrimSpace(input.Model)
	if input.LLMUrl == "" {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_url is required (or provide llm_id to auto-fetch)")
	}
	if input.Model == "" {
		return input, errors.Wrap(httperrors.ErrInputParameter, "model is required")
	}
	input.DefaultRoute = normalizeRouterRoute(input.DefaultRoute)
	input.MaxPromptChars, err = resolveRouterMaxPromptChars(ctx, input.MaxPromptChars, input.LLMUrl, input.Model, input.ApiKey)
	if err != nil {
		return input, err
	}
	input.MaxDecisionTokens = normalizePositiveInt(input.MaxDecisionTokens, api.LLM_ROUTER_DEFAULT_MAX_DECISION_TOKENS)
	input.CandidateMapping = normalizeCandidateMapping(input.CandidateMapping)
	promptConfig := normalizeRouterPromptConfig(api.LLMRouterPromptConfig{
		SimpleDefinition:  input.SimpleDefinition,
		ComplexDefinition: input.ComplexDefinition,
		SimpleExamples:    input.SimpleExamples,
		ComplexExamples:   input.ComplexExamples,
	})
	input.SimpleDefinition = promptConfig.SimpleDefinition
	input.ComplexDefinition = promptConfig.ComplexDefinition
	input.SimpleExamples = promptConfig.SimpleExamples
	input.ComplexExamples = promptConfig.ComplexExamples
	input.Status = api.STATUS_READY
	return input, nil
}

func (agent *SLLMRouterAgent) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMRouterAgentUpdateInput) (api.LLMRouterAgentUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = agent.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceBaseUpdateInput")
	}

	if input.LLMId != nil && strings.TrimSpace(*input.LLMId) != "" {
		llmId := strings.TrimSpace(*input.LLMId)
		llmUrl := ""
		model := ""
		if input.LLMUrl != nil {
			llmUrl = *input.LLMUrl
		}
		if input.Model != nil {
			model = *input.Model
		}
		if err := resolveRouterAgentLLM(ctx, userCred, query, &llmId, &llmUrl, &model); err != nil {
			return input, err
		}
		input.LLMId = &llmId
		input.LLMUrl = &llmUrl
		input.Model = &model
	}
	if input.LLMDriver != nil {
		driver := normalizeRouterLLMDriver(*input.LLMDriver)
		if !api.IsLLMClientType(driver) {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "llm_driver must be one of: %s, got: %s", api.LLM_CLIENT_TYPES.List(), driver)
		}
		input.LLMDriver = &driver
	}
	if input.LLMUrl != nil {
		llmUrl := strings.TrimSpace(*input.LLMUrl)
		if llmUrl == "" {
			return input, errors.Wrap(httperrors.ErrInputParameter, "llm_url cannot be empty")
		}
		input.LLMUrl = &llmUrl
	}
	if input.Model != nil {
		model := strings.TrimSpace(*input.Model)
		if model == "" {
			return input, errors.Wrap(httperrors.ErrInputParameter, "model cannot be empty")
		}
		input.Model = &model
	}
	if input.DefaultRoute != nil {
		route := normalizeRouterRoute(*input.DefaultRoute)
		input.DefaultRoute = &route
	}
	if err := agent.normalizeRouterUpdateMaxPromptChars(ctx, &input); err != nil {
		return input, err
	}
	if input.MaxDecisionTokens != nil && *input.MaxDecisionTokens <= 0 {
		val := api.LLM_ROUTER_DEFAULT_MAX_DECISION_TOKENS
		input.MaxDecisionTokens = &val
	}
	if input.CandidateMapping != nil {
		mapping := normalizeCandidateMapping(*input.CandidateMapping)
		input.CandidateMapping = &mapping
	}
	normalizeRouterPromptUpdate(&input)
	return input, nil
}

func resolveRouterAgentLLM(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, llmId *string, llmUrl *string, model *string) error {
	if llmId == nil || strings.TrimSpace(*llmId) == "" {
		return nil
	}
	llmObj, err := GetLLMManager().FetchByIdOrName(ctx, userCred, strings.TrimSpace(*llmId))
	if err != nil {
		return errors.Wrapf(err, "fetch LLM by id %s", *llmId)
	}
	llm := llmObj.(*SLLM)
	*llmId = llm.Id
	info, err := llm.GetLLMAccessUrlInfo(ctx, userCred, query)
	if err != nil {
		return errors.Wrapf(err, "get LLM URL from LLM %s", *llmId)
	}
	*llmUrl = info.LoginUrl
	if strings.TrimSpace(*model) != "" {
		return nil
	}
	mdlInfos, err := llm.getProbedInstantModelsExt(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "get probed models from LLM instance")
	}
	if len(mdlInfos) == 0 {
		return httperrors.NewBadRequestError("no available models found in LLM instance %s", *llmId)
	}
	var first api.LLMInternalInstantMdlInfo
	for _, mdlInfo := range mdlInfos {
		first = mdlInfo
		break
	}
	*model = fmt.Sprintf("%s:%s", first.Name, first.Tag)
	return nil
}

func normalizeRouterLLMDriver(driver string) string {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "" {
		return string(api.LLM_CLIENT_OPENAI)
	}
	return driver
}

func normalizeRouterRoute(route string) string {
	route = strings.ToLower(strings.TrimSpace(route))
	if route == "" {
		return api.LLM_ROUTER_DEFAULT_ROUTE
	}
	return route
}

func normalizePositiveInt(val int, def int) int {
	if val <= 0 {
		return def
	}
	return val
}

const routerModelsTimeout = 3 * time.Second

type routerModelsResponse struct {
	Data []routerModelEntry `json:"data"`
}

type routerModelEntry struct {
	ID            string `json:"id"`
	MaxModelLen   int    `json:"max_model_len"`
	ContextLength int    `json:"context_length"`
}

func buildRouterModelsURL(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.Error("llm_url is empty")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "parse llm_url")
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errors.Error("llm_url must be absolute")
	}
	u.RawQuery = ""
	u.Fragment = ""
	path := strings.TrimRight(u.Path, "/")
	switch {
	case path == "":
		u.Path = "/v1/models"
	case strings.HasSuffix(path, "/v1/models"):
		u.Path = path
	case strings.HasSuffix(path, "/v1"):
		u.Path = path + "/models"
	default:
		u.Path = path + "/v1/models"
	}
	return u.String(), nil
}

func fetchRouterContextWindow(ctx context.Context, client *http.Client, endpoint, model, apiKey string) (int, error) {
	modelsURL, err := buildRouterModelsURL(endpoint)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return 0, errors.Wrap(err, "create /v1/models request")
	}
	req.Header.Set("Accept", "application/json")
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "request /v1/models")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, errors.Errorf("/v1/models returned status %d", resp.StatusCode)
	}
	var out routerModelsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return 0, errors.Wrap(err, "decode /v1/models response")
	}
	model = strings.TrimSpace(model)
	for _, item := range out.Data {
		if strings.TrimSpace(item.ID) != model {
			continue
		}
		if item.MaxModelLen > 0 {
			return item.MaxModelLen, nil
		}
		if item.ContextLength > 0 {
			return item.ContextLength, nil
		}
		return 0, errors.Errorf("model %q has no valid context length", model)
	}
	return 0, errors.Errorf("model %q not found in /v1/models", model)
}

func routerMaxPromptChars(contextTokens int) int {
	if contextTokens <= 0 {
		contextTokens = api.LLM_DEFAULT_CONTEXT_TOKENS
	}
	return contextTokens * 3 / 4
}

func resolveRouterMaxPromptChars(ctx context.Context, requested int, endpoint, model, apiKey string) (int, error) {
	if requested < 0 {
		return 0, errors.Wrapf(httperrors.ErrInputParameter, "max_prompt_chars must be >= 0, got %d", requested)
	}
	if requested > 0 {
		return requested, nil
	}
	contextTokens, err := fetchRouterContextWindow(ctx, &http.Client{Timeout: routerModelsTimeout}, endpoint, model, apiKey)
	if err != nil {
		log.Warningf("resolve llm router context window for model %q failed: %v; using default %d", model, err, api.LLM_DEFAULT_CONTEXT_TOKENS)
		contextTokens = api.LLM_DEFAULT_CONTEXT_TOKENS
	}
	return routerMaxPromptChars(contextTokens), nil
}

func (agent *SLLMRouterAgent) normalizeRouterUpdateMaxPromptChars(ctx context.Context, input *api.LLMRouterAgentUpdateInput) error {
	if input.MaxPromptChars == nil {
		return nil
	}
	requested := *input.MaxPromptChars
	if requested != 0 {
		maxPromptChars, err := resolveRouterMaxPromptChars(ctx, requested, "", "", "")
		if err != nil {
			return err
		}
		input.MaxPromptChars = &maxPromptChars
		return nil
	}

	llmURL, model := agent.LLMUrl, agent.Model
	if input.LLMUrl != nil {
		llmURL = *input.LLMUrl
	}
	if input.Model != nil {
		model = *input.Model
	}
	apiKey := ""
	if input.ApiKey != nil {
		apiKey = *input.ApiKey
	} else {
		storedAPIKey, err := agent.GetApiKey()
		if err != nil {
			return errors.Wrap(err, "decrypt llm router api key")
		}
		apiKey = storedAPIKey
	}
	maxPromptChars, err := resolveRouterMaxPromptChars(ctx, 0, llmURL, model, apiKey)
	if err != nil {
		return err
	}
	input.MaxPromptChars = &maxPromptChars
	return nil
}

func normalizeCandidateMapping(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for route, models := range in {
		route = normalizeRouterRoute(route)
		for _, model := range models {
			model = strings.TrimSpace(model)
			if model != "" {
				out[route] = append(out[route], model)
			}
		}
	}
	return out
}

func normalizeRouterPromptConfig(cfg api.LLMRouterPromptConfig) api.LLMRouterPromptConfig {
	cfg.SimpleDefinition = normalizeRouterDefinition(cfg.SimpleDefinition, api.LLM_ROUTER_DEFAULT_SIMPLE_DEFINITION)
	cfg.ComplexDefinition = normalizeRouterDefinition(cfg.ComplexDefinition, api.LLM_ROUTER_DEFAULT_COMPLEX_DEFINITION)
	cfg.SimpleExamples = normalizeRouterExamples(cfg.SimpleExamples)
	cfg.ComplexExamples = normalizeRouterExamples(cfg.ComplexExamples)
	return cfg
}

func normalizeRouterDefinition(definition, fallback string) string {
	definition = strings.TrimSpace(definition)
	if definition == "" {
		return fallback
	}
	return definition
}

func normalizeRouterExamples(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, example := range in {
		example = strings.TrimSpace(example)
		if example == "" {
			continue
		}
		if _, ok := seen[example]; ok {
			continue
		}
		seen[example] = struct{}{}
		out = append(out, example)
	}
	return out
}

func normalizeRouterPromptUpdate(input *api.LLMRouterAgentUpdateInput) {
	if input.SimpleDefinition != nil {
		definition := normalizeRouterDefinition(*input.SimpleDefinition, api.LLM_ROUTER_DEFAULT_SIMPLE_DEFINITION)
		input.SimpleDefinition = &definition
	}
	if input.ComplexDefinition != nil {
		definition := normalizeRouterDefinition(*input.ComplexDefinition, api.LLM_ROUTER_DEFAULT_COMPLEX_DEFINITION)
		input.ComplexDefinition = &definition
	}
	if input.SimpleExamples != nil {
		examples := normalizeRouterExamples(*input.SimpleExamples)
		input.SimpleExamples = &examples
	}
	if input.ComplexExamples != nil {
		examples := normalizeRouterExamples(*input.ComplexExamples)
		input.ComplexExamples = &examples
	}
}

func (man *SLLMRouterAgentManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.LLMRouterAgentListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	if len(input.LLMDriver) > 0 {
		q = q.Equals("llm_driver", normalizeRouterLLMDriver(input.LLMDriver))
	}
	return q, nil
}

func (manager *SLLMRouterAgentManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.LLMRouterAgentDetails {
	rows := make([]api.LLMRouterAgentDetails, len(objs))
	vrows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	agents := []SLLMRouterAgent{}
	jsonutils.Update(&agents, objs)

	llmIds := make([]string, 0)
	for i := range agents {
		if agents[i].LLMId != "" {
			llmIds = append(llmIds, agents[i].LLMId)
		}
	}
	var llmIdNameMap map[string]string
	if len(llmIds) > 0 {
		var err error
		llmIdNameMap, err = db.FetchIdNameMap2(GetLLMManager(), llmIds)
		if err != nil {
			log.Errorf("FetchIdNameMap2 for LLMs failed: %v", err)
		}
	}
	for i := range rows {
		rows[i].SharableVirtualResourceDetails = vrows[i]
		if i < len(agents) {
			rows[i].LLMId = agents[i].LLMId
			if name, ok := llmIdNameMap[agents[i].LLMId]; ok {
				rows[i].LLMName = name
			}
		}
	}
	return rows
}

func (agent *SLLMRouterAgent) PerformRoute(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMRouterRouteRequest) (jsonutils.JSONObject, error) {
	out, err := agent.Route(ctx, input)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(out), nil
}

func (agent *SLLMRouterAgent) PerformExpandExamples(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMRouterExpandExamplesInput) (jsonutils.JSONObject, error) {
	cfg := agent.promptConfig()
	applyRouterPromptDraft(&cfg, input)
	cfg = normalizeRouterPromptConfig(cfg)
	content, err := callRouterAgentModel(ctx, agent, routerExpandSystemPrompt, buildRouterExpandPrompt(cfg))
	if err != nil {
		return nil, errors.Wrap(err, "expand router examples")
	}
	out, err := parseRouterPromptConfig(content)
	if err != nil {
		return nil, errors.Wrap(err, "parse expanded router examples")
	}
	return jsonutils.Marshal(out), nil
}

func (agent *SLLMRouterAgent) Route(ctx context.Context, req api.LLMRouterRouteRequest) (*api.LLMRouterRouteResponse, error) {
	if len(req.Candidates) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "candidates is required")
	}
	route, ok := agent.routeByRules(req)
	if !ok {
		route = agent.routeByDecisionModel(ctx, req)
	}
	if route == "" {
		route = normalizeRouterRoute(agent.DefaultRoute)
	}
	model := agent.pickCandidate(route, req.Candidates)
	if model == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no selectable candidate model")
	}
	return &api.LLMRouterRouteResponse{Model: model}, nil
}

func (agent *SLLMRouterAgent) routeByRules(req api.LLMRouterRouteRequest) (string, bool) {
	prompt := agent.requestText(req)
	maxChars := normalizePositiveInt(agent.MaxPromptChars, api.LLM_ROUTER_DEFAULT_MAX_PROMPT_CHARS)
	if len([]rune(prompt)) > maxChars {
		return api.LLM_ROUTER_ROUTE_COMPLEX, true
	}
	return "", false
}

const (
	routerDecisionSystemPrompt = "你是模型分流器。根据给定定义和样例把请求分类为 simple 或 complex。只能输出一个 JSON 对象，不得输出 Markdown、代码块或额外解释。"
	routerExpandSystemPrompt   = "你是模型分流规则编辑器。规范 simple 和 complex 的定义并补充代表性样例。只能输出指定的 JSON 对象，不得输出 Markdown、代码块或额外解释。"
)

var callRouterAgentModel = func(ctx context.Context, agent *SLLMRouterAgent, systemPrompt, userPrompt string) (string, error) {
	if strings.TrimSpace(agent.LLMUrl) == "" || strings.TrimSpace(agent.Model) == "" {
		return "", errors.Wrap(httperrors.ErrInvalidStatus, "router model URL and model are required")
	}
	driver, err := GetLLMClientDriverWithError(api.LLMClientType(normalizeRouterLLMDriver(agent.LLMDriver)))
	if err != nil {
		return "", err
	}
	apiKey, err := agent.GetApiKey()
	if err != nil {
		return "", err
	}
	tmp := &SMCPAgent{
		LLMUrl:    agent.LLMUrl,
		LLMDriver: normalizeRouterLLMDriver(agent.LLMDriver),
		Model:     agent.Model,
		ApiKey:    apiKey,
	}
	timeout := time.Duration(options.Options.MCPAgentTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := driver.Chat(callCtx, tmp, []ILLMChatMessage{
		driver.NewSystemMessage(systemPrompt),
		driver.NewUserMessage(userPrompt),
	}, nil)
	if err != nil {
		return "", err
	}
	return resp.GetContent(), nil
}

func (agent *SLLMRouterAgent) routeByDecisionModel(ctx context.Context, req api.LLMRouterRouteRequest) string {
	prompt := agent.BuildDecisionPrompt(req)
	content, err := callRouterAgentModel(ctx, agent, routerDecisionSystemPrompt, prompt)
	if err != nil {
		log.Warningf("llm router decision model failed: %v", err)
		return normalizeRouterRoute(agent.DefaultRoute)
	}
	decision, err := parseRouterDecision(content)
	if err != nil || decision.Confidence < 0.5 {
		if err != nil {
			log.Warningf("parse llm router decision failed: %v", err)
		}
		return normalizeRouterRoute(agent.DefaultRoute)
	}
	return normalizeRouterRoute(decision.Route)
}

func (agent *SLLMRouterAgent) BuildDecisionPrompt(req api.LLMRouterRouteRequest) string {
	promptConfig := agent.promptConfig()
	maxTokens := normalizePositiveInt(agent.MaxDecisionTokens, api.LLM_ROUTER_DEFAULT_MAX_DECISION_TOKENS)
	return fmt.Sprintf(`根据以下定义和样例选择 route。

simple 定义：
%s

complex 定义：
%s

simple 样例：
%s

complex 样例：
%s

候选模型：%s
最多输出约 %d tokens。

只返回 JSON，不要解释：
{"route":"simple or complex","confidence":0.0,"reason":"short reason"}

请求内容：
%s`, promptConfig.SimpleDefinition, promptConfig.ComplexDefinition, formatRouterExamples(promptConfig.SimpleExamples), formatRouterExamples(promptConfig.ComplexExamples), strings.Join(req.Candidates, ", "), maxTokens, agent.requestText(req))
}

func buildRouterExpandPrompt(cfg api.LLMRouterPromptConfig) string {
	return fmt.Sprintf(`请规范以下二分类定义和样例，并补充少量有代表性的样例。保持 simple 和 complex 边界互斥且清晰，不要增加其他 route。

simple 定义：
%s

complex 定义：
%s

simple 样例：
%s

complex 样例：
%s

只返回：
{"simple_definition":"...","complex_definition":"...","simple_examples":["..."],"complex_examples":["..."]}`,
		cfg.SimpleDefinition,
		cfg.ComplexDefinition,
		formatRouterExamples(cfg.SimpleExamples),
		formatRouterExamples(cfg.ComplexExamples),
	)
}

func applyRouterPromptDraft(cfg *api.LLMRouterPromptConfig, input api.LLMRouterExpandExamplesInput) {
	if input.SimpleDefinition != nil {
		cfg.SimpleDefinition = *input.SimpleDefinition
	}
	if input.ComplexDefinition != nil {
		cfg.ComplexDefinition = *input.ComplexDefinition
	}
	if input.SimpleExamples != nil {
		cfg.SimpleExamples = *input.SimpleExamples
	}
	if input.ComplexExamples != nil {
		cfg.ComplexExamples = *input.ComplexExamples
	}
}

func (agent *SLLMRouterAgent) promptConfig() api.LLMRouterPromptConfig {
	cfg := api.LLMRouterPromptConfig{
		SimpleDefinition:  agent.SimpleDefinition,
		ComplexDefinition: agent.ComplexDefinition,
	}
	if agent.SimpleExamples != nil {
		if err := agent.SimpleExamples.Unmarshal(&cfg.SimpleExamples); err != nil {
			log.Warningf("invalid simple_examples for llm router agent %s: %v", agent.Id, err)
		}
	}
	if agent.ComplexExamples != nil {
		if err := agent.ComplexExamples.Unmarshal(&cfg.ComplexExamples); err != nil {
			log.Warningf("invalid complex_examples for llm router agent %s: %v", agent.Id, err)
		}
	}
	return normalizeRouterPromptConfig(cfg)
}

func formatRouterExamples(examples []string) string {
	if len(examples) == 0 {
		return "无"
	}
	return "- " + strings.Join(examples, "\n- ")
}

func (agent *SLLMRouterAgent) requestText(req api.LLMRouterRouteRequest) string {
	parts := make([]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch val := msg.Content.(type) {
		case string:
			parts = append(parts, val)
		default:
			parts = append(parts, jsonutils.Marshal(val).String())
		}
	}
	return strings.Join(parts, "\n")
}

func (agent *SLLMRouterAgent) candidateMapping() map[string][]string {
	ret := map[string][]string{}
	if agent.CandidateMapping == nil {
		return ret
	}
	if err := agent.CandidateMapping.Unmarshal(&ret); err != nil {
		log.Warningf("invalid llm router candidate_mapping: %v", err)
		return map[string][]string{}
	}
	return normalizeCandidateMapping(ret)
}

func (agent *SLLMRouterAgent) pickCandidate(route string, candidates []string) string {
	mapping := agent.candidateMapping()
	if model := pickCandidateFromList(mapping[normalizeRouterRoute(route)], candidates); model != "" {
		return model
	}
	def := normalizeRouterRoute(agent.DefaultRoute)
	if def != normalizeRouterRoute(route) {
		if model := pickCandidateFromList(mapping[def], candidates); model != "" {
			return model
		}
	}
	if model := pickCandidateFromList([]string{route}, candidates); model != "" {
		return model
	}
	return strings.TrimSpace(candidates[0])
}

func pickCandidateFromList(models []string, candidates []string) string {
	for _, model := range models {
		model = strings.TrimSpace(model)
		for _, candidate := range candidates {
			if strings.EqualFold(strings.TrimSpace(candidate), model) {
				return strings.TrimSpace(candidate)
			}
		}
	}
	return ""
}

func parseRouterDecision(content string) (*api.LLMRouterDecision, error) {
	content = extractRouterJSON(content)
	decision := api.LLMRouterDecision{}
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return nil, err
	}
	decision.Route = normalizeRouterRoute(decision.Route)
	if decision.Route == "" {
		return nil, errors.Error("missing route")
	}
	if decision.Confidence == 0 {
		decision.Confidence = 1
	}
	return &decision, nil
}

func parseRouterPromptConfig(content string) (*api.LLMRouterPromptConfig, error) {
	var raw struct {
		SimpleDefinition  *string   `json:"simple_definition"`
		ComplexDefinition *string   `json:"complex_definition"`
		SimpleExamples    *[]string `json:"simple_examples"`
		ComplexExamples   *[]string `json:"complex_examples"`
	}
	if err := json.Unmarshal([]byte(extractRouterJSON(content)), &raw); err != nil {
		return nil, err
	}
	if raw.SimpleDefinition == nil || raw.ComplexDefinition == nil || raw.SimpleExamples == nil || raw.ComplexExamples == nil {
		return nil, errors.Error("router expansion response missing required fields")
	}
	cfg := normalizeRouterPromptConfig(api.LLMRouterPromptConfig{
		SimpleDefinition:  *raw.SimpleDefinition,
		ComplexDefinition: *raw.ComplexDefinition,
		SimpleExamples:    *raw.SimpleExamples,
		ComplexExamples:   *raw.ComplexExamples,
	})
	return &cfg, nil
}

func extractRouterJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end > start {
			content = content[start : end+1]
		}
	}
	return content
}
