package models

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestLLMModelSetSizeInB(t *testing.T) {
	cases := []struct {
		name string
		set  *api.LLMModelSet
		want float64
	}{
		{name: "nil set", set: nil, want: 0},
		{name: "billions default", set: &api.LLMModelSet{Size: 27}, want: 27},
		{name: "billions explicit", set: &api.LLMModelSet{Size: 8, SizeUnit: "B"}, want: 8},
		{name: "trillions", set: &api.LLMModelSet{Size: 2.8, SizeUnit: "T"}, want: 2800},
		{name: "millions", set: &api.LLMModelSet{Size: 809, SizeUnit: "M"}, want: 0.809},
		{name: "lowercase unit", set: &api.LLMModelSet{Size: 1.6, SizeUnit: "t"}, want: 1600},
		{name: "padded unit", set: &api.LLMModelSet{Size: 320, SizeUnit: " B "}, want: 320},
		{name: "unknown unit falls back to billions", set: &api.LLMModelSet{Size: 5, SizeUnit: "X"}, want: 5},
		{name: "zero size", set: &api.LLMModelSet{SizeUnit: "T"}, want: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.set.SizeInB()
			if math.Abs(got-c.want) > 1e-9 {
				t.Fatalf("SizeInB() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSortModelSetsDefaultOrder(t *testing.T) {
	names := func(sets []api.LLMModelSet) []string {
		out := make([]string, len(sets))
		for i := range sets {
			out[i] = sets[i].Name
		}
		return out
	}

	base := []api.LLMModelSet{
		{Name: "old", ReleaseDate: "2023-11-06"},
		{Name: "undated"},
		{Name: "newest", ReleaseDate: "2026-02-03"},
		{Name: "middle", ReleaseDate: "2025-04-19"},
	}

	t.Run("default sorts newest first", func(t *testing.T) {
		sets := append([]api.LLMModelSet(nil), base...)
		sortModelSets(sets, "", "")
		got := names(sets)
		want := []string{"newest", "middle", "old", "undated"}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("default order = %v, want %v", got, want)
			}
		}
	})

	t.Run("explicit release_date honours direction", func(t *testing.T) {
		sets := append([]api.LLMModelSet(nil), base...)
		sortModelSets(sets, sortFieldReleaseDate, "asc")
		got := names(sets)
		want := []string{"old", "middle", "newest", "undated"}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("asc order = %v, want %v", got, want)
			}
		}
	})

	t.Run("undated sets stay last in both directions", func(t *testing.T) {
		sets := append([]api.LLMModelSet(nil), base...)
		sortModelSets(sets, sortFieldReleaseDate, "desc")
		if last := names(sets)[len(sets)-1]; last != "undated" {
			t.Fatalf("desc last = %q, want undated", last)
		}
	})

	t.Run("other sort fields are unaffected", func(t *testing.T) {
		sets := []api.LLMModelSet{
			{Name: "a", Downloads: 10, ReleaseDate: "2026-01-01"},
			{Name: "b", Downloads: 99, ReleaseDate: "2023-01-01"},
		}
		sortModelSets(sets, "downloads", "desc")
		if got := names(sets); got[0] != "b" {
			t.Fatalf("downloads order = %v, want b first", got)
		}
	})
}

// catalogPath resolves a catalog YAML for the parsing tests. The catalog is
// published from the website repo rather than embedded here, so the tests skip
// when no file can be found instead of failing.
func catalogPath(t *testing.T, envKey, fallbackName string) string {
	t.Helper()
	if p := os.Getenv(envKey); p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s=%q: %v", envKey, p, err)
		}
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home dir: %v", err)
	}
	p := filepath.Join(home, "code", "go", "src", "yunion.io", "x", "website", "static", fallbackName)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("catalog %s not found (set %s to point at it)", p, envKey)
	}
	return p
}

func loadCatalogDoc(t *testing.T, path string) api.LLMCatalogDoc {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc := api.LLMCatalogDoc{}
	// Must use yaml.v2: ScalarOrList.UnmarshalYAML implements the v2 interface.
	if err := yaml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

func TestLLMModelCatalogFiles(t *testing.T) {
	catalogs := []struct {
		name       string
		envKey     string
		fileName   string
		wantSource string
	}{
		{
			name:       "huggingface",
			envKey:     "LLM_MODEL_CATALOG_PATH",
			fileName:   "model-catalog.yaml",
			wantSource: api.LLM_MODEL_SOURCE_HUGGINGFACE,
		},
		{
			name:       "modelscope",
			envKey:     "LLM_MODEL_CATALOG_MODELSCOPE_PATH",
			fileName:   "model-catalog-modelscope.yaml",
			wantSource: api.LLM_MODEL_SOURCE_MODEL_SCOPE,
		},
	}
	for _, cat := range catalogs {
		t.Run(cat.name, func(t *testing.T) {
			path := catalogPath(t, cat.envKey, cat.fileName)
			doc := loadCatalogDoc(t, path)
			if len(doc.ModelSets) == 0 {
				t.Fatalf("%s contains no model_sets", path)
			}

			seenSets := make(map[string]struct{}, len(doc.ModelSets))
			for i := range doc.ModelSets {
				s := &doc.ModelSets[i]
				if s.Name == "" {
					t.Fatalf("spec set #%d has empty name", i)
				}
				if _, dup := seenSets[s.Name]; dup {
					t.Fatalf("duplicate model set name %q", s.Name)
				}
				seenSets[s.Name] = struct{}{}
				if len(s.Specs) == 0 {
					t.Fatalf("model set %q has no specs", s.Name)
				}
				for _, c := range s.Categories {
					if !api.LLM_MODEL_CATEGORIES.Has(c) {
						t.Errorf("model set %q has unknown category %q", s.Name, c)
					}
				}
				for j := range s.Specs {
					checkCatalogSpec(t, s.Name, cat.wantSource, &s.Specs[j])
				}
			}
		})
	}
}

func checkCatalogSpec(t *testing.T, setName, wantSource string, sp *api.LLMModelSpec) {
	t.Helper()
	where := setName + " spec #" + sp.Mode + "/" + sp.Quantization
	switch sp.Source {
	case wantSource:
	case api.LLM_MODEL_SOURCE_LOCAL_PATH, "ollama":
		// Supported by the loader but not used by the published catalog.
	default:
		t.Errorf("%s: unexpected source %q (want %q)", where, sp.Source, wantSource)
	}
	switch sp.Source {
	case api.LLM_MODEL_SOURCE_HUGGINGFACE:
		if sp.HuggingfaceRepoId == "" {
			t.Errorf("%s: huggingface source without huggingface_repo_id", where)
		}
	case api.LLM_MODEL_SOURCE_MODEL_SCOPE:
		if sp.ModelScopeModelId == "" {
			t.Errorf("%s: model_scope source without model_scope_model_id", where)
		}
	}
	// The published catalog is limited to NVIDIA/AMD-deployable vLLM/SGLang
	// specs: MindIE, VoxBox and Ascend builds are intentionally omitted.
	if sp.Backend != "vLLM" && sp.Backend != "SGLang" {
		t.Errorf("%s: unexpected backend %q", where, sp.Backend)
	}
	if sp.GpuFilters != nil {
		for _, v := range sp.GpuFilters.Vendor {
			if v == "ascend" || v == "hygon" {
				t.Errorf("%s: unexpected gpu vendor %q", where, v)
			}
		}
	}
	if sp.ImageName != "" {
		t.Errorf("%s: spec requires a dedicated image (%q), not supported by the catalog", where, sp.ImageName)
	}
}

func TestLLMModelSetManagerLoadsCatalog(t *testing.T) {
	path := catalogPath(t, "LLM_MODEL_CATALOG_PATH", "model-catalog.yaml")
	doc := loadCatalogDoc(t, path)

	m := &SLLMModelSetManager{
		slots:            map[string]*catalogSlot{},
		configuredSource: path,
	}
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh from %s: %v", path, err)
	}

	sets, total := m.ListSets(api.LLMModelSetListInput{Limit: 1000})
	if total == 0 || len(sets) != total {
		t.Fatalf("ListSets returned %d of %d", len(sets), total)
	}
	// The manager appends the bundled Ollama registry; the YAML sets must all
	// survive the load.
	if total < len(doc.ModelSets) {
		t.Fatalf("loaded %d sets, want at least the %d YAML sets", total, len(doc.ModelSets))
	}

	seenIds := make(map[string]struct{})
	for _, s := range sets {
		if s.Id != s.Name {
			t.Errorf("set %q: id = %q, want name", s.Name, s.Id)
		}
		if _, ok := m.GetSet(s.Name, ""); !ok {
			t.Errorf("GetSet(%q) not found", s.Name)
		}
		specs, ok := m.ListSpecs(s.Name, "")
		if !ok || len(specs) != len(s.Specs) {
			t.Fatalf("ListSpecs(%q) returned %d specs, want %d", s.Name, len(specs), len(s.Specs))
		}
		for _, sp := range specs {
			// ids must be globally unique or specsById silently overwrites.
			if _, dup := seenIds[sp.Id]; dup {
				t.Errorf("duplicate spec id %q", sp.Id)
			}
			seenIds[sp.Id] = struct{}{}
			got, setName, ok := m.GetSpec(sp.Id, "")
			if !ok {
				t.Errorf("GetSpec(%q) not found", sp.Id)
				continue
			}
			if setName != s.Name || got.Id != sp.Id || got.ModelSetName != s.Name {
				t.Errorf("GetSpec(%q) = set %q / spec %q, want set %q", sp.Id, setName, got.Id, s.Name)
			}
			if got.Label == "" {
				t.Errorf("spec %q has empty label", sp.Id)
			}
		}
	}

	// Size filtering must respect size_unit (billions-normalised).
	min := 1000.0
	big, bigTotal := m.ListSets(api.LLMModelSetListInput{SizeMin: &min, Limit: 1000})
	if bigTotal == 0 {
		t.Fatalf("expected at least one set with >= %vB parameters", min)
	}
	for _, s := range big {
		if s.SizeInB() < min {
			t.Errorf("set %q (size %v %s = %vB) passed size_min %v", s.Name, s.Size, s.SizeUnit, s.SizeInB(), min)
		}
	}

	// Custom / single-file catalog: no dual-source picker.
	if got := m.AvailableSources(); len(got) != 0 {
		t.Fatalf("AvailableSources() = %v, want empty for custom catalog", got)
	}
}

func TestLLMModelSetManagerDualCatalog(t *testing.T) {
	hfPath := catalogPath(t, "LLM_MODEL_CATALOG_PATH", "model-catalog.yaml")
	msPath := catalogPath(t, "LLM_MODEL_CATALOG_MODELSCOPE_PATH", "model-catalog-modelscope.yaml")

	m := &SLLMModelSetManager{
		slots:            map[string]*catalogSlot{},
		dualMode:         true,
		configuredSource: DefaultModelCatalogURL,
		hfReachable:      true,
		msReachable:      true,
	}
	if err := m.refreshSlot(context.Background(), api.LLM_MODEL_SOURCE_HUGGINGFACE, hfPath); err != nil {
		t.Fatalf("refresh huggingface slot: %v", err)
	}
	if err := m.refreshSlot(context.Background(), api.LLM_MODEL_SOURCE_MODEL_SCOPE, msPath); err != nil {
		t.Fatalf("refresh model_scope slot: %v", err)
	}

	sources := m.AvailableSources()
	if len(sources) != 2 {
		t.Fatalf("AvailableSources() = %v, want both sources", sources)
	}

	hfSets, hfTotal := m.ListSets(api.LLMModelSetListInput{
		Source: api.LLM_MODEL_SOURCE_HUGGINGFACE,
		Limit:  1000,
	})
	msSets, msTotal := m.ListSets(api.LLMModelSetListInput{
		Source: api.LLM_MODEL_SOURCE_MODEL_SCOPE,
		Limit:  1000,
	})
	if hfTotal == 0 || msTotal == 0 {
		t.Fatalf("empty catalog: hf=%d ms=%d", hfTotal, msTotal)
	}
	if hfTotal != msTotal {
		t.Fatalf("set count mismatch: hf=%d ms=%d", hfTotal, msTotal)
	}

	// Default (no source) prefers huggingface when both are available.
	defaultSets, _ := m.ListSets(api.LLMModelSetListInput{Limit: 5})
	if len(defaultSets) == 0 {
		t.Fatal("default ListSets returned no sets")
	}
	for _, s := range defaultSets {
		for _, sp := range s.Specs {
			if sp.Source == "ollama" {
				continue
			}
			if sp.Source != api.LLM_MODEL_SOURCE_HUGGINGFACE {
				t.Fatalf("default source spec %q has source %q, want huggingface", sp.Id, sp.Source)
			}
		}
	}

	// Explicit model_scope returns ModelScope repo fields.
	var foundMS bool
	for _, s := range msSets {
		for _, sp := range s.Specs {
			if sp.Source != api.LLM_MODEL_SOURCE_MODEL_SCOPE {
				continue
			}
			foundMS = true
			if sp.ModelScopeModelId == "" {
				t.Fatalf("model_scope spec %q missing model_scope_model_id", sp.Id)
			}
			got, _, ok := m.GetSpec(sp.Id, api.LLM_MODEL_SOURCE_MODEL_SCOPE)
			if !ok || got.ModelScopeModelId != sp.ModelScopeModelId {
				t.Fatalf("GetSpec(%q, model_scope) mismatch", sp.Id)
			}
			break
		}
		if foundMS {
			break
		}
	}
	if !foundMS {
		t.Fatal("no model_scope specs found")
	}

	// Only ModelScope reachable → AvailableSources is just model_scope.
	m.mu.Lock()
	m.hfReachable = false
	m.msReachable = true
	m.mu.Unlock()
	onlyMS := m.AvailableSources()
	if len(onlyMS) != 1 || onlyMS[0] != api.LLM_MODEL_SOURCE_MODEL_SCOPE {
		t.Fatalf("AvailableSources with only MS = %v", onlyMS)
	}
	fallback, _ := m.ListSets(api.LLMModelSetListInput{Limit: 5})
	if len(fallback) == 0 {
		t.Fatal("expected model_scope fallback sets")
	}
	for _, s := range fallback {
		for _, sp := range s.Specs {
			if sp.Source == "ollama" {
				continue
			}
			if sp.Source != api.LLM_MODEL_SOURCE_MODEL_SCOPE {
				t.Fatalf("MS-only default spec source = %q", sp.Source)
			}
		}
	}

	_ = hfSets
}
