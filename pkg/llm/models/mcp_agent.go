package models

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	seclib "yunion.io/x/pkg/utils"
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

	// LLMId 关联的 LLM 实例 ID
	LLMId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

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

func (mcp *SMCPAgent) BeforeInsert() {
	if len(mcp.Id) == 0 {
		mcp.Id = db.DefaultUUIDGenerator()
	}
	if len(mcp.ApiKey) > 0 {
		sec, err := seclib.EncryptAESBase64(mcp.Id, mcp.ApiKey)
		if err != nil {
			log.Errorf("EncryptAESBase64 fail %s", err)
		} else {
			mcp.ApiKey = sec
		}
	}
	mcp.SSharableVirtualResourceBase.BeforeInsert()
}

func (mcp *SMCPAgent) BeforeUpdate() {
	if len(mcp.ApiKey) > 0 {
		// heuristic to check if it is plaintext
		_, err := seclib.DescryptAESBase64(mcp.Id, mcp.ApiKey)
		if err != nil {
			sec, err := seclib.EncryptAESBase64(mcp.Id, mcp.ApiKey)
			if err != nil {
				log.Errorf("EncryptAESBase64 fail %s", err)
			} else {
				mcp.ApiKey = sec
			}
		}
	}
}

func (mcp *SMCPAgent) GetApiKey() (string, error) {
	if len(mcp.ApiKey) == 0 {
		return "", nil
	}
	// try decrypt
	key, err := seclib.DescryptAESBase64(mcp.Id, mcp.ApiKey)
	if err == nil {
		return key, nil
	}
	return mcp.ApiKey, nil
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
		input.LLMId = llm.Id
		llmUrl, err := llm.GetLLMUrl(ctx, userCred)
		if err != nil {
			return input, errors.Wrapf(err, "get LLM URL from LLM %s", input.LLMId)
		}
		input.LLMUrl = llmUrl

		if len(input.Model) == 0 {
			mdlInfos, err := llm.getProbedInstantModelsExt(ctx, userCred)
			if err != nil {
				return input, errors.Wrap(err, "get probed models from LLM instance")
			}
			if len(mdlInfos) == 0 {
				return input, httperrors.NewBadRequestError("no available models found in LLM instance %s", input.LLMId)
			}
			var firstModel api.LLMInternalInstantMdlInfo
			for _, mdlInfo := range mdlInfos {
				firstModel = mdlInfo
				break
			}
			input.Model = fmt.Sprintf("%s:%s", firstModel.Name, firstModel.Tag)
		}
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

		if input.Model == nil || len(*input.Model) == 0 {
			mdlInfos, err := llm.getProbedInstantModelsExt(ctx, userCred)
			if err != nil {
				return input, errors.Wrap(err, "get probed models from LLM instance")
			}
			if len(mdlInfos) == 0 {
				return input, httperrors.NewBadRequestError("no available models found in LLM instance %s", *input.LLMId)
			}
			var firstModel api.LLMInternalInstantMdlInfo
			for _, mdlInfo := range mdlInfos {
				firstModel = mdlInfo
				break
			}
			modelStr := fmt.Sprintf("%s:%s", firstModel.Name, firstModel.Tag)
			input.Model = &modelStr
		}
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

	llmIds := make([]string, 0)
	for i := range agents {
		if len(agents[i].LLMId) > 0 {
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

// func (mcp *SMCPAgent) GetDetailsChatTest(
// 	ctx context.Context,
// 	userCred mcclient.TokenCredential,
// 	input api.LLMChatTestInput,
// ) (jsonutils.JSONObject, error) {
// 	llmClient := mcp.GetLLMClientDriver()
// 	if llmClient == nil {
// 		return nil, errors.Error("failed to get LLM client driver")
// 	}

// 	message := llmClient.NewUserMessage(input.Message)

// 	result, err := llmClient.Chat(ctx, mcp, []ILLMChatMessage{message}, nil)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "chat with LLM")
// 	}

// 	return jsonutils.Marshal(result), nil
// }

func (mcp *SMCPAgent) PerformChatStream(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMMCPAgentRequestInput,
) (jsonutils.JSONObject, error) {
	appParams := appsrv.AppContextGetParams(ctx)
	if appParams == nil {
		return nil, errors.Error("failed to get app params")
	}

	w := appParams.Response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	} else {
		return nil, errors.Error("Streaming unsupported!")
	}

	_, err := mcp.process(ctx, userCred, &input, func(content string) error {
		if len(content) > 0 {
			for line := range strings.SplitSeq(content, "\n") {
				fmt.Fprintf(w, "data: %s\n", line)
			}
			fmt.Fprintf(w, "\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(w, "data: Error: %v\n\n", err)
	}

	return nil, nil
}

// process 处理用户请求
// 强制分为两个阶段：
// 阶段一：使用 Chat 非流式获取工具调用参数，并执行工具
// 阶段二：使用 ChatStream 流式获取最终响应
func (mcp *SMCPAgent) process(ctx context.Context, userCred mcclient.TokenCredential, req *api.LLMMCPAgentRequestInput, onStream func(string) error) (*api.MCPAgentResponse, error) {
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

	// 初始化消息历史
	messages := make([]ILLMChatMessage, 0)
	messages = append(messages, llmClient.NewSystemMessage(systemPrompt))

	// 处理历史消息
	if len(req.History) > 0 {
		historyMessages := processHistoryMessages(
			req.History,
			llmClient,
			options.Options.MCPAgentUserCharLimit,
			options.Options.MCPAgentAssistantCharLimit,
		)
		messages = append(messages, historyMessages...)
	}

	messages = append(messages, llmClient.NewUserMessage(req.Message))

	// 记录工具调用
	var toolCallRecords []api.MCPAgentToolCallRecord

	log.Infof("Phase 1: Thinking & Acting...")
	resp, err := llmClient.Chat(ctx, mcp, messages, tools)
	if err != nil {
		return nil, errors.Wrap(err, "phase 1 chat error")
	}

	// 检查是否有工具调用
	if !resp.HasToolCalls() {
		// 如果阶段一没有调用工具，模拟推流返回结果
		content := resp.GetContent()
		if onStream != nil && len(content) > 0 {
			// 模拟流式输出：按字符逐块推送
			chunkSize := 10 // 每次推送10个字符
			for i := 0; i < len(content); i += chunkSize {
				end := i + chunkSize
				if end > len(content) {
					end = len(content)
				}
				chunk := content[i:end]
				if err := onStream(chunk); err != nil {
					return nil, errors.Wrap(err, "stream content error")
				}
				// 添加小延迟模拟真实流式输出
				time.Sleep(10 * time.Millisecond)
			}
		}
		return &api.MCPAgentResponse{
			Success:   true,
			Answer:    content,
			ToolCalls: toolCallRecords,
		}, nil
	}

	// 处理工具调用
	toolCalls := resp.GetToolCalls()
	log.Infof("Got %d tool calls from Phase 1", len(toolCalls))

	toolCallRecords, toolMessages, err := processToolCalls(ctx, toolCalls, mcpClient, llmClient)
	if err != nil {
		return nil, errors.Wrap(err, "process tool calls")
	}

	// 将工具调用相关的消息加入历史
	messages = append(messages, toolMessages...)

	log.Infof("Phase 2: Streaming Response...")

	var finalAnswer strings.Builder

	err = llmClient.ChatStream(ctx, mcp, messages, tools, func(chunk ILLMChatResponse) error {
		content := chunk.GetContent()
		if len(content) > 0 {
			// 聚合最终答案
			finalAnswer.WriteString(content)

			// 实时流式输出
			if onStream != nil {
				if err := onStream(content); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "phase 2 stream error")
	}

	return &api.MCPAgentResponse{
		Success:   true,
		Answer:    finalAnswer.String(),
		ToolCalls: toolCallRecords,
	}, nil
}

// buildSystemPrompt 构建系统提示词
func buildSystemPrompt() string {
	return api.MCP_AGENT_SYSTEM_PROMPT
}

func processHistoryMessages(
	history []api.MCPAgentChatMessage,
	llmClient ILLMClient,
	maxUserChars int,
	maxAssistantChars int,
) []ILLMChatMessage {
	if len(history) == 0 {
		return []ILLMChatMessage{}
	}

	var userChars, assistantChars int
	processedMessages := make([]ILLMChatMessage, 0)

	// 从最新的消息开始遍历，保留最新消息，丢弃最旧消息
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		msgChars := len(msg.Content)

		switch msg.Role {
		case "user":
			if userChars+msgChars > maxUserChars {
				break
			}
			userChars += msgChars
			processedMessages = append(processedMessages, llmClient.NewUserMessage(msg.Content))
		case "assistant":
			if assistantChars+msgChars > maxAssistantChars {
				break
			}
			assistantChars += msgChars

			if len(msg.Content) > 0 {
				processedMessages = append(processedMessages, llmClient.NewAssistantMessage(msg.Content))
			}
		}
	}

	for i, j := 0, len(processedMessages)-1; i < j; i, j = i+1, j-1 {
		processedMessages[i], processedMessages[j] = processedMessages[j], processedMessages[i]
	}

	return processedMessages
}

// processToolCalls 处理工具调用
func processToolCalls(
	ctx context.Context,
	toolCalls []ILLMToolCall,
	mcpClient *utils.MCPClient,
	llmClient ILLMClient,
) ([]api.MCPAgentToolCallRecord, []ILLMChatMessage, error) {
	toolCallRecords := make([]api.MCPAgentToolCallRecord, 0)
	messagesToAdd := make([]ILLMChatMessage, 0)

	messagesToAdd = append(messagesToAdd, llmClient.NewAssistantMessageWithToolCalls(toolCalls))

	// 执行每个工具调用
	for _, tc := range toolCalls {
		fc := tc.GetFunction()
		toolName := fc.GetName()
		arguments := fc.GetArguments()

		if arguments == nil {
			arguments = make(map[string]interface{})
		}

		log.Infof("Calling tool: %s with arguments: %v", toolName, arguments)

		// 调用 MCP 工具
		result, err := mcpClient.CallTool(ctx, toolName, arguments)
		resultText := utils.FormatToolResult(toolName, result, err)
		log.Infoln("Get result from mcp query", resultText)

		toolCallRecords = append(toolCallRecords, api.MCPAgentToolCallRecord{
			Id:        tc.GetId(),
			ToolName:  toolName,
			Arguments: arguments,
			Result:    resultText,
		})

		// 将工具执行结果加入历史
		messagesToAdd = append(messagesToAdd, llmClient.NewToolMessage(tc.GetId(), toolName, resultText))
	}

	return toolCallRecords, messagesToAdd, nil
}
