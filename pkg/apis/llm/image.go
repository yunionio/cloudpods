package llm

import (
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
)

// LLM image capability markers (orthogonal to llm_type).
const (
	LLM_IMAGE_CAPABILITY_OMNI = "omni"
)

var LLM_IMAGE_CAPABILITIES = sets.NewString(
	LLM_IMAGE_CAPABILITY_OMNI,
)

// NormalizeLLMImageCapabilities lowercases, trims and deduplicates capabilities.
func NormalizeLLMImageCapabilities(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		c := strings.ToLower(strings.TrimSpace(raw))
		if c == "" {
			continue
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

// ValidateLLMImageCapabilities rejects unknown capability values.
func ValidateLLMImageCapabilities(in []string) error {
	for _, raw := range in {
		c := strings.ToLower(strings.TrimSpace(raw))
		if c == "" {
			continue
		}
		if !LLM_IMAGE_CAPABILITIES.Has(c) {
			return errors.Errorf("invalid capability %q, must be one of %s", raw, strings.Join(LLM_IMAGE_CAPABILITIES.List(), ","))
		}
	}
	return nil
}

// HasLLMImageCapability reports whether caps contains want (case-insensitive).
func HasLLMImageCapability(caps []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, c := range caps {
		if strings.ToLower(strings.TrimSpace(c)) == want {
			return true
		}
	}
	return false
}

type LLMImageType string

const (
	LLM_IMAGE_TYPE_OLLAMA       LLMImageType = "ollama"
	LLM_IMAGE_TYPE_VLLM         LLMImageType = "vllm"
	LLM_IMAGE_TYPE_SGLANG       LLMImageType = "sglang"
	LLM_IMAGE_TYPE_DIFY         LLMImageType = "dify"
	LLM_IMAGE_TYPE_COMFYUI      LLMImageType = "comfyui"
	LLM_IMAGE_TYPE_OPENCLAW     LLMImageType = "openclaw"
	LLM_IMAGE_TYPE_HERMES_AGENT LLMImageType = "hermes-agent"
	LLM_IMAGE_TYPE_DESKTOP      LLMImageType = "desktop"
	LLM_IMAGE_TYPE_BENCHMARK    LLMImageType = "benchmark"
)

var (
	LLM_IMAGE_TYPES = sets.NewString(
		string(LLM_IMAGE_TYPE_OLLAMA),
		string(LLM_IMAGE_TYPE_VLLM),
		string(LLM_IMAGE_TYPE_SGLANG),
		string(LLM_IMAGE_TYPE_DIFY),
		string(LLM_IMAGE_TYPE_COMFYUI),
		string(LLM_IMAGE_TYPE_OPENCLAW),
		string(LLM_IMAGE_TYPE_HERMES_AGENT),
		string(LLM_IMAGE_TYPE_DESKTOP),
		string(LLM_IMAGE_TYPE_BENCHMARK),
	)
)

func IsLLMImageType(t string) bool {
	return LLM_IMAGE_TYPES.Has(t)
}

// LLM image source: how image_name/image_label are provided.
const (
	LLM_IMAGE_SOURCE_MANUAL          = "manual"
	LLM_IMAGE_SOURCE_CONTAINER_IMAGE = "container_image"
)

type LLMImageListInput struct {
	apis.SharableVirtualResourceListInput

	ImageLabel       string `json:"image_label"`
	ImageName        string `json:"image_name"`
	LLMType          string `json:"llm_type"`
	AppName          string `json:"app_name"`
	ImageSource      string `json:"image_source"`
	ContainerImageId string `json:"container_image_id"`
	// Capability filters images whose capabilities JSON contains this value
	// (e.g. "omni").
	Capability string `json:"capability"`
}

type LLMImageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName        string                 `json:"image_name"`
	ImageLabel       string                 `json:"image_label"`
	CredentialId     string                 `json:"credential_id"`
	LLMType          string                 `json:"llm_type"`
	AppName          string                 `json:"app_name"`
	DesktopConfig    *LLMImageDesktopConfig `json:"desktop_config,omitempty"`
	ImageSource      string                 `json:"image_source"`
	ContainerImageId string                 `json:"container_image_id"`
	Capabilities     []string               `json:"capabilities"`
}

type LLMImageUpdateInput struct {
	apis.SharableVirtualResourceCreateInput

	ImageName        *string                `json:"image_name,omitempty"`
	ImageLabel       *string                `json:"image_label,omitempty"`
	CredentialId     *string                `json:"credential_id,omitempty"`
	LLMType          *string                `json:"llm_type,omitempty"`
	AppName          *string                `json:"app_name,omitempty"`
	DesktopConfig    *LLMImageDesktopConfig `json:"desktop_config,omitempty"`
	ImageSource      *string                `json:"image_source,omitempty"`
	ContainerImageId *string                `json:"container_image_id,omitempty"`
	Capabilities     *[]string              `json:"capabilities,omitempty"`
}

type LLMImageDetails struct {
	apis.SharableVirtualResourceDetails

	// ContainerImage is the linked container_image resource name (for UI link text).
	ContainerImage string `json:"container_image"`
}
