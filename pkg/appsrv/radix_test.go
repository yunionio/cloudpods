package appsrv

import (
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

func TestParams(t *testing.T) {
	r := NewRadix()
	r.Add([]string{"POST", "clouds", "<action>"}, "classAction")
	r.Add([]string{"POST", "clouds", "<resid>", "<action>"}, "objectAction")
	params := make(map[string]string)
	ret := r.Match([]string{"POST", "clouds", "id", "sync"}, params)
	t.Logf("match: %s", ret)
	t.Logf("params: %s", params)

	params = make(map[string]string)
	ret = r.Match([]string{"POST", "clouds", "sync"}, params)
	t.Logf("match: %s", ret)
	t.Logf("params: %s", params)
}
