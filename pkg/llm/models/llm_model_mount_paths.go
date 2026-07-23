package models

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

// CollectContainerModelMountPaths returns container-absolute model directories from
// InstantModel.Mounts (effective llm/sku mounted models) and local_path HostPaths.
func CollectContainerModelMountPaths(llm *SLLM, sku *SLLMSku) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}

	ids := GetEffectiveMountedModels(llm, sku)
	if len(ids) > 0 {
		instModels := make(map[string]SInstantModel)
		if err := db.FetchModelObjectsByIds(GetInstantModelManager(), "id", ids, &instModels); err != nil {
			log.Errorf("CollectContainerModelMountPaths FetchModelObjectsByIds: %v", err)
		} else {
			for _, id := range ids {
				inst, ok := instModels[id]
				if !ok {
					continue
				}
				for _, m := range inst.Mounts {
					add(m)
				}
			}
		}
	}

	for _, p := range collectLocalHostPathMountPaths(sku) {
		add(p)
	}

	sort.Strings(out)
	return out
}

func collectLocalHostPathMountPaths(sku *SLLMSku) []string {
	if !SkuHasLocalHostPathModel(sku) {
		return nil
	}
	key := fmt.Sprintf("%d", 0)
	var out []string
	for _, hp := range *sku.HostPaths {
		if hp.IsZero() || hp.Containers == nil {
			continue
		}
		rel, ok := hp.Containers[key]
		if !ok || rel == nil {
			continue
		}
		if p := strings.TrimSpace(rel.MountPath); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// PickContainerModelMountPath selects a model directory from candidates.
// preferred may be an absolute path or a basename (e.g. PreferredModel).
func PickContainerModelMountPath(paths []string, preferred string) string {
	if len(paths) == 0 {
		return ""
	}
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		for _, p := range paths {
			if p == preferred {
				return p
			}
		}
		for _, p := range paths {
			if path.Base(p) == preferred {
				return p
			}
		}
	}
	return paths[0]
}
