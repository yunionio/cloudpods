package llm

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMRouterAgentListOptions struct {
	options.BaseListOptions

	LLMDriver string `json:"llm_driver" help:"filter by llm driver"`
}

func (o *LLMRouterAgentListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMRouterAgentShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMRouterAgentShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMRouterAgentCreateOptions struct {
	apis.SharableVirtualResourceCreateInput

	LlmId             string `help:"LLM instance ID; if set, llm_url is resolved from it" json:"llm_id"`
	LLM_URL           string `help:"decision model OpenAI-compatible base URL" json:"llm_url"`
	LLM_DRIVER        string `help:"decision model driver" json:"llm_driver" choices:"ollama|openai"`
	MODEL             string `help:"decision model name" json:"model"`
	API_KEY           string `help:"decision model API key" json:"api_key"`
	DefaultRoute      string `help:"fallback route, default complex" json:"default_route"`
	MaxPromptChars    int    `help:"request char limit; 0 auto-detects /v1/models context, longer requests use complex" json:"max_prompt_chars"`
	MaxDecisionTokens int    `help:"decision answer token hint" json:"max_decision_tokens"`
	CandidateMapping  string `help:"candidate mapping JSON, e.g. '{\"simple\":[\"qwen\"],\"complex\":[\"deepseek\"]}'" json:"-"`
	SimpleDefinition  string `help:"definition of simple requests" json:"simple_definition"`
	ComplexDefinition string `help:"definition of complex requests" json:"complex_definition"`
	SimpleExamples    string `help:"simple examples JSON array" json:"-"`
	ComplexExamples   string `help:"complex examples JSON array" json:"-"`
}

func (o *LLMRouterAgentCreateOptions) Params() (jsonutils.JSONObject, error) {
	obj := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	if o.CandidateMapping != "" {
		mapping, err := parseRouterCandidateMapping(o.CandidateMapping)
		if err != nil {
			return nil, err
		}
		obj.Set("candidate_mapping", jsonutils.Marshal(mapping))
	}
	if err := setRouterExamples(obj, "simple_examples", o.SimpleExamples); err != nil {
		return nil, err
	}
	if err := setRouterExamples(obj, "complex_examples", o.ComplexExamples); err != nil {
		return nil, err
	}
	return obj, nil
}

type LLMRouterAgentUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ID                string
	LlmId             *string `help:"LLM instance ID; if set, llm_url is resolved from it" json:"llm_id,omitempty"`
	LlmUrl            *string `help:"decision model OpenAI-compatible base URL" json:"llm_url,omitempty"`
	LlmDriver         *string `help:"decision model driver" json:"llm_driver,omitempty" choices:"ollama|openai"`
	Model             *string `help:"decision model name" json:"model,omitempty"`
	ApiKey            *string `help:"decision model API key" json:"api_key,omitempty"`
	DefaultRoute      *string `help:"fallback route" json:"default_route,omitempty"`
	MaxPromptChars    *int    `help:"request char limit; 0 recomputes from /v1/models context" json:"max_prompt_chars,omitempty"`
	MaxDecisionTokens *int    `help:"decision answer token hint" json:"max_decision_tokens,omitempty"`
	CandidateMapping  string  `help:"candidate mapping JSON" json:"-"`
	SimpleDefinition  *string `help:"definition of simple requests" json:"simple_definition,omitempty"`
	ComplexDefinition *string `help:"definition of complex requests" json:"complex_definition,omitempty"`
	SimpleExamples    string  `help:"replace simple examples with JSON array" json:"-"`
	ComplexExamples   string  `help:"replace complex examples with JSON array" json:"-"`
}

func (o *LLMRouterAgentUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMRouterAgentUpdateOptions) Params() (jsonutils.JSONObject, error) {
	obj := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	obj.Remove("id")
	if o.CandidateMapping != "" {
		mapping, err := parseRouterCandidateMapping(o.CandidateMapping)
		if err != nil {
			return nil, err
		}
		obj.Set("candidate_mapping", jsonutils.Marshal(mapping))
	}
	if err := setRouterExamples(obj, "simple_examples", o.SimpleExamples); err != nil {
		return nil, err
	}
	if err := setRouterExamples(obj, "complex_examples", o.ComplexExamples); err != nil {
		return nil, err
	}
	return obj, nil
}

type LLMRouterAgentDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMRouterAgentDeleteOptions) GetId() string {
	return o.ID
}

func (o *LLMRouterAgentDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMRouterAgentRouteOptions struct {
	ID         string   `help:"llm router agent id or name" json:"-"`
	Model      string   `help:"requested model" json:"model"`
	Message    []string `help:"user message; repeatable" json:"-"`
	Candidates []string `help:"candidate model; repeatable" json:"candidates"`
}

func (o *LLMRouterAgentRouteOptions) GetId() string {
	return o.ID
}

func (o *LLMRouterAgentRouteOptions) Params() (jsonutils.JSONObject, error) {
	messages := make([]api.LLMRouterMessage, 0, len(o.Message))
	for _, msg := range o.Message {
		messages = append(messages, api.LLMRouterMessage{Role: "user", Content: msg})
	}
	input := api.LLMRouterRouteRequest{
		Model:      o.Model,
		Messages:   messages,
		Candidates: o.Candidates,
	}
	return jsonutils.Marshal(input), nil
}

type LLMRouterAgentExpandExamplesOptions struct {
	ID                string  `help:"llm router agent id or name" json:"-"`
	SimpleDefinition  *string `help:"draft definition of simple requests" json:"simple_definition,omitempty"`
	ComplexDefinition *string `help:"draft definition of complex requests" json:"complex_definition,omitempty"`
	SimpleExamples    string  `help:"draft simple examples JSON array" json:"-"`
	ComplexExamples   string  `help:"draft complex examples JSON array" json:"-"`
}

func (o *LLMRouterAgentExpandExamplesOptions) GetId() string {
	return o.ID
}

func (o *LLMRouterAgentExpandExamplesOptions) Params() (jsonutils.JSONObject, error) {
	obj := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	obj.Remove("id")
	if err := setRouterExamples(obj, "simple_examples", o.SimpleExamples); err != nil {
		return nil, err
	}
	if err := setRouterExamples(obj, "complex_examples", o.ComplexExamples); err != nil {
		return nil, err
	}
	return obj, nil
}

func parseRouterCandidateMapping(raw string) (map[string][]string, error) {
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse candidate mapping JSON: %v", err)
	}
	mapping := map[string][]string{}
	if err := obj.Unmarshal(&mapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candidate mapping: %v", err)
	}
	return mapping, nil
}

func setRouterExamples(obj *jsonutils.JSONDict, key, raw string) error {
	if raw == "" {
		return nil
	}
	examples, err := parseRouterExamples(raw)
	if err != nil {
		return err
	}
	obj.Set(key, jsonutils.Marshal(examples))
	return nil
}

func parseRouterExamples(raw string) ([]string, error) {
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse router examples JSON: %v", err)
	}
	examples := []string{}
	if err := obj.Unmarshal(&examples); err != nil {
		return nil, fmt.Errorf("failed to unmarshal router examples: %v", err)
	}
	return examples, nil
}
