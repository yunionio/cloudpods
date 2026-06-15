package llm

// LLMImagesCatalogItem is one entry in llmimages.yaml — either a single
// community image or a bundle (e.g. dify with import_kind: bundle).
type LLMImagesCatalogItem struct {
	// Resource id for mcclient list/show. Populated at load time.
	Id string `json:"id" yaml:"-"`

	// Single-image fields
	Image       string                 `json:"image,omitempty" yaml:"image"`
	LLMType     string                 `json:"llm_type" yaml:"llm_type"`
	Description string                 `json:"description,omitempty" yaml:"description"`
	AppName     string                 `json:"app_name,omitempty" yaml:"app_name"`
	Desktop     *LLMImageDesktopConfig `json:"desktop,omitempty" yaml:"desktop"`

	// Bundle fields
	Name       string                        `json:"name,omitempty" yaml:"name"`
	ImportKind string                        `json:"import_kind,omitempty" yaml:"import_kind"`
	Images     []LLMImagesCatalogBundleImage `json:"images,omitempty" yaml:"images"`
	Sku        *LLMImagesCatalogSku          `json:"sku,omitempty" yaml:"sku"`
}

// LLMImagesCatalogBundleImage is one component image inside a bundle entry.
type LLMImagesCatalogBundleImage struct {
	Role         string `json:"role" yaml:"role"`
	GenerateName string `json:"generate_name" yaml:"generate_name"`
	Image        string `json:"image" yaml:"image"`
}

// LLMImagesCatalogSku holds optional default SKU spec embedded in yaml.
type LLMImagesCatalogSku struct {
	GenerateName string       `json:"generate_name,omitempty" yaml:"generate_name"`
	CPU          int          `json:"cpu,omitempty" yaml:"cpu"`
	Memory       int          `json:"memory,omitempty" yaml:"memory"`
	VolumeSizeMb int          `json:"volume_size_mb,omitempty" yaml:"volume_size_mb"`
	Bandwidth    int          `json:"bandwidth,omitempty" yaml:"bandwidth"`
	PortMappings PortMappings `json:"port_mappings,omitempty" yaml:"port_mappings"`
}

// LLMImagesCatalogListInput is the query filter for GET /llm_images_catalogs.
type LLMImagesCatalogListInput struct {
	LLMType string `json:"llm_type"`
	Search  string `json:"search"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

// LLMImagesCatalogListOutput mirrors the standard list-response envelope.
type LLMImagesCatalogListOutput struct {
	LLMImagesCatalogs []LLMImagesCatalogItem `json:"llm_images_catalogs"`
	Total             int                    `json:"total"`
	Limit             int                    `json:"limit"`
	Offset            int                    `json:"offset"`
}

// LLMImagesCatalogShowOutput is the single-item response wrapper.
type LLMImagesCatalogShowOutput struct {
	LLMImagesCatalog LLMImagesCatalogItem `json:"llm_images_catalog"`
}
