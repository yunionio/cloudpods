package models

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetMCPAgentManager()
}

var mcpAgentManager *SMCPAgentManager

var mcpAgentWorkerMan *appsrv.SWorkerManager

func GetMCPAgentManager() *SMCPAgentManager {
	if mcpAgentManager != nil {
		return mcpAgentManager
	}
	mcpAgentManager = &SMCPAgentManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SMCPAgent{},
			"mcp_agents_tbl",
			"mcp_agent",
			"mcp_agents",
		),
	}
	mcpAgentManager.SetVirtualObject(mcpAgentManager)
	return mcpAgentManager
}

type SMCPAgentManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SMCPAgent struct {
	db.SSharableVirtualResourceBase

	// LLMUrl 对应后端大模型的 base 请求地址
	LLMUrl string `width:"512" charset:"utf8" nullable:"false" list:"user" create:"required" update:"user"`
	// LLMDriver 对应使用的大模型驱动（llm_client），现在可以被设置为 ollama 或 openai
	LLMDriver string `width:"64" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// Model 使用的模型名称
	Model string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// ApiKey 即在 llm_driver 中需要用到的认证
	ApiKey string `width:"512" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	// McpServer 即 mcp 服务器的后端地址
	McpServer string `width:"512" charset:"utf8" nullable:"false" list:"user" create:"optional" update:"user"`
}

func (man *SMCPAgentManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	man.SSharableVirtualResourceBaseManager.CustomizeHandlerInfo(info)

	// log.Infoln("query name of handler info", info.GetName(nil))

	switch info.GetName(nil) {
	case "get_specific":
		info.SetProcessTimeout(time.Hour * 4).SetWorkerManager(mcpAgentWorkerMan)
	}
}

func (man *SMCPAgentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.MCPAgentCreateInput) (*api.MCPAgentCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	// 如果提供了 llm_id，则通过 LLM 获取 llm_url 和 model
	if len(input.LLMId) > 0 {
		llmObj, err := GetLLMManager().FetchByIdOrName(ctx, userCred, input.LLMId)
		if err != nil {
			return input, errors.Wrapf(err, "fetch LLM by id %s", input.LLMId)
		}
		llm := llmObj.(*SLLM)
		llmUrl, err := llm.GetLLMUrl(ctx, userCred)
		if err != nil {
			return input, errors.Wrapf(err, "get LLM URL from LLM %s", input.LLMId)
		}
		input.LLMUrl = llmUrl

		sku, err := llm.GetLLMSku("")
		if err != nil {
			return input, errors.Wrapf(err, "get LLM Sku from LLM %s", input.LLMId)
		}
		input.Model = sku.LLMModelName
	}

	// 验证 llm_url 不为空
	if len(input.LLMUrl) == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "llm_url is required (or provide llm_id to auto-fetch)")
	}

	// 验证 llm_driver 必须是 ollama 或 openai
	input.LLMDriver = strings.ToLower(strings.TrimSpace(input.LLMDriver))
	if !api.IsLLMClientType(input.LLMDriver) {
		return input, errors.Wrapf(httperrors.ErrInputParameter, "llm_driver must be one of: %s, got: %s", api.LLM_CLIENT_TYPES.List(), input.LLMDriver)
	}

	// 验证 model 不为空
	if len(input.Model) == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "model is required")
	}

	// 验证 mcp_server 不为空
	if len(input.McpServer) == 0 {
		input.McpServer = options.Options.MCPServerURL
	}

	// 对于 openai 驱动，api_key 是必需的
	if input.LLMDriver == string(api.LLM_CLIENT_OPENAI) && len(input.ApiKey) == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "api_key is required when llm_driver is openai")
	}

	input.Status = api.STATUS_READY
	return input, nil
}

func (man *SMCPAgentManager) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.MCPAgentUpdateInput) (*api.MCPAgentUpdateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}

	// 如果提供了 llm_id，则通过 LLM 获取 llm_url 和 model
	if input.LLMId != nil && len(*input.LLMId) > 0 {
		llmObj, err := GetLLMManager().FetchByIdOrName(ctx, userCred, *input.LLMId)
		if err != nil {
			return input, errors.Wrapf(err, "fetch LLM by id %s", *input.LLMId)
		}
		llm := llmObj.(*SLLM)
		llmUrl, err := llm.GetLLMUrl(ctx, userCred)
		if err != nil {
			return input, errors.Wrapf(err, "get LLM URL from LLM %s", *input.LLMId)
		}
		input.LLMUrl = &llmUrl

		sku, err := llm.GetLLMSku("")
		if err != nil {
			return input, errors.Wrapf(err, "get LLM Sku from LLM %s", *input.LLMId)
		}
		input.Model = &sku.LLMModelName
	}

	// 如果更新 llm_driver，验证其值
	if input.LLMDriver != nil {
		*input.LLMDriver = strings.ToLower(strings.TrimSpace(*input.LLMDriver))
		if !api.IsLLMClientType(*input.LLMDriver) {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "llm_driver must be one of: %s, got: %s", api.LLM_CLIENT_TYPES.List(), *input.LLMDriver)
		}
	}

	return input, nil
}

func (man *SMCPAgentManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.MCPAgentListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}

	if len(input.LLMDriver) > 0 {
		q = q.Equals("llm_driver", strings.ToLower(strings.TrimSpace(input.LLMDriver)))
	}

	return q, nil
}

func (manager *SMCPAgentManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.MCPAgentDetails {
	rows := make([]api.MCPAgentDetails, len(objs))
	vrows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	agents := []SMCPAgent{}
	jsonutils.Update(&agents, objs)

	for i := range rows {
		rows[i].SharableVirtualResourceDetails = vrows[i]
		if i < len(agents) {
			rows[i].LLMUrl = agents[i].LLMUrl
			rows[i].LLMDriver = agents[i].LLMDriver
			rows[i].Model = agents[i].Model
			rows[i].ApiKey = agents[i].ApiKey
			rows[i].McpServer = agents[i].McpServer
		}
	}

	return rows
}

func (mcp *SMCPAgent) GetLLMClientDriver() ILLMClient {
	return GetLLMClientDriver(api.LLMClientType(mcp.LLMDriver))
}

func (mcp *SMCPAgent) GetDetailsMcpTools(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// 创建 MCP 客户端
	timeout := time.Duration(options.Options.MCPAgentTimeout) * time.Second
	mcpClient := utils.NewMCPClient(options.Options.MCPServerURL, timeout, userCred)

	// 获取工具列表
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list MCP tools")
	}

	return jsonutils.Marshal(tools), nil
}

func (mcp *SMCPAgent) GetDetailsToolRequest(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.LLMToolRequestInput,
) (jsonutils.JSONObject, error) {
	// 创建 MCP 客户端
	timeout := time.Duration(options.Options.MCPAgentTimeout) * time.Second
	mcpClient := utils.NewMCPClient(options.Options.MCPServerURL, timeout, userCred)
	defer mcpClient.Close()

	// 调用工具
	result, err := mcpClient.CallTool(ctx, input.ToolName, input.Arguments)
	if err != nil {
		return nil, errors.Wrapf(err, "call tool %s", input.ToolName)
	}

	return jsonutils.Marshal(result), nil
}

func (mcp *SMCPAgent) GetDetailsChatTest(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.LLMChatTestInput,
) (jsonutils.JSONObject, error) {
	llmClient := mcp.GetLLMClientDriver()
	if llmClient == nil {
		return nil, errors.Error("failed to get LLM client driver")
	}

	message := llmClient.NewUserMessage(input.Message)

	result, err := llmClient.Chat(ctx, mcp, []ILLMChatMessage{message}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "chat with LLM")
	}

	return jsonutils.Marshal(result), nil
}

func (mcp *SMCPAgent) GetDetailsRequest(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.LLMMCPAgentRequestInput,
) (jsonutils.JSONObject, error) {
	// 调用 ProcessMCPAgentRequest
	answer, err := mcp.process(ctx, userCred, &input)
	if err != nil {
		return nil, errors.Wrap(err, "process MCP agent request")
	}

	// 返回结果
	result := map[string]interface{}{
		"answer": answer.Answer,
	}
	return jsonutils.Marshal(result), nil
}

// process 处理用户请求
func (mcp *SMCPAgent) process(ctx context.Context, userCred mcclient.TokenCredential, req *api.LLMMCPAgentRequestInput) (*api.MCPAgentResponse, error) {
	// 获取 MCP Server 的工具列表
	mcpClient := utils.NewMCPClient(mcp.McpServer, 10*time.Minute, userCred)
	defer mcpClient.Close()
	mcpTools, err := mcpClient.ListTools(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list MCP tools")
	}
	log.Infof("Got %d tools from MCP Server", len(mcpTools))

	// get llmClient
	llmClient := mcp.GetLLMClientDriver()
	if llmClient == nil {
		return nil, errors.Error("failed to get LLM client driver")
	}

	tools := llmClient.ConvertMCPTools(mcpTools)

	// 构建系统提示词
	systemPrompt := buildSystemPrompt()

	// 初始化消息历史，使用接口类型
	messages := []ILLMChatMessage{
		llmClient.NewSystemMessage(systemPrompt),
		llmClient.NewUserMessage(req.Query),
	}

	// 记录工具调用
	var toolCallRecords []api.MCPAgentToolCallRecord

	// Agent 循环
	for i := 0; i < api.MCPAgentMaxIterations; i++ {
		log.Infof("Agent iteration %d", i+1)

		// 调用 LLM 客户端，传入接口类型
		resp, err := llmClient.Chat(ctx, mcp, messages, tools)
		if err != nil {
			return nil, errors.Wrap(err, "chat with LLM client")
		}

		// 检查是否有工具调用
		if !resp.HasToolCalls() {
			// 没有工具调用，返回最终答案
			return &api.MCPAgentResponse{
				Success:   true,
				Answer:    resp.GetContent(),
				ToolCalls: toolCallRecords,
			}, nil
		}

		// 处理工具调用
		toolCalls := resp.GetToolCalls()
		log.Infof("Got %d tool calls from LLM", len(toolCalls))

		// 添加助手消息（带工具调用），使用接口类型
		messages = append(messages, llmClient.NewAssistantMessageWithToolCalls(toolCalls))

		// 执行每个工具调用
		for _, tc := range toolCalls {
			fc := tc.GetFunction()
			toolName := fc.GetName()
			arguments := fc.GetArguments()

			// 确保 arguments 不为 nil
			if arguments == nil {
				arguments = make(map[string]interface{})
			}

			log.Infof("Calling tool: %s with arguments: %v", toolName, arguments)

			// 调用 MCP 工具
			result, err := mcpClient.CallTool(ctx, toolName, arguments)
			resultText := utils.FormatToolResult(toolName, result, err)
			log.Infoln("Get result from mcp query", resultText)

			// 记录工具调用
			toolCallRecords = append(toolCallRecords, api.MCPAgentToolCallRecord{
				ToolName:  toolName,
				Arguments: arguments,
				Result:    resultText,
			})

			// 添加工具结果消息，使用接口类型
			messages = append(messages, llmClient.NewToolMessage(toolName, resultText))
		}
	}

	// 达到最大迭代次数
	return &api.MCPAgentResponse{
		Success:   false,
		Answer:    "处理请求时达到最大迭代次数，请尝试简化您的问题。",
		Error:     "max iterations reached",
		ToolCalls: toolCallRecords,
	}, nil
}

// buildSystemPrompt 构建系统提示词
func buildSystemPrompt() string {
	return api.MCP_AGENT_SYSTEM_PROMPT
}
