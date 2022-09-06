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

package appsrv

import (
	"strings"
	"testing"
)

func TestRadixNode(t *testing.T) {
	root := NewRadix()
	root.Add([]string{}, "root")
	root.Add([]string{"layer1"}, "layer1")
	if root.Add([]string{"layer1"}, "layer1") == nil {
		t.Fatal("add duplicate data should fail")
	}

	root.Add([]string{"layer1", "layer1.1", "layer1.1.1", "layer1.1.1.1"}, "layer1.1.1.1")
	root.Add([]string{"layer1", "layer1.2", "layer1.2.1"}, "layer1.2.1")
	root.Add([]string{"layer1", "<layer1.x>"}, "layer1.*")
	root.Add([]string{"layer1", "layer1.0"}, "layer1.0")
	root.Add([]string{"layer1", "<layer1.x>", "layer1.*.1"}, "layer1.*.1")
	root.Add([]string{"layer1", "<phone_number:^1[0-9-]{10}$>", "layer1.2.1"}, "layer1.*.1_CHINA_MOBILE_REG")
	root.Add([]string{"layer1", "layer1.0", "<phone_number:^1[0-9-]{10}$>"}, "layer1.1.*_CHINA_MOBILE_REG")
	root.Add([]string{"<phone_number:^1[0-9-]{10}$>"}, "phonelayer1")

	f := func(path string, data interface{}) {
		t.Logf("%s %s", path, data)
	}
	root.Walk(f)

	cases := []struct {
		caseName string
		segments []string
		want     string
	}{
		{"case1", []string{"layer1", "layer1.0", "layer2.0"}, "layer1.0"},
		{"case2", []string{"layer1", "layer1.0"}, "layer1.0"},
		{"case3", []string{"layer1", "layer1.1"}, "layer1.*"},
		{"case4", []string{"layer1", "layer1.4"}, "layer1.*"},
		{
			"case5", []string{"layer1", "layer1.1", "layer1.1.1", "layer1.1.1.1", "layer1.1.1.1.1", "layer1.1.1.1.1.1"},
			"layer1.1.1.1",
		},
		{"case6", []string{"layer1", "layer1.2", "layer1.2.1"}, "layer1.2.1"},
		{"case7", []string{"layer1", "layer1.2"}, "layer1.*"},
		{"case8", []string{"layer1", "layer1.3"}, "layer1.*"},
		{"case9", []string{"layer1", "layer1.3", "layer1.*.1"}, "layer1.*.1"},
		{"case10", []string{"layer1", "layer1.3", "layer1.*.1", "layer1.*.1.1"}, "layer1.*.1"},
		{"case11", []string{"layer1", "12345678901", "layer1.2.1"}, "layer1.*.1_CHINA_MOBILE_REG"},
		{"case12", []string{"layer1", "layer1.0", "12345678901"}, "layer1.1.*_CHINA_MOBILE_REG"},
		{"case13", []string{}, "root"},
		{"case14", []string{"12345678900"}, "phonelayer1"},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			params := make(map[string]string)
			got := root.Match(c.segments, params)
			if got.(string) != c.want {
				t.Error("unexpect result:", got, "!=", c.want)
			}
		})
	}
}

func TestRadixMatchParams(t *testing.T) {
	r := NewRadix()
	r.Add([]string{"POST", "clouds", "<cls_action>"}, "classAction")
	r.Add([]string{"POST", "clouds", "<resid>", "sync"}, "objectSyncAction")
	r.Add([]string{"POST", "clouds", "<resid>", "<obj_action>"}, "objectAction")
	r.Add([]string{"POST", "clouds", "<resid2:.*>", "<obj_action2:.*>", "over"}, "objectAction2")
	f := func(path string, data interface{}) {
		t.Logf("%s %s", path, data)
	}
	r.Walk(f)

	cases := []struct {
		in        []string
		out       interface{}
		outParams map[string]string
	}{
		{
			in:  []string{"POST", "clouds", "myid", "sync"},
			out: "objectSyncAction",
			outParams: map[string]string{
				"<resid>": "myid",
			},
		},
		{
			in:  []string{"POST", "clouds", "myid", "start"},
			out: "objectAction",
			outParams: map[string]string{
				"<resid>":      "myid",
				"<obj_action>": "start",
			},
		},
		{
			in:  []string{"POST", "clouds", "start"},
			out: "classAction",
			outParams: map[string]string{
				"<cls_action>": "start",
			},
		},
		{
			in:  []string{"POST", "clouds", "start", "test", "over"},
			out: "objectAction2",
			outParams: map[string]string{
				"<resid2>":      "start",
				"<obj_action2>": "test",
			},
		},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.in, "_"), func(t *testing.T) {
			gotParams := map[string]string{}
			got := r.Match(c.in, gotParams)
			if got != c.out {
				t.Fatalf("want %s, got %s", c.out, c.out)
			}
			if len(gotParams) != len(c.outParams) {
				t.Fatalf("params length mismatch\nwant %#v\ngot %#v",
					c.outParams, gotParams,
				)
			}
			for k, v := range gotParams {
				v1 := c.outParams[k]
				if v == v1 {
					continue
				}
				t.Fatalf("params key %s has mismatch value\nwant %#v\ngot %#v",
					k, c.outParams, gotParams,
				)
			}
		})
	}
}
