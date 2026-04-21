package llm

const (
	LLM_OPENCLAW_GATEWAY_TOKEN      = LLMEnvKey("OPENCLAW_GATEWAY_TOKEN")
	LLM_OPENCLAW_CUSTOM_CONFIG      = LLMEnvKey("OPENCLAW_CUSTOM_CONFIG")
	LLM_OPENCLAW_CUSTOM_CONFIG_FILE = "/opt/openclaw_base_config.json"

	// templates
	LLM_OPENCLAW_TEMPLATE_AGENTS_MD_B64 = LLMEnvKey("OPENCLAW_TEMPLATE_AGENTS_MD_B64")
	LLM_OPENCLAW_TEMPLATE_SOUL_MD_B64   = LLMEnvKey("OPENCLAW_TEMPLATE_SOUL_MD_B64")
	LLM_OPENCLAW_TEMPLATE_USER_MD_B64   = LLMEnvKey("OPENCLAW_TEMPLATE_USER_MD_B64")
)

const (
	LLM_DESKTOP_DEFAULT_PORT  = 3001
	LLM_DESKTOP_AUTH_USERNAME = LLMEnvKey("AUTH_USERNAME")
	LLM_DESKTOP_CUSTOM_USER   = LLMEnvKey("CUSTOM_USER")
	LLM_DESKTOP_AUTH_PASSWORD = LLMEnvKey("AUTH_PASSWORD")
	LLM_DESKTOP_PASSWORD      = LLMEnvKey("PASSWORD")
)

type OpenClawConfig struct {
	Browser *OpenClawConfigBrowser `json:"browser"`
	Agents  *OpenClawConfigAgents  `json:"agents"`
}

type OpenClawConfigAgents map[string]*OpenClawConfigAgent

type OpenClawConfigAgent struct {
	Model      *OpenClawConfigAgentModel `json:"model"`
	ImageModel *OpenClawConfigAgentModel `json:"imageModel"`
	Workspace  string                    `json:"workspace"`
}

type OpenClawConfigAgentModel struct {
	Primary string `json:"primary"`
}

// ref: https://docs.openclaw.ai/tools/browser#configuration
type OpenClawConfigBrowser struct {
	Enabled        bool                   `json:"enabled"`
	SSRFPolicy     map[string]interface{} `json:"ssrfPolicy"`
	DefaultProfile string                 `json:"defaultProfile"`
	Headless       bool                   `json:"headless"`
	NoSandbox      bool                   `json:"noSandbox"`
}
