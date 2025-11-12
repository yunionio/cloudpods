package llm

type LLMInternalPkgInfo struct {
	ModelId string `json:"model_id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Size    int64  `json:"size"`
	// Modified string   `json:"modified"`
	Blobs []string `json:"blobs"`
}
