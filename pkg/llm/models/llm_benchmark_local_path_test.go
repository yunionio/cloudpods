package models

import (
	"strings"
	"testing"

	apis "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestPickLocalPathBenchmarkModelPath(t *testing.T) {
	cases := []struct {
		name    string
		paths   []string
		model   string
		want    string
		wantErr string
	}{
		{
			name:  "exact match",
			paths: []string{"/models/qwen", "/models/embedding"},
			model: "/models/qwen",
			want:  "/models/qwen",
		},
		{
			name:  "basename match",
			paths: []string{"/models/qwen", "/models/embedding"},
			model: "qwen",
			want:  "/models/qwen",
		},
		{
			name:    "multiple paths without match",
			paths:   []string{"/models/qwen", "/models/embedding"},
			model:   "other",
			wantErr: "cannot uniquely resolve tokenizer model",
		},
		{
			name:    "ambiguous basenames",
			paths:   []string{"/a/qwen", "/b/qwen"},
			model:   "qwen",
			wantErr: "multiple mounted models match",
		},
		{
			name:  "single path falls back regardless of model",
			paths: []string{"/models/qwen"},
			model: "other",
			want:  "/models/qwen",
		},
		{
			name:  "empty model with single path",
			paths: []string{"/models/qwen"},
			want:  "/models/qwen",
		},
		{
			name:    "empty model with multiple paths",
			paths:   []string{"/models/qwen", "/models/embedding"},
			wantErr: "cannot uniquely resolve tokenizer model",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := pickLocalPathBenchmarkModelPath(c.paths, c.model)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("got (%q, %v), want error containing %q", got, err, c.wantErr)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("got (%q, %v), want (%q, nil)", got, err, c.want)
			}
		})
	}
}

func localPathSkuForTest(hostPaths api.HostPaths) *SLLMSku {
	sku := &SLLMSku{}
	sku.Source = api.LLM_MODEL_SOURCE_LOCAL_PATH
	sku.LocalPath = "/opt/models"
	sku.HostPaths = &hostPaths
	return sku
}

func TestBuildLocalPathBenchmarkTokenizerMount(t *testing.T) {
	dirType := apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY
	hostPaths := api.HostPaths{
		{
			Type: dirType,
			Path: "/opt/models/qwen",
			Containers: api.ContainerHostPathRelations{
				"0": &api.ContainerHostPathRelation{MountPath: "/models/qwen"},
			},
		},
	}
	sku := localPathSkuForTest(hostPaths)
	if !SkuHasLocalHostPathModel(sku) {
		t.Fatal("fixture sku not recognized as local_path")
	}
	mount, err := buildLocalPathBenchmarkTokenizerMount(&SLLM{}, sku, "qwen")
	if err != nil {
		t.Fatal(err)
	}
	if mount.ImageId != "" || mount.ModelPath != "/models/qwen" {
		t.Fatalf("got %+v", mount)
	}
	if mount.HostPaths == nil || mount.HostPaths.IsZero() {
		t.Fatal("host paths not carried")
	}

	// multiple mount paths without a model match fail
	multi := api.HostPaths{
		{Type: dirType, Path: "/opt/models/qwen", Containers: api.ContainerHostPathRelations{
			"0": &api.ContainerHostPathRelation{MountPath: "/models/qwen"},
		}},
		{Type: dirType, Path: "/opt/models/embedding", Containers: api.ContainerHostPathRelations{
			"0": &api.ContainerHostPathRelation{MountPath: "/models/embedding"},
		}},
	}
	if _, err := buildLocalPathBenchmarkTokenizerMount(&SLLM{}, localPathSkuForTest(multi), "other"); err == nil {
		t.Fatal("expected cannot uniquely resolve error")
	}

	// host path without a container mount fails
	empty := api.HostPaths{{Type: dirType, Path: "/opt/models/qwen"}}
	if _, err := buildLocalPathBenchmarkTokenizerMount(&SLLM{}, localPathSkuForTest(empty), "qwen"); err == nil {
		t.Fatal("expected missing mount path error")
	}
}
