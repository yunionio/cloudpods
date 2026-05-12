package llm

import "yunion.io/x/onecloud/pkg/apis"

type LLMInternalInstantMdlInfo struct {
	ModelId string `json:"model_id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Size    int64  `json:"size"`
	// Modified string   `json:"modified"`
	Blobs     []string `json:"blobs"`
	Mounts    []string `json:"mounts,omitempty"`
	ModelType string   `json:"model_type,omitempty"`
}

type LLMSaveInstantModelInput struct {
	apis.ProjectizedResourceCreateInput

	ModelId       string   `json:"model_id"`
	Name          string   `json:"name"`
	ModelFullName string   `json:"model_full_name"`
	Mounts        []string `json:"mounts"`

	InstantModelId string `json:"instant_model_id"`

	AutoRestart bool `json:"auto_restart"`
}
