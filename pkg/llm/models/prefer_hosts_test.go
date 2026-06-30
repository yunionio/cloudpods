package models

import (
	"testing"
)

func TestNormalizePreferHostInputs(t *testing.T) {
	got := normalizePreferHostInputs([]string{" host-1 ", "host-2", "host-1", "", "host-3"})
	want := []string{"host-1", "host-2", "host-3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestGetSkuPreferHosts(t *testing.T) {
	sku := &SLLMSku{PreferHosts: []string{"host-a", "host-b"}}
	got := GetSkuPreferHosts(sku)
	if len(got) != 2 || got[0] != "host-a" || got[1] != "host-b" {
		t.Fatalf("unexpected hosts: %v", got)
	}
}

func TestGetDeploymentPreferHostsPrefersDeployment(t *testing.T) {
	sku := &SLLMSku{PreferHosts: []string{"sku-host"}}
	dep := &SLLMDeployment{PreferHosts: []string{"dep-host"}}
	got := GetDeploymentPreferHosts(dep, sku)
	if len(got) != 1 || got[0] != "dep-host" {
		t.Fatalf("expected deployment hosts, got %v", got)
	}
}

func TestGetDeploymentPreferHostsFallsBackToSku(t *testing.T) {
	sku := &SLLMSku{PreferHosts: []string{"sku-host"}}
	dep := &SLLMDeployment{}
	got := GetDeploymentPreferHosts(dep, sku)
	if len(got) != 1 || got[0] != "sku-host" {
		t.Fatalf("expected sku hosts, got %v", got)
	}
}

func TestSelectPreferHostForInstanceIndexRoundRobin(t *testing.T) {
	hosts := []string{"h1", "h2", "h3"}
	cases := []struct {
		index int
		want  string
	}{
		{0, "h1"},
		{1, "h2"},
		{2, "h3"},
		{3, "h1"},
		{5, "h3"},
	}
	for _, c := range cases {
		if got := SelectPreferHostForInstanceIndex(hosts, c.index); got != c.want {
			t.Fatalf("index %d: got %q, want %q", c.index, got, c.want)
		}
	}
	if got := SelectPreferHostForInstanceIndex(nil, 0); got != "" {
		t.Fatalf("expected empty for nil hosts, got %q", got)
	}
}

func TestValidatePreferHostsSubset(t *testing.T) {
	if err := validatePreferHostsSubset([]string{"h1"}, []string{"h1", "h2"}); err != nil {
		t.Fatalf("expected valid subset, got %v", err)
	}
	if err := validatePreferHostsSubset([]string{"h3"}, []string{"h1", "h2"}); err == nil {
		t.Fatal("expected error for host not on sku")
	}
	if err := validatePreferHostsSubset(nil, []string{"h1"}); err == nil {
		t.Fatal("expected error for empty selected hosts")
	}
}
