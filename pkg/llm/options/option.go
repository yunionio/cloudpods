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

package options

import common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"

type LLMOptions struct {
	common_options.CommonOptions
	common_options.DBOptions

	LLMWorkingDirectory string `help:"llm working directory" default:"/opt/cloud/workspace/llm/"`

	InstantModelSyncTaskWorkerCount int `help:"backup task worker count" default:"128"`
	ModelSyncTaskWaitSecs           int `help:"model sync task wait seconds" default:"30"`

	BackupTaskWorkerCount int `help:"backup task worker count" default:"128"`
	ImportTaskWorkerCount int `help:"import task worker count" default:"8"`
	StartTaskWorkerCount  int `help:"start task worker count" default:"128"`

	// MCP Agent 配置
	MCPServerURL    string `help:"MCP Server URL" default:"http://default-mcp-server:30876"`
	MCPAgentTimeout int    `help:"MCP Agent request timeout in seconds" default:"120"`

	MCPAgentUserCharLimit      int `help:"MCP Agent user char limit" default:"3200"`
	MCPAgentAssistantCharLimit int `help:"MCP Agent assistant char limit" default:"6400"`

	// LLM model catalog (browsable curated entries). Value can be either an
	// http(s) URL or a local file path; sources without an http:// or https://
	// prefix are treated as local files.
	ModelCatalogURL                  string `help:"URL of the LLM model catalog YAML; values without http(s):// prefix are treated as local file paths" default:"https://www.cloudpods.org/model-catalog.yaml"`
	LLMImagesCatalogURL              string `help:"URL of the LLM community images YAML; values without http(s):// prefix are treated as local file paths" default:"https://www.cloudpods.org/llmimages.yaml"`
	LLMCatalogRefreshIntervalMinutes int    `help:"Catalog refresh interval in minutes; 0 disables periodic refresh" default:"60"`

	// Server-side proxy for outbound HuggingFace / ModelScope calls (used by
	// the dashboard for model browsing). Mirrors GPUStack's /v1/proxy design.
	HuggingFaceEndpoint string `help:"Replacement endpoint for huggingface.co (e.g., https://hf-mirror.com); empty means no substitution"`
	HuggingFaceToken    string `help:"Optional HuggingFace bearer token; injected as Authorization header on huggingface.co requests"`

	ModelScopeEndpoint string `help:"ModelScope API endpoint (e.g., https://www.modelscope.cn)" default:"https://www.modelscope.cn"`
	ModelScopeToken    string `help:"Optional ModelScope bearer token; injected as Authorization header on modelscope.cn requests"`
}

var (
	Options LLMOptions
)
