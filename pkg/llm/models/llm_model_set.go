package models

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

const (
	// DefaultModelCatalogURL is the catalog source used when none is
	// configured. Overridable via options.ModelCatalogURL — values without an
	// http:// or https:// prefix are treated as local file paths.
	DefaultModelCatalogURL = "https://www.cloudpods.org/llm-catalog.yaml"
	// DefaultModelCatalogModelScopeURL is the ModelScope mirror of the catalog.
	DefaultModelCatalogModelScopeURL = "https://www.cloudpods.org/model-catalog-modelscope.yaml"
	// DefaultLLMCatalogRefreshInterval is how often the LLM service polls the
	// upstream source in the background. Local file sources are also re-read
	// on each tick (cheap) so an admin edit is picked up without restart.
	DefaultLLMCatalogRefreshInterval = time.Hour
	// catalogFetchTimeout caps a single HTTP fetch attempt.
	catalogFetchTimeout = 30 * time.Second
)

// SLLMModelSetManager is an in-memory store of curated model sets (and their
// deployable specs) loaded from a configurable YAML source. The YAML shape
// matches GPUStack's model catalog (`model_sets` + `draft_models`).
//
// Keys:
//   - sets are keyed by their `name` field (required and unique upstream).
//   - specs don't carry an explicit id in the YAML, so the manager synthesises
//     one at load time from (set name, mode, quantization, backend) and
//     populates LLMModelSpec.SpecId on each spec.
type SLLMModelSetManager struct {
	mu          sync.RWMutex
	sets        []api.LLMModelSet
	setsByName  map[string]*api.LLMModelSet
	specsById   map[string]*specRef
	lastFetched time.Time
	lastErr     error
	source      string
}

// specRef tracks a spec plus the name of the set it belongs to.
type specRef struct {
	SetName string
	Spec    *api.LLMModelSpec
}

var llmModelSetManager *SLLMModelSetManager
var llmModelSetManagerOnce sync.Once

// GetLLMModelSetManager returns the singleton catalog manager.
func GetLLMModelSetManager() *SLLMModelSetManager {
	llmModelSetManagerOnce.Do(func() {
		llmModelSetManager = &SLLMModelSetManager{
			setsByName: map[string]*api.LLMModelSet{},
			specsById:  map[string]*specRef{},
		}
	})
	return llmModelSetManager
}

// Start kicks off the initial load (async, on a goroutine so a slow upstream
// doesn't block service startup) and an optional periodic refresh.
//
// source may be either a local file path or an http(s) URL — same convention
// as GPUStack's --model-catalog-file flag.
func (m *SLLMModelSetManager) Start(ctx context.Context, source string, interval time.Duration) {
	if source == "" {
		source = DefaultModelCatalogURL
	}
	source = resolveModelCatalogSource(source)
	m.mu.Lock()
	m.source = source
	m.mu.Unlock()

	go func() {
		if err := m.Refresh(ctx); err != nil {
			log.Warningf("LLMModelSet: initial load from %s failed: %s", source, err)
		}
	}()

	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.Refresh(ctx); err != nil {
					log.Warningf("LLMModelSet: periodic refresh failed: %s", err)
				}
			}
		}
	}()
}

// Refresh re-loads the configured source and replaces the in-memory cache.
// On error the previous cache is preserved.
func (m *SLLMModelSetManager) Refresh(ctx context.Context) error {
	m.mu.RLock()
	source := m.source
	m.mu.RUnlock()
	if source == "" {
		source = DefaultModelCatalogURL
	}

	body, err := m.loadSource(ctx, source)
	if err != nil {
		m.recordErr(err)
		return err
	}

	doc := api.LLMCatalogDoc{}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		m.recordErr(err)
		return errors.Wrap(err, "parse catalog yaml")
	}

	// Merge OneCloud's bundled Ollama registry into the catalog so the v2 UI
	// can browse + one-click deploy Ollama models alongside the YAML-sourced
	// HF / ModelScope entries.
	doc.ModelSets = append(doc.ModelSets, buildOllamaModelSets(api.OllamaRegistry)...)

	sets := make([]api.LLMModelSet, 0, len(doc.ModelSets))
	setsByName := make(map[string]*api.LLMModelSet, len(doc.ModelSets))
	specsById := make(map[string]*specRef)
	totalSpecs := 0
	for i := range doc.ModelSets {
		s := doc.ModelSets[i]
		if s.Name == "" {
			continue
		}
		s.Id = s.Name
		sets = append(sets, s)
		setRef := &sets[len(sets)-1]
		setsByName[s.Name] = setRef

		// Synthesise stable spec ids and back-populate parent set name.
		usedIds := make(map[string]int)
		for j := range setRef.Specs {
			sp := &setRef.Specs[j]
			sp.ModelSetName = setRef.Name
			base := specBaseId(setRef.Name, sp)
			// Dedup: append -2, -3 … on collision within the set.
			id := base
			if seen := usedIds[base]; seen > 0 {
				id = fmt.Sprintf("%s-%d", base, seen+1)
			}
			usedIds[base]++
			sp.Id = id
			sp.Label = modelSpecLabel(sp, id)
			sp.SpecId = id
			specsById[id] = &specRef{SetName: setRef.Name, Spec: sp}
			totalSpecs++
		}
	}

	m.mu.Lock()
	m.sets = sets
	m.setsByName = setsByName
	m.specsById = specsById
	m.lastFetched = time.Now()
	m.lastErr = nil
	m.mu.Unlock()

	log.Infof("LLMModelSet: refreshed %d sets / %d specs from %s", len(sets), totalSpecs, source)
	return nil
}

func modelSpecLabel(sp *api.LLMModelSpec, id string) string {
	for _, s := range []string{sp.Name, sp.Quantization, sp.Mode, sp.Backend, id} {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return id
}

// specBaseId composes a slug from the spec's identifying fields. Stable across
// loads as long as the upstream YAML keeps the same (set.name, mode,
// quantization, backend) tuple per spec.
func specBaseId(setName string, sp *api.LLMModelSpec) string {
	parts := []string{setName}
	if sp.Mode != "" {
		parts = append(parts, sp.Mode)
	}
	if sp.Quantization != "" {
		parts = append(parts, sp.Quantization)
	}
	if sp.Backend != "" {
		parts = append(parts, sp.Backend)
	}
	return slugify(strings.Join(parts, "-"))
}

// slugify lowercases, replaces every run of non-alphanumeric chars with a
// single dash, and trims leading/trailing dashes.
func slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevDash := true
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			prevDash = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	return out
}

// loadSource reads catalog YAML bytes from either an http(s) URL or a local file path.
func (m *SLLMModelSetManager) loadSource(ctx context.Context, source string) ([]byte, error) {
	lower := strings.ToLower(source)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return m.fetchHTTP(ctx, source)
	}
	if strings.HasPrefix(lower, "file://") {
		source = strings.TrimPrefix(source, "file://")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, errors.Wrapf(err, "read catalog file %s", source)
	}
	return data, nil
}

func (m *SLLMModelSetManager) fetchHTTP(ctx context.Context, url string) ([]byte, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, catalogFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build catalog request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetch catalog")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, errors.Errorf("upstream returned %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (m *SLLMModelSetManager) recordErr(err error) {
	m.mu.Lock()
	m.lastErr = err
	m.mu.Unlock()
}

// ListSets returns sets matching the filter, plus the total count BEFORE
// pagination is applied.
func (m *SLLMModelSetManager) ListSets(input api.LLMModelSetListInput) ([]api.LLMModelSet, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filtered := make([]api.LLMModelSet, 0, len(m.sets))
	for _, s := range m.sets {
		if !modelSetMatches(&s, input) {
			continue
		}
		filtered = append(filtered, s)
	}
	sortModelSets(filtered, input.Sort, input.Direction)
	total := len(filtered)

	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

// GetSet returns the set with the given name (the YAML `name` field).
// Returns a copy so callers can't mutate the cache.
func (m *SLLMModelSetManager) GetSet(name string) (*api.LLMModelSet, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.setsByName[name]
	if !ok {
		return nil, false
	}
	cp := *s
	return &cp, true
}

// ListSpecs returns the specs under one set, identified by the set name.
func (m *SLLMModelSetManager) ListSpecs(setName string) ([]api.LLMModelSpec, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.setsByName[setName]
	if !ok {
		return nil, false
	}
	out := make([]api.LLMModelSpec, len(s.Specs))
	copy(out, s.Specs)
	return out, true
}

// GetSpec returns one spec by its synthetic id (and its parent set's name).
func (m *SLLMModelSetManager) GetSpec(specId string) (*api.LLMModelSpec, string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ref, ok := m.specsById[specId]
	if !ok {
		return nil, "", false
	}
	cp := *ref.Spec
	return &cp, ref.SetName, true
}

// LastFetched reports cache freshness; mostly for debug.
func (m *SLLMModelSetManager) LastFetched() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastFetched
}

func modelSetMatches(s *api.LLMModelSet, input api.LLMModelSetListInput) bool {
	if input.Category != "" && !stringSliceContains(s.Categories, input.Category) {
		return false
	}
	if input.Backend != "" {
		matched := false
		for _, sp := range s.Specs {
			if strings.EqualFold(sp.Backend, input.Backend) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if input.SizeMin != nil && s.Size < *input.SizeMin {
		return false
	}
	if input.SizeMax != nil && s.Size > *input.SizeMax {
		return false
	}
	if input.Search != "" {
		needle := strings.ToLower(input.Search)
		if !strings.Contains(strings.ToLower(s.Name), needle) &&
			!strings.Contains(strings.ToLower(s.Description), needle) &&
			!sliceContainsLower(s.Capabilities, needle) {
			return false
		}
	}
	return true
}

func stringSliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.EqualFold(s, needle) {
			return true
		}
	}
	return false
}

func sliceContainsLower(items []string, needleLower string) bool {
	for _, t := range items {
		if strings.Contains(strings.ToLower(t), needleLower) {
			return true
		}
	}
	return false
}

// buildOllamaModelSets converts OneCloud's bundled `SOllamaRegistry` (used by
// the older /llm-instantmodel/import-community page) into the v2 LLMModelSet
// shape. One ollama model becomes one LLMModelSet; each tag becomes one
// LLMModelSpec. Lets the v2 catalog browse + one-click-deploy Ollama models
// without maintaining a separate data source.
func buildOllamaModelSets(reg api.SOllamaRegistry) []api.LLMModelSet {
	out := make([]api.LLMModelSet, 0, len(reg.Ollama))
	for _, m := range reg.Ollama {
		set := api.LLMModelSet{
			Name:         m.Name,
			Description:  m.Description,
			Icon:         "/static/catalog_icons/ollama.png",
			Categories:   []string{api.LLM_MODEL_CATEGORY_LLM},
			Capabilities: unionOllamaCapabilities(m.Tags),
			Specs:        make([]api.LLMModelSpec, 0, len(m.Tags)),
		}
		for _, t := range m.Tags {
			sp := api.LLMModelSpec{
				Name:         fmt.Sprintf("%s:%s", m.Name, t.Name),
				Quantization: t.Name, // ollama tag (e.g. "8b") shown as quant chip
				Mode:         "standard",
				Source:       "ollama",
				OllamaModel:  m.Name,
				OllamaTag:    t.Name,
				Backend:      "Ollama",
			}
			set.Specs = append(set.Specs, sp)
		}
		out = append(out, set)
	}
	return out
}

// sortModelSets sorts in place by the requested field. Empty `sort` leaves
// the slice in its original (catalog declaration) order.
//
// Supported fields: "downloads", "likes", "size", "name". Direction defaults
// to "desc"; pass "asc" to flip.
func sortModelSets(sets []api.LLMModelSet, sortField, direction string) {
	if sortField == "" {
		return
	}
	asc := strings.EqualFold(direction, "asc")
	less := func(i, j int) bool {
		switch sortField {
		case "downloads":
			if asc {
				return sets[i].Downloads < sets[j].Downloads
			}
			return sets[i].Downloads > sets[j].Downloads
		case "likes":
			if asc {
				return sets[i].Likes < sets[j].Likes
			}
			return sets[i].Likes > sets[j].Likes
		case "size":
			if asc {
				return sets[i].Size < sets[j].Size
			}
			return sets[i].Size > sets[j].Size
		case "name":
			if asc {
				return strings.ToLower(sets[i].Name) < strings.ToLower(sets[j].Name)
			}
			return strings.ToLower(sets[i].Name) > strings.ToLower(sets[j].Name)
		}
		return false
	}
	sort.SliceStable(sets, less)
}

func unionOllamaCapabilities(tags []api.SOllamaTag) []string {
	seen := make(map[string]struct{})
	out := []string{}
	for _, t := range tags {
		for _, c := range t.Capabilities {
			if _, dup := seen[c]; dup {
				continue
			}
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}

var defaultHuggingFaceCatalogURLs = map[string]struct{}{
	DefaultModelCatalogURL:                         {},
	"https://www.cloudpods.org/model-catalog.yaml": {},
}

// resolveModelCatalogSource switches to the ModelScope catalog URL when the
// configured source is a built-in HuggingFace URL and HF is unreachable while
// ModelScope is reachable. Mirrors GPUStack's get_builtin_model_catalog_file.
func resolveModelCatalogSource(source string) string {
	source = strings.TrimSpace(source)
	if _, ok := defaultHuggingFaceCatalogURLs[source]; !ok {
		return source
	}
	hfURL := "https://huggingface.co"
	msURL := "https://www.modelscope.cn"
	if canAccessCatalogURL(hfURL) || !canAccessCatalogURL(msURL) {
		return source
	}
	log.Infof("Cannot access %s, using ModelScope model catalog at %s", hfURL, DefaultModelCatalogModelScopeURL)
	return DefaultModelCatalogModelScopeURL
}

func canAccessCatalogURL(rawURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), catalogFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
