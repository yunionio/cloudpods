// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

// catalogSeedModel is one row to insert into ai_models when seeding a standard provider.
// ModelKey is the id sent to the upstream API (no "provider/" prefix).
type catalogSeedModel struct {
	ModelKey    string
	Description string
}

// catalogSeedModelsForProvider returns known public model ids for seeding.
// Curated from vendor/provider docs; extend as products ship.
// Providers without a list return nil and the seeder inserts model_key "default".
func catalogSeedModelsForProvider(providerKey string) []catalogSeedModel {
	switch providerKey {
	case "anthropic":
		return []catalogSeedModel{
			{ModelKey: "claude-opus-4-20250514", Description: "Anthropic Claude Opus 4"},
			{ModelKey: "claude-sonnet-4-20250514", Description: "Anthropic Claude Sonnet 4"},
			{ModelKey: "claude-3-7-sonnet-20250219", Description: "Anthropic Claude 3.7 Sonnet"},
			{ModelKey: "claude-3-5-sonnet-20241022", Description: "Anthropic Claude 3.5 Sonnet"},
			{ModelKey: "claude-3-5-haiku-20241022", Description: "Anthropic Claude 3.5 Haiku"},
			{ModelKey: "claude-3-opus-20240229", Description: "Anthropic Claude 3 Opus"},
			{ModelKey: "claude-3-haiku-20240307", Description: "Anthropic Claude 3 Haiku"},
		}
	case "azure":
		// Azure OpenAI uses deployment names; these match common Azure OpenAI deployment ids.
		return []catalogSeedModel{
			{ModelKey: "gpt-4o", Description: "Azure OpenAI GPT-4o deployment"},
			{ModelKey: "gpt-4o-mini", Description: "Azure OpenAI GPT-4o mini deployment"},
			{ModelKey: "gpt-4", Description: "Azure OpenAI GPT-4 deployment"},
			{ModelKey: "gpt-35-turbo", Description: "Azure OpenAI GPT-3.5 Turbo deployment"},
			{ModelKey: "o3-mini", Description: "Azure OpenAI o3-mini deployment"},
		}
	case "bedrock":
		return []catalogSeedModel{
			{ModelKey: "anthropic.claude-3-5-sonnet-20241022-v2:0", Description: "Bedrock Claude 3.5 Sonnet"},
			{ModelKey: "anthropic.claude-3-5-haiku-20241022-v1:0", Description: "Bedrock Claude 3.5 Haiku"},
			{ModelKey: "anthropic.claude-3-opus-20240229-v1:0", Description: "Bedrock Claude 3 Opus"},
			{ModelKey: "anthropic.claude-3-sonnet-20240229-v1:0", Description: "Bedrock Claude 3 Sonnet"},
			{ModelKey: "anthropic.claude-3-haiku-20240307-v1:0", Description: "Bedrock Claude 3 Haiku"},
			{ModelKey: "meta.llama3-70b-instruct-v1:0", Description: "Bedrock Llama 3 70B Instruct"},
			{ModelKey: "meta.llama3-8b-instruct-v1:0", Description: "Bedrock Llama 3 8B Instruct"},
			{ModelKey: "mistral.mistral-large-2402-v1:0", Description: "Bedrock Mistral Large"},
			{ModelKey: "amazon.titan-text-express-v1", Description: "Bedrock Amazon Titan Text Express"},
		}
	case "cerebras":
		return []catalogSeedModel{
			{ModelKey: "llama3.1-8b", Description: "Cerebras Llama 3.1 8B"},
			{ModelKey: "llama3.1-70b", Description: "Cerebras Llama 3.1 70B"},
			{ModelKey: "llama-3.3-70b", Description: "Cerebras Llama 3.3 70B"},
		}
	case "cohere":
		return []catalogSeedModel{
			{ModelKey: "command-r-plus", Description: "Cohere Command R+"},
			{ModelKey: "command-r", Description: "Cohere Command R"},
			{ModelKey: "command-a", Description: "Cohere Command A"},
			{ModelKey: "command", Description: "Cohere Command"},
			{ModelKey: "command-light", Description: "Cohere Command Light"},
			{ModelKey: "embed-english-v3.0", Description: "Cohere Embed English v3"},
			{ModelKey: "embed-multilingual-v3.0", Description: "Cohere Embed Multilingual v3"},
		}
	case "elevenlabs":
		return []catalogSeedModel{
			{ModelKey: "eleven_multilingual_v2", Description: "ElevenLabs multilingual v2"},
			{ModelKey: "eleven_turbo_v2_5", Description: "ElevenLabs Turbo v2.5"},
			{ModelKey: "eleven_flash_v2_5", Description: "ElevenLabs Flash v2.5"},
			{ModelKey: "eleven_multilingual_v1", Description: "ElevenLabs multilingual v1"},
		}
	case "fireworks":
		return []catalogSeedModel{
			{ModelKey: "accounts/fireworks/models/llama-v3p1-8b-instruct", Description: "Fireworks Llama 3.1 8B Instruct"},
			{ModelKey: "accounts/fireworks/models/llama-v3p1-70b-instruct", Description: "Fireworks Llama 3.1 70B Instruct"},
			{ModelKey: "accounts/fireworks/models/llama-v3p3-70b-instruct", Description: "Fireworks Llama 3.3 70B Instruct"},
			{ModelKey: "accounts/fireworks/models/mixtral-8x7b-instruct", Description: "Fireworks Mixtral 8x7B Instruct"},
		}
	case "gemini":
		return []catalogSeedModel{
			{ModelKey: "gemini-2.0-flash", Description: "Google Gemini 2.0 Flash"},
			{ModelKey: "gemini-2.0-flash-lite", Description: "Google Gemini 2.0 Flash-Lite"},
			{ModelKey: "gemini-1.5-pro", Description: "Google Gemini 1.5 Pro"},
			{ModelKey: "gemini-1.5-flash", Description: "Google Gemini 1.5 Flash"},
			{ModelKey: "gemini-1.5-flash-8b", Description: "Google Gemini 1.5 Flash 8B"},
			{ModelKey: "gemini-embedding-001", Description: "Google Gemini Embedding 001"},
		}
	case "groq":
		return []catalogSeedModel{
			{ModelKey: "llama-3.3-70b-versatile", Description: "Groq Llama 3.3 70B Versatile"},
			{ModelKey: "llama-3.1-8b-instant", Description: "Groq Llama 3.1 8B Instant"},
			{ModelKey: "llama-3.1-70b-versatile", Description: "Groq Llama 3.1 70B Versatile"},
			{ModelKey: "mixtral-8x7b-32768", Description: "Groq Mixtral 8x7B"},
			{ModelKey: "gemma2-9b-it", Description: "Groq Gemma2 9B IT"},
		}
	case "huggingface":
		return []catalogSeedModel{
			{ModelKey: "meta-llama/Meta-Llama-3.1-8B-Instruct", Description: "HF Llama 3.1 8B Instruct"},
			{ModelKey: "meta-llama/Meta-Llama-3.1-70B-Instruct", Description: "HF Llama 3.1 70B Instruct"},
			{ModelKey: "mistralai/Mistral-7B-Instruct-v0.3", Description: "HF Mistral 7B Instruct"},
			{ModelKey: "Qwen/Qwen2.5-72B-Instruct", Description: "HF Qwen2.5 72B Instruct"},
		}
	case "mistral":
		return []catalogSeedModel{
			{ModelKey: "mistral-large-latest", Description: "Mistral Large (latest)"},
			{ModelKey: "mistral-small-latest", Description: "Mistral Small (latest)"},
			{ModelKey: "pixtral-12b-2409", Description: "Mistral Pixtral 12B"},
			{ModelKey: "codestral-latest", Description: "Mistral Codestral (latest)"},
			{ModelKey: "ministral-8b-latest", Description: "Mistral Ministral 8B"},
			{ModelKey: "open-mistral-nemo", Description: "Mistral Open Mistral Nemo"},
			{ModelKey: "mixtral-8x22b", Description: "Mistral Mixtral 8x22B"},
			{ModelKey: "mixtral-8x7b", Description: "Mistral Mixtral 8x7B"},
		}
	case "nebius":
		return []catalogSeedModel{
			{ModelKey: "deepseek-ai/DeepSeek-V3", Description: "Nebius DeepSeek V3"},
			{ModelKey: "Qwen/Qwen2.5-72B-Instruct", Description: "Nebius Qwen2.5 72B Instruct"},
			{ModelKey: "meta-llama/Llama-3.3-70B-Instruct", Description: "Nebius Llama 3.3 70B Instruct"},
		}
	case "ollama":
		return []catalogSeedModel{
			{ModelKey: "llama3.2", Description: "Ollama Llama 3.2"},
			{ModelKey: "llama3.1", Description: "Ollama Llama 3.1"},
			{ModelKey: "mistral", Description: "Ollama Mistral"},
			{ModelKey: "qwen2.5", Description: "Ollama Qwen 2.5"},
			{ModelKey: "codellama", Description: "Ollama Code Llama"},
			{ModelKey: "phi3", Description: "Ollama Phi 3"},
		}
	case "vllm":
		return []catalogSeedModel{
			{ModelKey: "meta-llama/Meta-Llama-3.1-8B-Instruct", Description: "vLLM Llama 3.1 8B Instruct"},
			{ModelKey: "meta-llama/Meta-Llama-3.1-70B-Instruct", Description: "vLLM Llama 3.1 70B Instruct"},
			{ModelKey: "Qwen/Qwen2.5-7B-Instruct", Description: "vLLM Qwen2.5 7B Instruct"},
			{ModelKey: "Qwen/Qwen2.5-72B-Instruct", Description: "vLLM Qwen2.5 72B Instruct"},
			{ModelKey: "mistralai/Mistral-7B-Instruct-v0.3", Description: "vLLM Mistral 7B Instruct"},
		}
	case "openai":
		return []catalogSeedModel{
			{ModelKey: "gpt-5-nano", Description: "OpenAI GPT-5 nano"},
			{ModelKey: "gpt-5-mini", Description: "OpenAI GPT-5 mini"},
			{ModelKey: "gpt-5", Description: "OpenAI GPT-5"},
			{ModelKey: "gpt-5.1", Description: "OpenAI GPT-5.1"},
			{ModelKey: "gpt-5.1-mini", Description: "OpenAI GPT-5.1 mini"},
			{ModelKey: "gpt-5.1-codex", Description: "OpenAI GPT-5.1 Codex"},
			{ModelKey: "gpt-5.1-codex-max", Description: "OpenAI GPT-5.1 Codex Max"},
			{ModelKey: "gpt-5.2", Description: "OpenAI GPT-5.2"},
			{ModelKey: "gpt-5.2-pro", Description: "OpenAI GPT-5.2 pro"},
			{ModelKey: "gpt-5.2-codex", Description: "OpenAI GPT-5.2 Codex"},
			{ModelKey: "gpt-4.1", Description: "OpenAI GPT-4.1"},
			{ModelKey: "gpt-4.1-mini", Description: "OpenAI GPT-4.1 mini"},
			{ModelKey: "gpt-4.1-nano", Description: "OpenAI GPT-4.1 nano"},
			{ModelKey: "gpt-4o", Description: "OpenAI GPT-4o"},
			{ModelKey: "gpt-4o-mini", Description: "OpenAI GPT-4o mini"},
			{ModelKey: "chatgpt-4o-latest", Description: "OpenAI ChatGPT-4o latest"},
			{ModelKey: "gpt-4-turbo", Description: "OpenAI GPT-4 Turbo"},
			{ModelKey: "gpt-4", Description: "OpenAI GPT-4"},
			{ModelKey: "gpt-3.5-turbo", Description: "OpenAI GPT-3.5 Turbo"},
			{ModelKey: "o1", Description: "OpenAI o1"},
			{ModelKey: "o1-mini", Description: "OpenAI o1-mini"},
			{ModelKey: "o1-preview", Description: "OpenAI o1-preview"},
			{ModelKey: "o3", Description: "OpenAI o3"},
			{ModelKey: "o3-mini", Description: "OpenAI o3-mini"},
			{ModelKey: "o4-mini", Description: "OpenAI o4-mini"},
			{ModelKey: "text-embedding-3-small", Description: "OpenAI text-embedding-3-small"},
			{ModelKey: "text-embedding-3-large", Description: "OpenAI text-embedding-3-large"},
			{ModelKey: "text-embedding-ada-002", Description: "OpenAI text-embedding-ada-002"},
		}
	case "openrouter":
		return []catalogSeedModel{
			{ModelKey: "openai/gpt-4o", Description: "OpenRouter OpenAI GPT-4o"},
			{ModelKey: "openai/gpt-4o-mini", Description: "OpenRouter OpenAI GPT-4o mini"},
			{ModelKey: "anthropic/claude-3.5-sonnet", Description: "OpenRouter Claude 3.5 Sonnet"},
			{ModelKey: "anthropic/claude-3.5-haiku", Description: "OpenRouter Claude 3.5 Haiku"},
			{ModelKey: "google/gemini-2.0-flash-001", Description: "OpenRouter Gemini 2.0 Flash"},
			{ModelKey: "meta-llama/llama-3.3-70b-instruct", Description: "OpenRouter Llama 3.3 70B Instruct"},
			{ModelKey: "mistralai/mistral-large", Description: "OpenRouter Mistral Large"},
		}
	case "perplexity":
		return []catalogSeedModel{
			{ModelKey: "sonar", Description: "Perplexity Sonar"},
			{ModelKey: "sonar-pro", Description: "Perplexity Sonar Pro"},
			{ModelKey: "sonar-reasoning", Description: "Perplexity Sonar Reasoning"},
			{ModelKey: "llama-3.1-sonar-small-128k-online", Description: "Perplexity Llama 3.1 Sonar Small online"},
			{ModelKey: "llama-3.1-sonar-large-128k-online", Description: "Perplexity Llama 3.1 Sonar Large online"},
		}
	case "replicate":
		return []catalogSeedModel{
			{ModelKey: "meta/meta-llama-3-8b-instruct", Description: "Replicate Meta Llama 3 8B Instruct"},
			{ModelKey: "meta/meta-llama-3-70b-instruct", Description: "Replicate Meta Llama 3 70B Instruct"},
			{ModelKey: "mistralai/mixtral-8x7b-instruct-v0.1", Description: "Replicate Mixtral 8x7B Instruct"},
		}
	case "runway":
		return []catalogSeedModel{
			{ModelKey: "gen3a_turbo", Description: "Runway Gen-3 Alpha Turbo"},
			{ModelKey: "gen3a", Description: "Runway Gen-3 Alpha"},
			{ModelKey: "gen4_aleph", Description: "Runway Gen-4 Aleph"},
		}
	case "vertex":
		return []catalogSeedModel{
			{ModelKey: "gemini-2.0-flash", Description: "Vertex AI Gemini 2.0 Flash"},
			{ModelKey: "gemini-1.5-pro", Description: "Vertex AI Gemini 1.5 Pro"},
			{ModelKey: "gemini-1.5-flash", Description: "Vertex AI Gemini 1.5 Flash"},
			{ModelKey: "publishers/google/models/gemini-1.5-pro", Description: "Vertex publisher path Gemini 1.5 Pro"},
		}
	case "xai":
		return []catalogSeedModel{
			{ModelKey: "grok-3", Description: "xAI Grok 3"},
			{ModelKey: "grok-3-mini", Description: "xAI Grok 3 mini"},
			{ModelKey: "grok-2-latest", Description: "xAI Grok 2 latest"},
			{ModelKey: "grok-2-1212", Description: "xAI Grok 2 1212"},
			{ModelKey: "grok-beta", Description: "xAI Grok beta"},
		}
	case "aliyun":
		return aliyunQwenSeedModels()
	case "baidu":
		return baiduErnieSeedModels()
	case "xiaomi":
		return xiaomiMimoSeedModels()
	default:
		return nil
	}
}

func aliyunQwenSeedModels() []catalogSeedModel {
	return []catalogSeedModel{
		{ModelKey: "qwen-turbo", Description: "Alibaba Qwen Turbo"},
		{ModelKey: "qwen-plus", Description: "Alibaba Qwen Plus"},
		{ModelKey: "qwen-max", Description: "Alibaba Qwen Max"},
		{ModelKey: "qwen-long", Description: "Alibaba Qwen Long context"},
		{ModelKey: "qwen-vl-max", Description: "Alibaba Qwen-VL Max"},
		{ModelKey: "qwen-vl-plus", Description: "Alibaba Qwen-VL Plus"},
		{ModelKey: "qwen-vl-ocr", Description: "Alibaba Qwen-VL OCR"},
		{ModelKey: "qwen2.5-0.5b-instruct", Description: "Alibaba Qwen2.5 0.5B Instruct"},
		{ModelKey: "qwen2.5-1.5b-instruct", Description: "Alibaba Qwen2.5 1.5B Instruct"},
		{ModelKey: "qwen2.5-3b-instruct", Description: "Alibaba Qwen2.5 3B Instruct"},
		{ModelKey: "qwen2.5-7b-instruct", Description: "Alibaba Qwen2.5 7B Instruct"},
		{ModelKey: "qwen2.5-14b-instruct", Description: "Alibaba Qwen2.5 14B Instruct"},
		{ModelKey: "qwen2.5-32b-instruct", Description: "Alibaba Qwen2.5 32B Instruct"},
		{ModelKey: "qwen2.5-72b-instruct", Description: "Alibaba Qwen2.5 72B Instruct"},
		{ModelKey: "qwen2.5-coder-7b-instruct", Description: "Alibaba Qwen2.5 Coder 7B Instruct"},
		{ModelKey: "qwen2.5-coder-32b-instruct", Description: "Alibaba Qwen2.5 Coder 32B Instruct"},
		{ModelKey: "qwen3-30b-a3b", Description: "Alibaba Qwen3 30B A3B MoE"},
		{ModelKey: "qwen3-32b", Description: "Alibaba Qwen3 32B"},
		{ModelKey: "qwen3-235b-a22b", Description: "Alibaba Qwen3 235B A22B MoE"},
		{ModelKey: "qwen-math-plus", Description: "Alibaba Qwen Math Plus"},
		{ModelKey: "qwen-coder-plus", Description: "Alibaba Qwen Coder Plus"},
		{ModelKey: "text-embedding-v3", Description: "Alibaba text-embedding-v3"},
		{ModelKey: "text-embedding-v4", Description: "Alibaba text-embedding-v4"},
	}
}

func baiduErnieSeedModels() []catalogSeedModel {
	return []catalogSeedModel{
		{ModelKey: "ernie-4.0-turbo-8k", Description: "Baidu ERNIE 4.0 Turbo 8K"},
		{ModelKey: "ernie-4.0-8k", Description: "Baidu ERNIE 4.0 8K"},
		{ModelKey: "ernie-4.0-turbo-128k", Description: "Baidu ERNIE 4.0 Turbo 128K"},
		{ModelKey: "ernie-3.5-8k", Description: "Baidu ERNIE 3.5 8K"},
		{ModelKey: "ernie-3.5-128k", Description: "Baidu ERNIE 3.5 128K"},
		{ModelKey: "ernie-speed-128k", Description: "Baidu ERNIE Speed 128K"},
		{ModelKey: "ernie-lite-8k", Description: "Baidu ERNIE Lite 8K"},
		{ModelKey: "ernie-char-8k", Description: "Baidu ERNIE Character 8K"},
		{ModelKey: "embedding-v1", Description: "Baidu Wenxin embedding-v1"},
		{ModelKey: "tao-8k", Description: "Baidu ERNIE Tao 8K"},
	}
}

func xiaomiMimoSeedModels() []catalogSeedModel {
	return []catalogSeedModel{
		{ModelKey: "mimo-v2.5-pro", Description: "Xiaomi MiMo 2.5 Pro (flagship text)"},
		{ModelKey: "mimo-v2-pro", Description: "Xiaomi MiMo 2 Pro"},
		{ModelKey: "mimo-v2.5", Description: "Xiaomi MiMo 2.5 (multimodal text)"},
		{ModelKey: "mimo-v2-omni", Description: "Xiaomi MiMo 2 Omni (multimodal)"},
		{ModelKey: "mimo-v2-flash", Description: "Xiaomi MiMo 2 Flash (fast)"},
	}
}
