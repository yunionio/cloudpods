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

package taskman

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestSTaskGetParams(t *testing.T) {
	newParams := func(pairs map[string]jsonutils.JSONObject) *jsonutils.JSONDict {
		params := jsonutils.NewDict()
		for k, v := range pairs {
			params.Set(k, v)
		}
		return params
	}

	nested := jsonutils.NewDict()
	nested.Set("ip", jsonutils.NewString("10.0.0.1"))
	stages := jsonutils.NewArray(
		jsonutils.NewDict(),
		jsonutils.NewString("on_init"),
	)

	cases := []struct {
		name   string
		params *jsonutils.JSONDict
		want   *jsonutils.JSONDict
	}{
		{
			name:   "nil_params",
			params: nil,
			want:   jsonutils.NewDict(),
		},
		{
			name:   "empty_params",
			params: jsonutils.NewDict(),
			want:   jsonutils.NewDict(),
		},
		{
			name: "keep_public_keys",
			params: newParams(map[string]jsonutils.JSONObject{
				"parent_task_id": jsonutils.NewString("task-1"),
				"auto_start":     jsonutils.JSONTrue,
			}),
			want: newParams(map[string]jsonutils.JSONObject{
				"parent_task_id": jsonutils.NewString("task-1"),
				"auto_start":     jsonutils.JSONTrue,
			}),
		},
		{
			name: "drop_internal_keys",
			params: newParams(map[string]jsonutils.JSONObject{
				"__stages":                stages,
				"__pending_usage__":       jsonutils.NewDict(),
				"__request_context":       jsonutils.NewDict(),
				"__parent_task_notifyurl": jsonutils.NewString("http://notify"),
			}),
			want: jsonutils.NewDict(),
		},
		{
			name: "mixed_keys",
			params: newParams(map[string]jsonutils.JSONObject{
				"desc":     nested,
				"__stages": stages,
				"_private": jsonutils.NewString("keep"),
				"__":       jsonutils.NewString("drop"),
				"guest_id": jsonutils.NewString("g-1"),
			}),
			want: newParams(map[string]jsonutils.JSONObject{
				"desc":     nested,
				"_private": jsonutils.NewString("keep"),
				"guest_id": jsonutils.NewString("g-1"),
			}),
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			task := &STask{}
			task.Params = c.params
			got := task.GetParams()
			if got == nil {
				t.Fatal("GetParams() returned nil, want empty dict")
			}
			if !got.Equals(c.want) {
				t.Fatalf("GetParams() = %s, want %s", got, c.want)
			}
		})
	}
}

func TestSTaskGetParamsDoesNotMutateOriginal(t *testing.T) {
	orig := jsonutils.NewDict()
	orig.Set("guest_id", jsonutils.NewString("g-1"))
	orig.Set("__stages", jsonutils.NewArray(jsonutils.NewString("on_init")))
	nested := jsonutils.NewDict()
	nested.Set("ip", jsonutils.NewString("10.0.0.1"))
	orig.Set("desc", nested)

	task := &STask{STaskBase: STaskBase{Params: orig}}
	got := task.GetParams()
	got.Set("guest_id", jsonutils.NewString("g-2"))
	got.Set("extra", jsonutils.JSONTrue)
	desc, err := got.Get("desc")
	if err != nil {
		t.Fatalf("get desc: %v", err)
	}
	descDict, ok := desc.(*jsonutils.JSONDict)
	if !ok {
		t.Fatalf("desc type %T, want *jsonutils.JSONDict", desc)
	}
	descDict.Set("ip", jsonutils.NewString("10.0.0.2"))

	if got.Contains("__stages") {
		t.Fatal("GetParams() leaked internal key __stages")
	}
	guestId, _ := orig.GetString("guest_id")
	if guestId != "g-1" {
		t.Fatalf("original guest_id = %s, want g-1", guestId)
	}
	if !orig.Contains("__stages") {
		t.Fatal("original __stages was removed")
	}
	origIp, _ := orig.GetString("desc", "ip")
	if origIp != "10.0.0.1" {
		t.Fatalf("original nested ip = %s, want 10.0.0.1", origIp)
	}
}
