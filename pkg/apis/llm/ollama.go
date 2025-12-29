package llm

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
)

type OllamaModelFileSpec struct {
	Parameter *OllamaModelFileParameter `json:"parameter"`
	Template  *string                   `json:"template,omitempty"`
	System    *string                   `json:"system,omitempty"`
	// Adapter   *string                `json:"adapter,omitempty"`
	License *string                   `json:"license,omitempty"`
	Message []*OllamaModelFileMessage `json:"message,omitempty"`
}

type OllamaGgufSpec struct {
	GgufFile  string               `json:"gguf_file"`
	Source    string               `json:"source"`
	ModelFile *OllamaModelFileSpec `json:"modelfile,omitempty"`
}

// type OllamaPullModelInput struct {
// 	Model string          `json:"model"`
// 	Gguf  *OllamaGgufSpec `json:"gguf,omitempty"`
// }

// type OllamaCreateInput struct {
// 	compute.ServerCreateInput
// 	OllamaPullModelInput
// }

type OllamaListInput struct {
	apis.VirtualResourceListInput
	GuestId string `json:"guest_id"`
}

type OllamaAccessCacheInput struct {
	ModelName string   `json:"model_name"`
	Blobs     []string `json:"blobs"`
}

type OllamaAccessGgufFileInput struct {
	HostPath  string `json:"host_path"`
	TargetDir string `json:"target_dir"`
}

type OllamaModelFileMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (m *OllamaModelFileMessage) ValidateRole() error {
	switch m.Role {
	case LLM_OLLAMA_GGUF_MESSAGE_ROLE_SYSTEM,
		LLM_OLLAMA_GGUF_MESSAGE_ROLE_USER,
		LLM_OLLAMA_GGUF_MESSAGE_ROLE_ASSISTANT:
		return nil
	default:
		return errors.Errorf("invalid role: %s, must be one of: system, user, assistant", m.Role)
	}
}

type OllamaModelFileParameter struct {
	NumCtx        *int     `json:"num_ctx,omitempty"`
	RepeatLastN   *int     `json:"repeat_last_n,omitempty"`
	RepeatPenalty *float64 `json:"repeat_penalty,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	Seed          *int     `json:"seed,omitempty"`
	Stop          *string  `json:"stop,omitempty"`
	NumPredict    *int     `json:"num_predict,omitempty"`
	TopK          *int     `json:"top_k,omitempty"`
	TopP          *float64 `json:"top_p,omitempty"`
	MinP          *float64 `json:"min_p,omitempty"`
}

func (p *OllamaModelFileParameter) GetParameters() map[string]string {
	pairs := make(map[string]string)

	addInt := func(key string, val *int) {
		if val != nil {
			pairs[key] = fmt.Sprintf("%d", *val)
		}
	}
	addFloat := func(key string, val *float64) {
		if val != nil {
			pairs[key] = fmt.Sprintf("%f", *val)
		}
	}
	addString := func(key string, val *string) {
		if val != nil {
			pairs[key] = fmt.Sprintf("\"%s\"", *val)
		}
	}

	addInt(LLM_OLLAMA_MODELFILE_PARAMETER_NUM_CTX, p.NumCtx)
	addInt(LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_LAST_N, p.RepeatLastN)
	addFloat(LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_PENALTY, p.RepeatPenalty)
	addFloat(LLM_OLLAMA_MODELFILE_PARAMETER_TEMPERATURE, p.Temperature)
	addInt(LLM_OLLAMA_MODELFILE_PARAMETER_SEED, p.Seed)
	addString(LLM_OLLAMA_MODELFILE_PARAMETER_STOP, p.Stop)
	addInt(LLM_OLLAMA_MODELFILE_PARAMETER_NUM_PREDICT, p.NumPredict)
	addInt(LLM_OLLAMA_MODELFILE_PARAMETER_TOP_K, p.TopK)
	addFloat(LLM_OLLAMA_MODELFILE_PARAMETER_TOP_P, p.TopP)
	addFloat(LLM_OLLAMA_MODELFILE_PARAMETER_MIN_P, p.MinP)

	return pairs
}
