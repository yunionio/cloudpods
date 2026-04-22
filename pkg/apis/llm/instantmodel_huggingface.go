package llm

type InstantModelHuggingFaceSearchInput struct {
	Q     string `json:"q"`
	Limit int    `json:"limit,omitempty"`
	Sort  string `json:"sort,omitempty"`
}

type InstantModelHuggingFaceRepoInfoInput struct {
	RepoId   string `json:"repo_id"`
	Revision string `json:"revision,omitempty"`
}

type InstantModelHuggingFaceSearchResult struct {
	RepoId            string   `json:"repo_id"`
	Author            string   `json:"author,omitempty"`
	Sha               string   `json:"sha,omitempty"`
	LastModified      string   `json:"last_modified,omitempty"`
	Downloads         int64    `json:"downloads,omitempty"`
	Likes             int64    `json:"likes,omitempty"`
	PipelineTag       string   `json:"pipeline_tag,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Private           bool     `json:"private,omitempty"`
	Gated             bool     `json:"gated,omitempty"`
	Disabled          bool     `json:"disabled,omitempty"`
	Supported         bool     `json:"supported"`
	UnsupportedReason string   `json:"unsupported_reason,omitempty"`
}

type InstantModelHuggingFaceRepoInfo struct {
	RepoId             string   `json:"repo_id"`
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
