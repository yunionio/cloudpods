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

package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/climcgen"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
	"yunion.io/x/onecloud/pkg/mcp-server/registry"
)

// CloudpodsMCPServer 是 MCP 服务器的核心结构体，包含配置、日志、MCP 实例、注册中心和工具列表
type CloudpodsMCPServer struct {
	mcpServer *server.MCPServer
	registry  *registry.Registry
	tools     []climcgen.Tool
}

// NewServerOptions 创建 MCP 服务器时可覆盖的选项。
type NewServerOptions struct {
	// Instructions MCP 全局说明；空则使用 climcgen.BuildServerInstructions(PlatformName)（再叠加 RegisterExtraInstructions）
	Instructions string
}

// NewServer 创建一个新的 Cloudpods MCP 服务器实例。
// 工具从 climc shell.CommandTable + Options struct tag（mcp-desc）自动生成；
// 可通过 climcgen.RegisterExtraTools 追加工具。
func NewServer() *CloudpodsMCPServer {
	return NewServerWithOptions(nil)
}

// NewServerWithOptions 同 NewServer，允许覆盖 Instructions 等。
// 额外 NewMCPServer 参数可通过 RegisterMCPServerOptions / RegisterMCPHooks 注册。
func NewServerWithOptions(opt *NewServerOptions) *CloudpodsMCPServer {
	buildInstructions := func() string {
		instructions := climcgen.BuildServerInstructions(options.ResolvedPlatformName())
		if opt != nil && strings.TrimSpace(opt.Instructions) != "" {
			instructions = opt.Instructions
		}
		if extra := climcgen.BuildExtraInstructions(); extra != "" {
			instructions = strings.TrimSpace(instructions) + "\n\n" + extra
		}
		return instructions
	}

	serverName := strings.TrimSpace(options.Options.MCPServerName)
	if serverName == "" {
		serverName = options.ResolvedPlatformName()
	}
	version := options.Options.MCPServerVersion
	serverOpts := []server.ServerOption{
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithInstructions(buildInstructions()),
	}
	serverOpts = append(serverOpts, buildRegisteredMCPServerOptions(MCPServerBuildContext{
		BuildInstructions: buildInstructions,
		ServerName:        serverName,
		Version:           version,
	})...)
	mcpServer := server.NewMCPServer(serverName, version, serverOpts...)

	reg := registry.NewRegistry()
	adapter := adapters.NewCloudpodsAdapter()

	allTools, err := climcgen.BuildTools(adapter)
	if err != nil {
		log.Fatalf("build climc MCP tools failed: %s", err)
	}
	if extra := climcgen.BuildExtraTools(); len(extra) > 0 {
		allTools = append(extra, allTools...)
	}

	return &CloudpodsMCPServer{
		mcpServer: mcpServer,
		registry:  reg,
		tools:     allTools,
	}
}

// Initialize 初始化注册中心和所有工具
func (s *CloudpodsMCPServer) Initialize() error {
	if err := s.registry.Initialize(s.mcpServer); err != nil {
		return fmt.Errorf("初始化工具注册中心失败: %w", err)
	}
	if err := s.registerAllTools(); err != nil {
		return fmt.Errorf("注册内置工具失败: %w", err)
	}
	return nil
}

func (s *CloudpodsMCPServer) registerAllTools() error {
	for _, tool := range s.tools {
		if err := s.registry.RegisterTool(
			tool.GetName(),
			tool.GetTool(),
			tool.Handle,
		); err != nil {
			return fmt.Errorf("注册工具失败: %w", err)
		}
	}
	log.Infof("All tools register completed, count=%d", len(s.tools))
	return nil
}

// authenticateRequest 从 Header 注入会话凭据（可选）；无凭据时 ok=false，仍返回原 ctx。
// /sse 与 tools/list 允许匿名；tools/call 在工具 Handler 内强制鉴权。
func authenticateRequest(ctx context.Context, r *http.Request) (context.Context, bool) {
	tokenStr := r.Header.Get(api.AUTH_TOKEN_HEADER)
	akStr := r.Header.Get("AK")
	skStr := r.Header.Get("SK")
	apiKey := r.Header.Get("X-API-Key")

	if tokenStr != "" {
		if auth.IsAuthed() {
			userCred, err := auth.Verify(ctx, tokenStr)
			if err != nil {
				log.Errorf("Verify token failed: %s", err)
			} else {
				ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
				return ctx, true
			}
		}
	}

	if akStr != "" && skStr != "" {
		ctx = context.WithValue(ctx, adapters.ContextKeyAK, akStr)
		ctx = context.WithValue(ctx, adapters.ContextKeySK, skStr)
		return ctx, true
	}

	if apiKey != "" {
		if auth.IsAuthed() {
			if userCred, err := auth.Verify(ctx, apiKey); err == nil {
				ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, userCred)
				return ctx, true
			}
		}
		decoded, err := base64.StdEncoding.DecodeString(apiKey)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				ctx = context.WithValue(ctx, adapters.ContextKeyAK, parts[0])
				ctx = context.WithValue(ctx, adapters.ContextKeySK, parts[1])
				return ctx, true
			}
		}
	}

	return ctx, false
}

// Start 以sse模式启动 mcp 服务
func (s *CloudpodsMCPServer) Start() error {
	contextFunc := func(ctx context.Context, r *http.Request) context.Context {
		ctx2, _ := authenticateRequest(ctx, r)
		return ctx2
	}

	mux := http.NewServeMux()

	sseServer := server.NewSSEServer(
		s.mcpServer,
		server.WithSSEContextFunc(contextFunc),
		server.WithHTTPServer(&http.Server{Handler: mux}),
	)
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		appsrv.VersionHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		appsrv.StatisticHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		appsrv.PingHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/worker_stats", func(w http.ResponseWriter, r *http.Request) {
		appsrv.WorkerStatsHandler(context.Background(), w, r)
	})
	mux.HandleFunc("/process_stats", func(w http.ResponseWriter, r *http.Request) {
		appsrv.ProcessStatsHandler(context.Background(), w, r)
	})
	mux.Handle(sseServer.CompleteSsePath(), sseServer.SSEHandler())
	mux.Handle(sseServer.CompleteMessagePath(), sseServer.MessageHandler())

	addr := fmt.Sprintf("%s:%d", options.Options.Address, options.Options.Port)
	errCh := make(chan error, 1)
	go func() {
		errCh <- sseServer.Start(addr)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Infof("Start mcp server successfully on %s", addr)
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case sig := <-sigCh:
		log.Infof("Received signal %v, shutting down mcp server...", sig)
		signal.Stop(sigCh)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := sseServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		log.Infof("Mcp server stopped")
		return nil
	}
}

// StartStdio 以stdio模式启动 mcp 服务
func (s *CloudpodsMCPServer) StartStdio() error {
	err := server.ServeStdio(s.mcpServer)
	if err != nil {
		return err
	}
	log.Infof("Start mcp server successfully")
	return nil
}
