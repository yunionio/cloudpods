package models

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestLLMImagesCatalogRefreshLocalYaml(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	yamlPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "website", "static", "llmimages.yaml")
	mgr := &SLLMImagesCatalogManager{
		itemsById: map[string]*api.LLMImagesCatalogItem{},
	}
	mgr.source = yamlPath
	if err := mgr.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	items, total := mgr.ListItems(api.LLMImagesCatalogListInput{})
	if total == 0 {
		t.Fatal("expected catalog items")
	}
	var hasComfyui, hasDifyBundle bool
	for _, item := range items {
		if item.LLMType == "comfyui" {
			hasComfyui = true
			if item.Sku == nil {
				t.Fatal("comfyui sku is nil")
			}
			if len(item.Sku.PortMappings) != 1 {
				t.Fatalf("comfyui port_mappings: got %d want 1", len(item.Sku.PortMappings))
			}
			pm := item.Sku.PortMappings[0]
			if pm.Protocol != "tcp" {
				t.Fatalf("comfyui port protocol: got %q want tcp", pm.Protocol)
			}
			if pm.ContainerPort != 8188 {
				t.Fatalf("comfyui container_port: got %d want 8188", pm.ContainerPort)
			}
		}
		if item.ImportKind == bundleImportKind && item.Name == "dify-1.7.2" {
			hasDifyBundle = true
			if len(item.Images) != 9 {
				t.Fatalf("dify bundle images: got %d want 9", len(item.Images))
			}
			if item.Sku == nil || len(item.Sku.PortMappings) != 1 {
				t.Fatalf("dify sku port_mappings: got %v", item.Sku)
			}
			if item.Sku.PortMappings[0].ContainerPort != 80 {
				t.Fatalf("dify container_port: got %d want 80", item.Sku.PortMappings[0].ContainerPort)
			}
		}
	}
	if !hasComfyui {
		t.Fatal("missing comfyui entry")
	}
	if !hasDifyBundle {
		t.Fatal("missing dify bundle entry")
	}
}
