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

package climcgen

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
)

// Tool MCP 工具接口
type Tool interface {
	GetTool() mcp.Tool
	Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
	GetName() string
}

// ClimcTool 由 climc CommandTable 自动生成的 MCP 工具
type ClimcTool struct {
	cmd     shell.CMD
	name    string
	tool    mcp.Tool
	adapter *adapters.CloudpodsAdapter
}

func NewClimcTool(cmd shell.CMD, adapter *adapters.CloudpodsAdapter) (*ClimcTool, error) {
	schema, err := buildInputSchema(cmd)
	if err != nil {
		return nil, err
	}
	name := toolNameFromCommand(cmd.Command)
	desc := buildDescription(cmd)
	tool := mcp.NewToolWithRawSchema(name, desc, schema)
	return &ClimcTool{
		cmd:     cmd,
		name:    name,
		tool:    tool,
		adapter: adapter,
	}, nil
}

func (t *ClimcTool) GetTool() mcp.Tool {
	return t.tool
}

func (t *ClimcTool) GetName() string {
	return t.name
}

func (t *ClimcTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := map[string]interface{}{}
	if req.Params.Arguments != nil {
		switch a := req.Params.Arguments.(type) {
		case map[string]interface{}:
			args = a
		default:
			return nil, fmt.Errorf("unexpected arguments type %T", req.Params.Arguments)
		}
	}

	ak, _ := args["ak"].(string)
	sk, _ := args["sk"].(string)
	delete(args, "ak")
	delete(args, "sk")

	if !adapters.HasRequestCredentials(ctx) && (ak == "" || sk == "") {
		return mcp.NewToolResultError(adapters.ErrAuthenticationRequired.Error()), nil
	}

	session, err := t.adapter.GetSession(ctx, ak, sk)
	if err != nil {
		log.Errorf("climc tool %s get session failed: %s", t.name, err)
		return mcp.NewToolResultError(fmt.Sprintf("authentication/session failed: %s", err.Error())), nil
	}

	var output string
	if t.cmd.Command == "server-create" {
		output, err = t.handleServerCreate(ctx, session, args)
	} else {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		output, err = invokeCommand(t.cmd, session, args)
	}
	if err != nil {
		log.Errorf("climc tool %s failed: %s", t.name, err)
		if output != "" {
			return mcp.NewToolResultError(fmt.Sprintf("%s\noutput:\n%s", err.Error(), output)), nil
		}
		return nil, fmt.Errorf("execute %s: %w", t.cmd.Command, err)
	}
	if strings.TrimSpace(output) == "" {
		output = fmt.Sprintf(`{"command":%q,"success":true}`, t.cmd.Command)
	}
	if hint := nextStepHint(t.cmd.Command, t.cmd.Options); hint != "" {
		output = output + "\n\n" + hint
	}
	return mcp.NewToolResultText(output), nil
}

func nextStepHint(command string, options interface{}) string {
	switch {
	case command == "cloud-region-list":
		return `[MCP下一步] 创建虚拟机尚未完成。请用区域 id 调 climc_cloud_region_capability，再 climc_cached_image_list + climc_server_sku_list（可并行），最后 climc_server_create。net 可省略走自动调度，默认不要 network-list。不要在此处结束。`
	case command == "cloud-region-capability":
		return `[MCP下一步] 记下 storage_types2 中的系统盘类型（如 cloud_essd）。继续 climc_cached_image_list 与 climc_server_sku_list，然后 climc_server_create（disk 带 backend；net 可省略）。`
	case command == "cached-image-list":
		return `[MCP下一步] 确认已带 provider 及 region=区域id。继续 climc_server_sku_list（若未查），然后立刻 climc_server_create（disk 含 image+backend；net 可省略）；禁止用 ISO，不要重复查同一镜像列表。`
	case command == "image-list":
		return `[MCP下一步] KVM 用本结果；公有云请改用 climc_cached_image_list。继续 sku 后立刻 climc_server_create。`
	case command == "network-list" || command == "vpc-list":
		return `[MCP下一步] 若用户未指定网络，可省略 net，直接 climc_server_create 走自动调度；若已选中网络 id，create 时传入 net=["<id>"]。`
	case command == "server-sku-list" || command == "storage-list":
		return `[MCP下一步] 请立刻调用 climc_server_create；disk 须含 backend；net 可省略（自动 random / nets:[{exit:false}]）。`
	case command == "server-list":
		return `[MCP下一步] 若用户要启动/停止/重启/删除/改配/重置密码/挂盘/绑定 EIP，拿到 id 后立刻调用对应操作工具，不要只查询就结束。`
	case command == "cloud-account-list":
		return `[MCP下一步] 查详情用 climc_cloud_account_show；同步资源用 climc_cloud_account_sync（可 force=true）。`
	case command == "cloud-account-sync":
		return `[MCP下一步] 同步为异步任务，可用 climc_cloud_account_show 查看 sync_status。`
	case command == "eip-list":
		return `[MCP下一步] 绑定到虚机用 climc_server_associate_eip（需 server id 与 eip id）；详情用 climc_eip_show。`
	case command == "disk-list":
		return `[MCP下一步] 挂到虚机用 climc_server_attach_disk；扩容 climc_disk_resize；详情 climc_disk_show。`
	case command == "host-list":
		return `[MCP下一步] 查宿主机详情用 climc_host_show。`
	case command == "docs_search":
		return `[MCP下一步] 对最相关 path 调用 docs_get 阅读正文后再回答用户。`
	case command == "action-show":
		return `[MCP下一步] 这是操作审计日志。若要继续改资源，回到对应 climc_* 操作工具。`
	case isCreateFlowListCommand(options):
		return `[MCP下一步] 这只是创建前的资源查询。任务完成条件是成功调用 climc_server_create。`
	default:
		return ""
	}
}
