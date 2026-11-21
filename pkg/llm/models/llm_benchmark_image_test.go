package models

import (
	"strings"
	"testing"
)

func TestParseImageRef(t *testing.T) {
	cases := []struct {
		ref       string
		wantName  string
		wantLabel string
	}{
		{"registry.cn-beijing.aliyuncs.com/cloudpods/guidellm:v0.7.0-amd64", "registry.cn-beijing.aliyuncs.com/cloudpods/guidellm", "v0.7.0-amd64"},
		{"localhost:5000/guidellm:v1", "localhost:5000/guidellm", "v1"},
		{"guidellm", "guidellm", "latest"},
		{"registry/guidellm", "registry/guidellm", "latest"},
	}
	for _, c := range cases {
		name, label := parseImageRef(c.ref)
		if name != c.wantName || label != c.wantLabel {
			t.Errorf("parseImageRef(%q) = (%q, %q), want (%q, %q)", c.ref, name, label, c.wantName, c.wantLabel)
		}
	}
}

func TestSanitizeBenchmarkImageLabel(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"v0.7.0-amd64", "v0.7.0-amd64"},
		{"V1.2 Beta", "v1.2-beta"},
		{"a/b:c", "a-b-c"},
		{"", ""},
	}
	for _, c := range cases {
		if got := sanitizeBenchmarkImageLabel(c.in); got != c.want {
			t.Errorf("sanitizeBenchmarkImageLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDefaultBenchmarkImageName(t *testing.T) {
	// deterministic per ref
	a := defaultBenchmarkImageName("registry/guidellm", "v1")
	b := defaultBenchmarkImageName("registry/guidellm", "v1")
	if a != b {
		t.Errorf("name not deterministic: %q != %q", a, b)
	}
	if !strings.HasPrefix(a, "default-benchmark-") {
		t.Errorf("name %q missing prefix", a)
	}
	// different refs produce different names
	if a == defaultBenchmarkImageName("registry/guidellm", "v2") {
		t.Errorf("different refs produced same name %q", a)
	}
	// fits the 128-char name column, even for long refs
	long := defaultBenchmarkImageName("registry/"+strings.Repeat("a", 200), strings.Repeat("b", 200))
	if len(long) > 128 {
		t.Errorf("name length %d exceeds 128: %q", len(long), long)
	}
	// empty sanitized label falls back to the image name fragment
	if got := defaultBenchmarkImageName("registry/guidellm", "---"); !strings.Contains(got, "registry-guidellm") {
		t.Errorf("fallback to image name missing in %q", got)
	}
}
