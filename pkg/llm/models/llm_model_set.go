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
	// Must match the default of options.ModelCatalogURL and the file published
	// from the website repo's static/ directory.
	DefaultModelCatalogURL = "https://www.cloudpods.org/model-catalog.yaml"
	// DefaultModelCatalogModelScopeURL is the ModelScope mirror of the catalog.
	DefaultModelCatalogModelScopeURL = "https://www.cloudpods.org/model-catalog-modelscope.yaml"
	// legacyModelCatalogURL is the previous default, which is no longer
	// published (404). Kept so deployments that still point at it get the
	// dual-catalog / ModelScope path instead of a hard failure.
	legacyModelCatalogURL = "https://www.cloudpods.org/llm-catalog.yaml"
	// DefaultLLMCatalogRefreshInterval is how often the LLM service polls the
	// upstream source in the background. Local file sources are also re-read
	// on each tick (cheap) so an admin edit is picked up without restart.
	DefaultLLMCatalogRefreshInterval = time.Hour
	// catalogFetchTimeout caps a single HTTP fetch attempt.
	catalogFetchTimeout = 30 * time.Second

	catalogHubHuggingFace = "https://huggingface.co"
	catalogHubModelScope  = "https://www.modelscope.cn"

	// catalogSlotDefault is the sole slot key used for a custom ModelCatalogURL.
	catalogSlotDefault = "default"
)

// catalogSlot holds one in-memory catalog view (HuggingFace, ModelScope, or
// a custom single-file catalog).
type catalogSlot struct {
	sets        []api.LLMModelSet
	setsByName  map[string]*api.LLMModelSet
	specsById   map[string]*specRef
	url         string
	lastFetched time.Time
	lastErr     error
}

// SLLMModelSetManager is an in-memory store of curated model sets (and their
// deployable specs) loaded from a configurable YAML source. The YAML shape
// matches GPUStack's model catalog (`model_sets` + `draft_models`).
//
// When the configured source is a built-in HuggingFace catalog URL, the
// manager loads both the HuggingFace and ModelScope YAML mirrors into separate
// slots so the UI can let the user pick a download source. Custom URLs keep a
// single slot.
//
// Keys:
//   - sets are keyed by their `name` field (required and unique upstream).
//   - specs don't carry an explicit id in the YAML, so the manager synthesises
//     one at load time from (set name, mode, quantization, backend) and
//     populates LLMModelSpec.SpecId on each spec.
type SLLMModelSetManager struct {
	mu               sync.RWMutex
	slots            map[string]*catalogSlot
	dualMode         bool
	configuredSource string
	hfReachable      bool
	msReachable      bool
	lastErr          error
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
			slots: map[string]*catalogSlot{},
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
	m.mu.Lock()
	m.configuredSource = source
	m.dualMode = isBuiltinHuggingFaceCatalogURL(source)
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

// Refresh re-loads the configured source(s) and replaces the in-memory cache.
// On error the previous cache for a slot is preserved. Hub reachability is
// re-probed and cached for AvailableSources.
func (m *SLLMModelSetManager) Refresh(ctx context.Context) error {
	m.mu.RLock()
	configured := m.configuredSource
	dual := m.dualMode
	m.mu.RUnlock()

	if configured == "" {
		configured = DefaultModelCatalogURL
		dual = isBuiltinHuggingFaceCatalogURL(configured)
	}

	hfReachable := canAccessCatalogURL(catalogHubHuggingFace)
	msReachable := canAccessCatalogURL(catalogHubModelScope)

	if dual {
		var firstErr error
		okCount := 0
		if err := m.refreshSlot(ctx, api.LLM_MODEL_SOURCE_HUGGINGFACE, DefaultModelCatalogURL); err != nil {
			log.Warningf("LLMModelSet: refresh huggingface catalog failed: %s", err)
			firstErr = err
		} else {
			okCount++
		}
		if err := m.refreshSlot(ctx, api.LLM_MODEL_SOURCE_MODEL_SCOPE, DefaultModelCatalogModelScopeURL); err != nil {
			log.Warningf("LLMModelSet: refresh model_scope catalog failed: %s", err)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			okCount++
		}
		m.mu.Lock()
		m.configuredSource = configured
		m.dualMode = true
		m.hfReachable = hfReachable
		m.msReachable = msReachable
		if okCount == 0 {
			m.lastErr = firstErr
			m.mu.Unlock()
			return firstErr
		}
		m.lastErr = nil
		m.mu.Unlock()
		return nil
	}

	if err := m.refreshSlot(ctx, catalogSlotDefault, configured); err != nil {
		m.mu.Lock()
		m.configuredSource = configured
		m.dualMode = false
		m.hfReachable = hfReachable
		m.msReachable = msReachable
		m.lastErr = err
		m.mu.Unlock()
		return err
	}
	m.mu.Lock()
	m.configuredSource = configured
	m.dualMode = false
	m.hfReachable = hfReachable
	m.msReachable = msReachable
	m.lastErr = nil
	m.mu.Unlock()
	return nil
}

func (m *SLLMModelSetManager) refreshSlot(ctx context.Context, key, source string) error {
	body, err := m.loadSource(ctx, source)
	if err != nil {
		m.recordSlotErr(key, source, err)
		return err
	}

	doc := api.LLMCatalogDoc{}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		err = errors.Wrap(err, "parse catalog yaml")
		m.recordSlotErr(key, source, err)
		return err
	}

	// Merge OneCloud's bundled Ollama registry into the catalog so the v2 UI
	// can browse + one-click deploy Ollama models alongside the YAML-sourced
	// HF / ModelScope entries.
	doc.ModelSets = append(doc.ModelSets, buildOllamaModelSets(api.OllamaRegistry)...)

	slot, totalSpecs := buildCatalogSlot(doc, source)

	m.mu.Lock()
	if m.slots == nil {
		m.slots = map[string]*catalogSlot{}
	}
	m.slots[key] = slot
	m.mu.Unlock()

	log.Infof("LLMModelSet: refreshed %d sets / %d specs from %s (slot=%s)", len(slot.sets), totalSpecs, source, key)
	return nil
}

func buildCatalogSlot(doc api.LLMCatalogDoc, source string) (*catalogSlot, int) {
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
		s.ParameterSizeB = s.SizeInB()
		sets = append(sets, s)
		setRef := &sets[len(sets)-1]
		setsByName[s.Name] = setRef

		usedIds := make(map[string]int)
		for j := range setRef.Specs {
			sp := &setRef.Specs[j]
			sp.ModelSetName = setRef.Name
			base := specBaseId(setRef.Name, sp)
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
	return &catalogSlot{
		sets:        sets,
		setsByName:  setsByName,
		specsById:   specsById,
		url:         source,
		lastFetched: time.Now(),
	}, totalSpecs
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

func (m *SLLMModelSetManager) recordSlotErr(key, source string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.slots == nil {
		m.slots = map[string]*catalogSlot{}
	}
	slot := m.slots[key]
	if slot == nil {
		slot = &catalogSlot{
			setsByName: map[string]*api.LLMModelSet{},
			specsById:  map[string]*specRef{},
			url:        source,
		}
		m.slots[key] = slot
	}
	slot.lastErr = err
	m.lastErr = err
}

// AvailableSources returns catalog download sources the UI may offer. Empty
// when dual-mode is off (custom ModelCatalogURL). When both hubs are
// reachable, both sources are returned; otherwise only the reachable side.
// If neither hub is reachable but catalogs are loaded, falls back to the
// HuggingFace slot (matching the historical default preference).
func (m *SLLMModelSetManager) AvailableSources() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.availableSourcesLocked()
}

func (m *SLLMModelSetManager) availableSourcesLocked() []string {
	if !m.dualMode {
		return nil
	}
	var out []string
	if m.hfReachable && m.slotHasSets(api.LLM_MODEL_SOURCE_HUGGINGFACE) {
		out = append(out, api.LLM_MODEL_SOURCE_HUGGINGFACE)
	}
	if m.msReachable && m.slotHasSets(api.LLM_MODEL_SOURCE_MODEL_SCOPE) {
		out = append(out, api.LLM_MODEL_SOURCE_MODEL_SCOPE)
	}
	if len(out) > 0 {
		return out
	}
	// Neither hub reachable: still expose a loaded catalog so the UI works.
	if m.slotHasSets(api.LLM_MODEL_SOURCE_HUGGINGFACE) {
		return []string{api.LLM_MODEL_SOURCE_HUGGINGFACE}
	}
	if m.slotHasSets(api.LLM_MODEL_SOURCE_MODEL_SCOPE) {
		return []string{api.LLM_MODEL_SOURCE_MODEL_SCOPE}
	}
	return nil
}

func (m *SLLMModelSetManager) slotHasSets(key string) bool {
	slot := m.slots[key]
	return slot != nil && len(slot.sets) > 0
}

// resolveSlotKey picks which catalog slot to serve. Must be called with m.mu held.
func (m *SLLMModelSetManager) resolveSlotKey(requested string) string {
	if !m.dualMode {
		return catalogSlotDefault
	}
	available := m.availableSourcesLocked()
	req := strings.TrimSpace(requested)
	if req != "" {
		for _, s := range available {
			if s == req {
				return req
			}
		}
		// Requested source has a loaded slot even if currently marked
		// unreachable — honour an explicit request when data exists.
		if m.slotHasSets(req) {
			return req
		}
	}
	for _, preferred := range []string{api.LLM_MODEL_SOURCE_HUGGINGFACE, api.LLM_MODEL_SOURCE_MODEL_SCOPE} {
		for _, s := range available {
			if s == preferred {
				return preferred
			}
		}
	}
	if len(available) > 0 {
		return available[0]
	}
	if m.slotHasSets(api.LLM_MODEL_SOURCE_HUGGINGFACE) {
		return api.LLM_MODEL_SOURCE_HUGGINGFACE
	}
	if m.slotHasSets(api.LLM_MODEL_SOURCE_MODEL_SCOPE) {
		return api.LLM_MODEL_SOURCE_MODEL_SCOPE
	}
	return catalogSlotDefault
}

func (m *SLLMModelSetManager) slotLocked(requested string) *catalogSlot {
	key := m.resolveSlotKey(requested)
	return m.slots[key]
}

// ListSets returns sets matching the filter, plus the total count BEFORE
// pagination is applied. input.Source selects the HuggingFace or ModelScope
// catalog when dual-mode is active.
func (m *SLLMModelSetManager) ListSets(input api.LLMModelSetListInput) ([]api.LLMModelSet, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slot := m.slotLocked(input.Source)
	if slot == nil {
		return nil, 0
	}

	filtered := make([]api.LLMModelSet, 0, len(slot.sets))
	for _, s := range slot.sets {
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
// source selects the catalog slot in dual-mode; empty uses the default.
// Returns a copy so callers can't mutate the cache.
func (m *SLLMModelSetManager) GetSet(name, source string) (*api.LLMModelSet, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	slot := m.slotLocked(source)
	if slot == nil {
		return nil, false
	}
	s, ok := slot.setsByName[name]
	if !ok {
		return nil, false
	}
	cp := *s
	return &cp, true
}

// ListSpecs returns the specs under one set, identified by the set name.
func (m *SLLMModelSetManager) ListSpecs(setName, source string) ([]api.LLMModelSpec, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	slot := m.slotLocked(source)
	if slot == nil {
		return nil, false
	}
	s, ok := slot.setsByName[setName]
	if !ok {
		return nil, false
	}
	out := make([]api.LLMModelSpec, len(s.Specs))
	copy(out, s.Specs)
	return out, true
}

// GetSpec returns one spec by its synthetic id (and its parent set's name).
// When source is empty in dual-mode, the default slot is tried first, then the
// other slots (SpecIds collide across HF/MS mirrors).
func (m *SLLMModelSetManager) GetSpec(specId, source string) (*api.LLMModelSpec, string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if source != "" || !m.dualMode {
		slot := m.slotLocked(source)
		if slot == nil {
			return nil, "", false
		}
		ref, ok := slot.specsById[specId]
		if !ok {
			return nil, "", false
		}
		cp := *ref.Spec
		return &cp, ref.SetName, true
	}
	// Prefer the resolved default slot, then walk remaining slots.
	primary := m.resolveSlotKey("")
	order := []string{primary}
	for _, key := range []string{api.LLM_MODEL_SOURCE_HUGGINGFACE, api.LLM_MODEL_SOURCE_MODEL_SCOPE} {
		if key != primary {
			order = append(order, key)
		}
	}
	for _, key := range order {
		slot := m.slots[key]
		if slot == nil {
			continue
		}
		ref, ok := slot.specsById[specId]
		if !ok {
			continue
		}
		cp := *ref.Spec
		return &cp, ref.SetName, true
	}
	return nil, "", false
}

// LastFetched reports cache freshness of the active/default slot; mostly for debug.
func (m *SLLMModelSetManager) LastFetched() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	slot := m.slotLocked("")
	if slot == nil {
		return time.Time{}
	}
	return slot.lastFetched
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
	if input.SizeMin != nil && s.SizeInB() < *input.SizeMin {
		return false
	}
	if input.SizeMax != nil && s.SizeInB() > *input.SizeMax {
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

// sortFieldReleaseDate is the default model-set ordering: newest model first.
const sortFieldReleaseDate = "release_date"

// sortModelSets sorts in place by the requested field. An empty `sort` orders
// by release date, newest first, so the catalog opens on the latest models.
//
// Supported fields: "release_date", "downloads", "likes", "size", "name".
// Direction defaults to "desc"; pass "asc" to flip. Sets without a release date
// always sink to the end, regardless of direction.
func sortModelSets(sets []api.LLMModelSet, sortField, direction string) {
	if sortField == "" {
		sortModelSets(sets, sortFieldReleaseDate, "desc")
		return
	}
	asc := strings.EqualFold(direction, "asc")
	less := func(i, j int) bool {
		switch sortField {
		case sortFieldReleaseDate:
			// Catalog dates are ISO-8601 (YYYY-MM-DD), so lexical order is
			// chronological. Undated sets (e.g. bundled Ollama models) go last.
			di := strings.TrimSpace(sets[i].ReleaseDate)
			dj := strings.TrimSpace(sets[j].ReleaseDate)
			if di != dj && (di == "" || dj == "") {
				return dj == ""
			}
			if asc {
				return di < dj
			}
			return di > dj
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
			// Compare normalised parameter counts so T/M-sized sets sort correctly.
			if asc {
				return sets[i].SizeInB() < sets[j].SizeInB()
			}
			return sets[i].SizeInB() > sets[j].SizeInB()
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

// defaultHuggingFaceCatalogURLs lists the built-in catalog sources whose
// contents are HuggingFace-sourced, i.e. the ones eligible for dual-catalog
// loading (HF + ModelScope mirrors). legacyModelCatalogURL is included so
// deployments still pointing at the retired default get both mirrors.
var defaultHuggingFaceCatalogURLs = map[string]struct{}{
	DefaultModelCatalogURL: {},
	legacyModelCatalogURL:  {},
}

func isBuiltinHuggingFaceCatalogURL(source string) bool {
	_, ok := defaultHuggingFaceCatalogURLs[strings.TrimSpace(source)]
	return ok
}

// resolveModelCatalogSource switches to the ModelScope catalog URL when the
// configured source is a built-in HuggingFace URL and HF is unreachable while
// ModelScope is reachable. Kept for callers/tests that still expect the
// historical single-URL resolution; dual-mode Start/Refresh no longer uses it.
func resolveModelCatalogSource(source string) string {
	source = strings.TrimSpace(source)
	if !isBuiltinHuggingFaceCatalogURL(source) {
		return source
	}
	if canAccessCatalogURL(catalogHubHuggingFace) || !canAccessCatalogURL(catalogHubModelScope) {
		return source
	}
	log.Infof("Cannot access %s, using ModelScope model catalog at %s", catalogHubHuggingFace, DefaultModelCatalogModelScopeURL)
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
