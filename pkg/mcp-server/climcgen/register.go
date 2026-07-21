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
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
)

// BuildTools 扫描 shell.CommandTable，注册 Options 上带 mcp-desc 的命令为 MCP tools。
func BuildTools(adapter *adapters.CloudpodsAdapter) ([]Tool, error) {
	byName := make(map[string]shell.CMD, len(shell.CommandTable))
	for _, cmd := range shell.CommandTable {
		byName[cmd.Command] = cmd
	}

	commands := discoverMcpDescCommands()
	result := make([]Tool, 0, len(commands))
	for _, name := range commands {
		cmd, ok := byName[name]
		if !ok {
			continue
		}
		tool, err := NewClimcTool(cmd, adapter)
		if err != nil {
			return nil, fmt.Errorf("build tool for %s: %w", name, err)
		}
		result = append(result, tool)
		log.Infof("climcgen: registered MCP tool %s (%s)", tool.GetName(), name)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no climc tools registered; add mcp-desc on Options")
	}
	return result, nil
}

// discoverMcpDescCommands 扫描 CommandTable，收集 Options 带 mcp-desc 的命令。
// server-create /「最终动作」优先排前，便于模型在创建意图下选中。
func discoverMcpDescCommands() []string {
	type ranked struct {
		name  string
		rank  int
		order int
	}
	found := make([]ranked, 0)
	for i, cmd := range shell.CommandTable {
		desc := collectMcpDesc(cmd.Options)
		if desc == "" {
			continue
		}
		r := ranked{name: cmd.Command, rank: 2, order: i}
		switch {
		case cmd.Command == "server-create" || strings.Contains(desc, "最终动作") || strings.Contains(desc, "优先调用"):
			r.rank = 0
		case strings.Contains(desc, createFlowMidStepMarker):
			r.rank = 1
		}
		found = append(found, r)
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].rank != found[j].rank {
			return found[i].rank < found[j].rank
		}
		return found[i].order < found[j].order
	})
	out := make([]string, 0, len(found))
	seen := make(map[string]bool, len(found))
	for _, r := range found {
		if seen[r.name] {
			continue
		}
		seen[r.name] = true
		out = append(out, r.name)
	}
	return out
}
