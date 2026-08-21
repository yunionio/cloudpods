package models

import (
	"testing"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestAppendLLMSkuEnvsNilSku(t *testing.T) {
	primary := &computeapi.PodContainerCreateInput{
		ContainerSpec: computeapi.ContainerSpec{
			ContainerSpec: apis.ContainerSpec{
				Envs: []*apis.ContainerKeyValue{{Key: "HF_ENDPOINT", Value: "https://hf.co"}},
			},
		},
	}
	containers := []*computeapi.PodContainerCreateInput{primary}
	AppendLLMSkuEnvs(containers, nil)
	if len(primary.Envs) != 1 || primary.Envs[0].Key != "HF_ENDPOINT" {
		t.Fatalf("expected driver envs unchanged, got %#v", primary.Envs)
	}
}

func TestAppendLLMSkuEnvsEmptySkipped(t *testing.T) {
	primary := &computeapi.PodContainerCreateInput{
		ContainerSpec: computeapi.ContainerSpec{
			ContainerSpec: apis.ContainerSpec{
				Envs: []*apis.ContainerKeyValue{{Key: "HF_ENDPOINT", Value: "https://hf.co"}},
			},
		},
	}
	empty := api.Envs{}
	sku := &SLLMSkuBase{Envs: &empty}
	AppendLLMSkuEnvs([]*computeapi.PodContainerCreateInput{primary}, sku)
	if len(primary.Envs) != 1 || primary.Envs[0].Value != "https://hf.co" {
		t.Fatalf("expected empty sku envs to skip, got %#v", primary.Envs)
	}
}

func TestAppendLLMSkuEnvsInjectsPrimaryAndOverrides(t *testing.T) {
	primary := &computeapi.PodContainerCreateInput{
		ContainerSpec: computeapi.ContainerSpec{
			ContainerSpec: apis.ContainerSpec{
				Envs: []*apis.ContainerKeyValue{
					{Key: "HF_ENDPOINT", Value: "https://hf.co"},
					{Key: "FOO", Value: "bar"},
				},
			},
		},
	}
	sidecar := &computeapi.PodContainerCreateInput{
		ContainerSpec: computeapi.ContainerSpec{
			ContainerSpec: apis.ContainerSpec{
				Envs: []*apis.ContainerKeyValue{{Key: "KEEP", Value: "me"}},
			},
		},
	}
	skuEnvs := api.Envs{
		{Key: "HF_ENDPOINT", Value: "https://mirror.example"},
		{Key: "  ", Value: "ignored"},
		{Key: "NEW_KEY", Value: "new-val"},
	}
	sku := &SLLMSkuBase{Envs: &skuEnvs}
	AppendLLMSkuEnvs([]*computeapi.PodContainerCreateInput{primary, sidecar}, sku)

	if len(primary.Envs) != 3 {
		t.Fatalf("expected 3 primary envs, got %#v", primary.Envs)
	}
	got := map[string]string{}
	for _, e := range primary.Envs {
		got[e.Key] = e.Value
	}
	if got["HF_ENDPOINT"] != "https://mirror.example" {
		t.Fatalf("expected HF_ENDPOINT override, got %#v", got)
	}
	if got["FOO"] != "bar" {
		t.Fatalf("expected FOO preserved, got %#v", got)
	}
	if got["NEW_KEY"] != "new-val" {
		t.Fatalf("expected NEW_KEY injected, got %#v", got)
	}
	if len(sidecar.Envs) != 1 || sidecar.Envs[0].Key != "KEEP" {
		t.Fatalf("expected sidecar envs unchanged, got %#v", sidecar.Envs)
	}
}
