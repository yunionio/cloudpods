package llm

import "yunion.io/x/onecloud/pkg/apis"

type LLMInternalPkgInfo struct {
	ModelId string `json:"model_id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Size    int64  `json:"size"`
	// Modified string   `json:"modified"`
	Blobs []string `json:"blobs"`
}

type LLMSaveInstantAppInput struct {
	apis.ProjectizedResourceCreateInput

	PackageName string `json:"package_name"`
	ImageName   string `json:"image_name"`

	InstantAppId string `json:"instant_app_id"`

	// AutoRestart bool `json:"auto_restart"`
}
