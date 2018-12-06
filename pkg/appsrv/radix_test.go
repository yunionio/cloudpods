package appsrv

import (
	"strings"
	"testing"
)

func TestRadixNode(t *testing.T) {
	r := NewRadix()
	r.Add([]string{}, "root")
	r.Add([]string{"layer1"}, "layer1")
	r.Add([]string{"layer1", "layer1.1", "layer1.1.1", "layer1.1.1.1"}, "layer1.1.1.1")
	r.Add([]string{"layer1", "layer1.2", "layer1.2.1"}, "layer1.2.1")
	r.Add([]string{"layer1", "<layer1.x>"}, "layer1.*")
	r.Add([]string{"layer1", "layer1.0"}, "layer1.0")
	r.Add([]string{"layer1", "<layer1.x>", "layer1.*.1"}, "layer1.*.1")
	params := make(map[string]string)
	ret := r.Match([]string{"layer1", "layer1.0"}, params)
	if ret.(string) != "layer1.0" {
		t.Error("0 Unexpect result:", ret, "!= layer1.0")
	}
	ret = r.Match([]string{"layer1", "layer1.1"}, params)
	if ret.(string) != "layer1.*" {
		t.Error("1 Unexpect result:", ret, "!= layer1.*")
	}
	ret = r.Match([]string{"layer1", "layer1.4"}, params)
	if ret.(string) != "layer1.*" {
		t.Error("2 Unexpect result:", ret, "!= layer1.*")
	}
	ret = r.Match([]string{"layer1", "layer1.1", "layer1.1.1", "layer1.1.1.1", "layer1.1.1.1.1", "layer1.1.1.1.1.1"}, params)
	if ret.(string) != "layer1.1.1.1" {
		t.Error("3 Unexpect result:", ret, "!= layer1.1.1.1")
	}
	ret = r.Match([]string{"layer1", "layer1.2", "layer1.2.1"}, params)
	if ret.(string) != "layer1.2.1" {
		t.Error("4 Unexpect result:", ret, "!= layer1.2.1")
	}
	ret = r.Match([]string{"layer1", "layer1.2"}, params)
	if ret.(string) != "layer1.*" {
		t.Error("5 Unexpect result:", ret, "!= layer1.*")
	}
	ret = r.Match([]string{"layer1", "layer1.3"}, params)
	if ret.(string) != "layer1.*" {
		t.Error("6 Unexpect result:", ret, "!= layer1.*")
	}
	ret = r.Match([]string{"layer1", "layer1.3", "layer1.*.1"}, params)
	if ret.(string) != "layer1.*.1" {
		t.Error("7 Unexpect result:", ret, "!= layer1.*.1")
	}
	ret = r.Match([]string{"layer1", "layer1.3", "layer1.*.1", "layer1.*.1.1"}, params)
	if ret.(string) != "layer1.*.1" {
		t.Error("8 Unexpect result:", ret, "!= layer1.*.1")
	}
	ret = r.Match([]string{"layer2"}, params)
	if ret.(string) != "root" {
		t.Error("9 Unexpect result:", ret, "!= root")
	}
	if r.Add([]string{"layer1"}, "layer1") == nil {
		t.Error("10 Add duplicate data should fail")
	}
}

func TestRadixMatchParams(t *testing.T) {
	r := NewRadix()
	r.Add([]string{"POST", "clouds", "<cls_action>"}, "classAction")
	r.Add([]string{"POST", "clouds", "<resid>", "sync"}, "objectSyncAction")
	r.Add([]string{"POST", "clouds", "<resid>", "<obj_action>"}, "objectAction")
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
