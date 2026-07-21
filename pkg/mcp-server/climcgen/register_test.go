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

func TestDiscoverMcpDescCommands_UsesTagNotHardcodedList(t *testing.T) {
	saved := shell.CommandTable
	defer func() { shell.CommandTable = saved }()

	type withDesc struct {
		_    struct{} `mcp-desc:"【创建虚拟机的最终动作/优先调用】demo"`
		Name string   `mcp:"required"`
	}
	type noDesc struct {
		Name string
	}
	type listDesc struct {
		_ struct{} `mcp-desc:"【创建流程中的中间步骤】list demo"`
	}

	shell.CommandTable = []shell.CMD{
		{Options: &noDesc{}, Command: "should-skip", Desc: "no mcp-desc"},
		{Options: &listDesc{}, Command: "demo-list", Desc: "list"},
		{Options: &withDesc{}, Command: "server-create", Desc: "create"},
	}

	got := discoverMcpDescCommands()
	if len(got) != 2 {
		t.Fatalf("want 2 commands, got %v", got)
	}
	if got[0] != "server-create" {
		t.Fatalf("server-create should be first, got %v", got)
	}
	if got[1] != "demo-list" {
		t.Fatalf("want demo-list second, got %v", got)
	}
}

func TestBuildDescription_PrefersMcpDesc(t *testing.T) {
	type withDesc struct {
		_ struct{} `mcp-desc:"中文说明"`
	}
	got := buildDescription(shell.CMD{
		Command: "server-list",
		Desc:    "List servers",
		Options: &withDesc{},
	})
	want := "[climc server-list] 中文说明"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = buildDescription(shell.CMD{
		Command: "demo",
		Desc:    "English only",
		Options: &struct{ Name string }{},
	})
	want = "[climc demo] English only"
	if got != want {
		t.Fatalf("fallback got %q want %q", got, want)
	}
}

func TestCollectMcpDesc_DoesNotInheritFromEmbed(t *testing.T) {
	type base struct {
		_ struct{} `mcp-desc:"should not inherit"`
	}
	type wrapped struct {
		base
	}
	if d := collectMcpDesc(&wrapped{}); d != "" {
		t.Fatalf("embedded mcp-desc must not register, got %q", d)
	}
	type ownDesc struct {
		_ struct{} `mcp-desc:"own"`
		base
	}
	if d := collectMcpDesc(&ownDesc{}); d != "own" {
		t.Fatalf("want own desc, got %q", d)
	}
}
