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
	"encoding/json"
	"strings"
	"testing"

	_ "yunion.io/x/onecloud/cmd/climc/shell/compute"
	_ "yunion.io/x/onecloud/cmd/climc/shell/image"

	"yunion.io/x/onecloud/cmd/climc/shell"
)

func TestBuildInputSchemaUsesMcpTags(t *testing.T) {
	var cmd shell.CMD
	found := false
	for _, c := range shell.CommandTable {
		if c.Command == "server-create" {
			cmd = c
			found = true
			break
		}
	}
	if !found {
		t.Fatal("server-create not registered")
	}
	raw, err := buildInputSchema(cmd)
	if err != nil {
		t.Fatalf("buildInputSchema: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	props := schema["properties"].(map[string]interface{})
	for _, key := range []string{"name", "net", "disk", "ncpu", "mem-spec", "prefer-region", "hypervisor", "cdrom"} {
		if _, ok := props[key]; !ok {
			t.Errorf("expected mcp-tagged property %q in schema, got keys=%v", key, propKeys(props))
		}
	}
	// kickstart 等未打 mcp tag 的字段不应出现
	for _, key := range []string{"kickstart-os-type", "user-data-file", "fake-create"} {
		if _, ok := props[key]; ok {
			t.Errorf("unexpected untagged property %q in schema", key)
		}
	}
	req, _ := schema["required"].([]interface{})
	reqSet := map[string]bool{}
	for _, r := range req {
		reqSet[r.(string)] = true
	}
	if !reqSet["name"] || reqSet["net"] {
		t.Errorf("expected name required and net optional, got %v", req)
	}

	desc := buildDescription(cmd)
	if !strings.Contains(desc, "创建虚拟机的最终动作") {
		t.Errorf("expected mcp-desc in tool description, got %q", desc)
	}
}

func propKeys(props map[string]interface{}) []string {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	return keys
}
