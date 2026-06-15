package models

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

const (
	DefaultLLMImagesCatalogURL = "https://www.cloudpods.org/llmimages.yaml"
	imagesCatalogFetchTimeout  = 30 * time.Second
	bundleImportKind           = "bundle"
)

// SLLMImagesCatalogManager caches community image catalog entries loaded from
// a configurable YAML source (http(s) URL or local file path).
type SLLMImagesCatalogManager struct {
	mu          sync.RWMutex
	items       []api.LLMImagesCatalogItem
	itemsById   map[string]*api.LLMImagesCatalogItem
	lastFetched time.Time
	lastErr     error
	source      string
}

var llmImagesCatalogManager *SLLMImagesCatalogManager
var llmImagesCatalogManagerOnce sync.Once

// GetLLMImagesCatalogManager returns the singleton catalog manager.
func GetLLMImagesCatalogManager() *SLLMImagesCatalogManager {
	llmImagesCatalogManagerOnce.Do(func() {
		llmImagesCatalogManager = &SLLMImagesCatalogManager{
			itemsById: map[string]*api.LLMImagesCatalogItem{},
		}
	})
	return llmImagesCatalogManager
}

// Start kicks off the initial load and optional periodic refresh.
func (m *SLLMImagesCatalogManager) Start(ctx context.Context, source string, interval time.Duration) {
	if source == "" {
		source = DefaultLLMImagesCatalogURL
	}
	m.mu.Lock()
	m.source = source
	m.mu.Unlock()

	go func() {
		if err := m.Refresh(ctx); err != nil {
			log.Warningf("LLMImagesCatalog: initial load from %s failed: %s", source, err)
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
					log.Warningf("LLMImagesCatalog: periodic refresh failed: %s", err)
				}
			}
		}
	}()
}

// Refresh re-loads the configured source and replaces the in-memory cache.
// On error the previous cache is preserved.
func (m *SLLMImagesCatalogManager) Refresh(ctx context.Context) error {
	m.mu.RLock()
	source := m.source
	m.mu.RUnlock()
	if source == "" {
		source = DefaultLLMImagesCatalogURL
	}

	body, err := m.loadSource(ctx, source)
	if err != nil {
		m.recordErr(err)
		return err
	}

	var rawItems []api.LLMImagesCatalogItem
	if err := yaml.Unmarshal(body, &rawItems); err != nil {
		m.recordErr(err)
		return errors.Wrap(err, "parse llmimages yaml")
	}

	items := make([]api.LLMImagesCatalogItem, 0, len(rawItems))
	itemsById := make(map[string]*api.LLMImagesCatalogItem, len(rawItems))
	for i := range rawItems {
		item := rawItems[i]
		if item.LLMType == "" {
			continue
		}
		item.Id = catalogItemId(&item, i)
		if item.Id == "" {
			continue
		}
		items = append(items, item)
		itemsById[item.Id] = &items[len(items)-1]
	}

	m.mu.Lock()
	m.items = items
	m.itemsById = itemsById
	m.lastFetched = time.Now()
	m.lastErr = nil
	m.mu.Unlock()

	log.Infof("LLMImagesCatalog: refreshed %d items from %s", len(items), source)
	return nil
}

func catalogItemId(item *api.LLMImagesCatalogItem, index int) string {
	if item.ImportKind == bundleImportKind && item.Name != "" {
		return item.Name
	}
	if item.Image != "" {
		imageName, imageLabel := parseCatalogImageRef(item.Image)
		return item.LLMType + ":" + imageName + ":" + imageLabel
	}
	return fmt.Sprintf("%s-%d", item.LLMType, index)
}

func parseCatalogImageRef(imageStr string) (string, string) {
	idx := strings.LastIndex(imageStr, ":")
	if idx > 0 {
		return imageStr[:idx], imageStr[idx+1:]
	}
	return imageStr, "latest"
}

func (m *SLLMImagesCatalogManager) loadSource(ctx context.Context, source string) ([]byte, error) {
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

func (m *SLLMImagesCatalogManager) fetchHTTP(ctx context.Context, url string) ([]byte, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, imagesCatalogFetchTimeout)
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

func (m *SLLMImagesCatalogManager) recordErr(err error) {
	m.mu.Lock()
	m.lastErr = err
	m.mu.Unlock()
}

// ListItems returns items matching the filter. limit <= 0 returns all items
// from offset (community catalog is small).
func (m *SLLMImagesCatalogManager) ListItems(input api.LLMImagesCatalogListInput) ([]api.LLMImagesCatalogItem, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filtered := make([]api.LLMImagesCatalogItem, 0, len(m.items))
	for _, item := range m.items {
		if !catalogItemMatches(&item, input) {
			continue
		}
		filtered = append(filtered, item)
	}
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
		return filtered[offset:], total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

// GetItem returns one catalog entry by id.
func (m *SLLMImagesCatalogManager) GetItem(id string) (*api.LLMImagesCatalogItem, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.itemsById[id]
	if !ok {
		return nil, false
	}
	cp := *item
	return &cp, true
}

func catalogItemMatches(item *api.LLMImagesCatalogItem, input api.LLMImagesCatalogListInput) bool {
	if input.LLMType != "" && !strings.EqualFold(item.LLMType, input.LLMType) {
		return false
	}
	if input.Search == "" {
		return true
	}
	needle := strings.ToLower(input.Search)
	haystack := strings.ToLower(strings.Join([]string{
		item.Image,
		item.LLMType,
		item.Description,
		item.AppName,
		item.Name,
	}, " "))
	return strings.Contains(haystack, needle)
}
