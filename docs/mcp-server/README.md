# MCP Server

Cloudpods MCP Server：通过 MCP 协议把 climc 能力暴露给 AI 客户端（Cursor / Claude 等）。

## 目录结构

```
├── adapters/   # Cloudpods 认证与 ClientSession
├── climcgen/   # 从 climc CommandTable + Options tag 生成 MCP tools
├── options/    # 服务配置
├── registry/   # MCP tool 注册
├── server/     # SSE / stdio 服务
└── service/    # 进程入口装配
```

## 运行机制

1. 启动时 blank-import climc shell 包，填充 `shell.CommandTable`
2. 扫描 Options 上带 `mcp-desc` 的命令，用 Options struct tag 生成 schema 并注册 tools
3. 工具调用时用 AK/SK（或 Header）建 session，执行对应 climc callback，JSON 输出返回给客户端

## 扩展工具

在对应 climc Options 上增加 `_ struct{} \`mcp-desc:"..."\``（并按需给字段加 `mcp:"true"`），重启 mcp-server 即可注册。
