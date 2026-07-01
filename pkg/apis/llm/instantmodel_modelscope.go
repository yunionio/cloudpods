package llm

type InstantModelModelScopeSearchInput struct {
	Q        string `json:"q"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

type InstantModelModelScopeSearchOutput struct {
	Data    []InstantModelModelScopeSearchResult `json:"data"`
	Page    int                                  `json:"page"`
	HasMore bool                                 `json:"has_more"`
	Total   int                                  `json:"total,omitempty"`
}

type InstantModelModelScopeSearchResult struct {
	ModelId           string   `json:"model_id"`
	Name              string   `json:"name,omitempty"`
	Author            string   `json:"author,omitempty"`
	PipelineTag       string   `json:"pipeline_tag,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Downloads         int64    `json:"downloads,omitempty"`
	Likes             int64    `json:"likes,omitempty"`
	LastModified      string   `json:"last_modified,omitempty"`
	Supported         bool     `json:"supported"`
	UnsupportedReason string   `json:"unsupported_reason,omitempty"`
}

type InstantModelModelScopeRepoInfoInput struct {
	ModelId  string `json:"model_id"`
	Revision string `json:"revision,omitempty"`
}

type InstantModelModelScopeRepoInfo struct {
	ModelId            string   `json:"model_id"`
	RequestedRevision  string   `json:"requested_revision,omitempty"`
	ResolvedRevision   string   `json:"resolved_revision,omitempty"`
	Siblings           []string `json:"siblings,omitempty"`
	ConfigPresent      bool     `json:"config_present"`
	SafetensorsPresent bool     `json:"safetensors_present"`
	GgufPresent        bool     `json:"gguf_present"`
	ReadmePresent      bool     `json:"readme_present"`
	SizeBytes          int64    `json:"size_bytes,omitempty"`
	Supported          bool     `json:"supported"`
	UnsupportedReason  string   `json:"unsupported_reason,omitempty"`
	ImportMode         string   `json:"import_mode,omitempty"`
}
