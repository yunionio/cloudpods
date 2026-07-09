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

package openai

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestDecodeNamespaceToolCall(t *testing.T) {
	toolMap := CodexToolMap{
		"myns": {Kind: CodexToolNestedOneOf, Namespace: "myns", Actions: []string{"search"}},
	}
	ns, name, args := DecodeNamespaceToolCall(toolMap, "myns", `{"action":"search","q":"hello"}`)
	if ns != "myns" || name != "search" {
		t.Fatalf("ns=%q name=%q", ns, name)
	}
	if args != `{"q":"hello"}` {
		t.Fatalf("args=%q", args)
	}
}

func TestNamespacedToolName(t *testing.T) {
	if got := NamespacedToolName("ns", "tool"); got != "ns_tool" {
		t.Fatalf("got %q", got)
	}
}

func TestFlattenResponsesFunctionToolParametersObject(t *testing.T) {
	arr := jsonutils.NewArray()
	fn := jsonutils.NewDict()
	fn.Set("type", jsonutils.NewString("function"))
	fn.Set("name", jsonutils.NewString("exec_command"))
	params := jsonutils.NewDict()
	params.Set("type", jsonutils.NewString("object"))
	props := jsonutils.NewDict()
	cmd := jsonutils.NewDict()
	cmd.Set("type", jsonutils.NewString("string"))
	props.Set("cmd", cmd)
	params.Set("properties", props)
	params.Set("required", jsonutils.NewArray(jsonutils.NewString("cmd")))
	fn.Set("parameters", params)
	arr.Add(fn)

	tools, _, err := FlattenResponsesTools(arr)
	if err != nil {
		t.Fatal(err)
	}
	if tools == nil || tools.Length() != 1 {
		t.Fatalf("tools = %v", tools)
	}
	toolWrap, _ := tools.GetAt(0)
	fnOut, _ := toolWrap.Get("function")
	paramsOut, _ := fnOut.Get("parameters")
	if _, ok := paramsOut.(*jsonutils.JSONDict); !ok {
		t.Fatalf("parameters should be object, got %T value=%v", paramsOut, paramsOut)
	}
}
