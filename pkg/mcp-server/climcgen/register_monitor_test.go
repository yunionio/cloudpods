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
	"testing"

	"yunion.io/x/onecloud/cmd/climc/shell"
)

// server-monitor sends arbitrary QMP/HMP commands to a running guest, which
// exceeds the semantics of monitoring and must never be callable by AI/MCP
// (e.g. via prompt injection or the tool-request endpoint)
func TestServerMonitorNotExposedToMcp(t *testing.T) {
	for _, cmd := range shell.CommandTable {
		if cmd.Command != "server-monitor" {
			continue
		}
		if d := collectMcpDesc(cmd.Options); d != "" {
			t.Fatalf("server-monitor must not be registered as MCP tool, got mcp-desc %q", d)
		}
		return
	}
	t.Fatalf("server-monitor command not found in CommandTable")
}
