package llm

import "yunion.io/x/onecloud/pkg/apis"

const (
	LLM_ROUTER_ROUTE_SIMPLE  = "simple"
	LLM_ROUTER_ROUTE_COMPLEX = "complex"

	LLM_ROUTER_DEFAULT_ROUTE               = LLM_ROUTER_ROUTE_COMPLEX
	LLM_ROUTER_DEFAULT_MAX_PROMPT_CHARS    = LLM_DEFAULT_CONTEXT_TOKENS * 3 / 4
	LLM_ROUTER_DEFAULT_MAX_DECISION_TOKENS = 64
	LLM_ROUTER_DEFAULT_SIMPLE_DEFINITION   = "适合翻译、摘要、短问答和格式转换。"
	LLM_ROUTER_DEFAULT_COMPLEX_DEFINITION  = "适合代码、数学、长上下文、复杂推理和日志分析。"
)

type LLMRouterAgentListInput struct {
	apis.SharableVirtualResourceListInput

	LLMDriver string `json:"llm_driver"`
}

type LLMRouterAgentCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	LLMId             string              `json:"llm_id" help:"LLM instance ID; if set, llm_url is resolved from it"`
	LLMUrl            string              `json:"llm_url" help:"decision model OpenAI-compatible base URL"`
	LLMDriver         string              `json:"llm_driver" help:"decision model driver, default openai"`
	Model             string              `json:"model" help:"decision model name"`
	ApiKey            string              `json:"api_key" help:"decision model API key"`
	DefaultRoute      string              `json:"default_route" help:"fallback route, default complex"`
	MaxPromptChars    int                 `json:"max_prompt_chars" help:"request char limit; 0 auto-detects /v1/models context, longer requests use complex"`
	MaxDecisionTokens int                 `json:"max_decision_tokens" help:"decision answer token hint"`
	CandidateMapping  map[string][]string `json:"candidate_mapping" help:"route to candidate model names"`
	SimpleDefinition  string              `json:"simple_definition" help:"definition of simple requests"`
	ComplexDefinition string              `json:"complex_definition" help:"definition of complex requests"`
	SimpleExamples    []string            `json:"simple_examples" help:"examples routed to simple"`
	ComplexExamples   []string            `json:"complex_examples" help:"examples routed to complex"`
}

type LLMRouterAgentUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	LLMId             *string              `json:"llm_id,omitempty" help:"LLM instance ID; if set, llm_url is resolved from it"`
	LLMUrl            *string              `json:"llm_url,omitempty" help:"decision model OpenAI-compatible base URL"`
	LLMDriver         *string              `json:"llm_driver,omitempty" help:"decision model driver"`
	Model             *string              `json:"model,omitempty" help:"decision model name"`
	ApiKey            *string              `json:"api_key,omitempty" help:"decision model API key"`
	DefaultRoute      *string              `json:"default_route,omitempty" help:"fallback route"`
	MaxPromptChars    *int                 `json:"max_prompt_chars,omitempty" help:"request char limit; 0 recomputes from /v1/models context"`
	MaxDecisionTokens *int                 `json:"max_decision_tokens,omitempty" help:"decision answer token hint"`
	CandidateMapping  *map[string][]string `json:"candidate_mapping,omitempty" help:"route to candidate model names"`
	SimpleDefinition  *string              `json:"simple_definition,omitempty" help:"definition of simple requests"`
	ComplexDefinition *string              `json:"complex_definition,omitempty" help:"definition of complex requests"`
	SimpleExamples    *[]string            `json:"simple_examples,omitempty" help:"replace examples routed to simple"`
	ComplexExamples   *[]string            `json:"complex_examples,omitempty" help:"replace examples routed to complex"`
}

type LLMRouterAgentDetails struct {
	apis.SharableVirtualResourceDetails

	LLMId   string `json:"llm_id"`
	LLMName string `json:"llm_name"`
}

type LLMRouterPromptConfig struct {
	SimpleDefinition  string   `json:"simple_definition"`
	ComplexDefinition string   `json:"complex_definition"`
	SimpleExamples    []string `json:"simple_examples"`
	ComplexExamples   []string `json:"complex_examples"`
}

type LLMRouterExpandExamplesInput struct {
	SimpleDefinition  *string   `json:"simple_definition,omitempty"`
	ComplexDefinition *string   `json:"complex_definition,omitempty"`
	SimpleExamples    *[]string `json:"simple_examples,omitempty"`
	ComplexExamples   *[]string `json:"complex_examples,omitempty"`
}

type LLMRouterMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type LLMRouterRouteRequest struct {
	Model      string             `json:"model"`
	Messages   []LLMRouterMessage `json:"messages"`
	Candidates []string           `json:"candidates"`
}

type LLMRouterRouteResponse struct {
	Model string `json:"model"`
}

type LLMRouterDecision struct {
	Route      string  `json:"route"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
}
