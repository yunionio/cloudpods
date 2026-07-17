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

/*
Package climcgen 从 climc 的 shell.CommandTable 与 Options struct tag
自动生成 MCP tool schema，并调用对应 callback 执行。

注册范围：只注册 Options 上带 mcp-desc 的命令。

AI/MCP 参数通过 Options 字段上的 mcp tag 标记：
  - mcp:"true"     暴露给 MCP schema
  - mcp:"required" 暴露且在 MCP schema 中标记为 required

未标记的可选参数不会进入 schema（positional / climc required 仍会保留）。

MCP tool 补充说明通过 Options 上的 mcp-desc tag 写入（常用 `_ struct{}` 承载），
由 buildDescription 拼入 tool description。
*/
package climcgen
