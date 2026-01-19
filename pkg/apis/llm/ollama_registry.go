package llm

import (
	"yunion.io/x/jsonutils"
)

type SOllamaTag struct {
	Name          string   `json:"name" yaml:"name"`
	ModelSize     string   `json:"model_size" yaml:"model_size"`
	ContextLength string   `json:"context_length" yaml:"context_length"`
	Capabilities  []string `json:"capabilities" yaml:"capabilities"`
	IsLatest      bool     `json:"is_latest,omitempty" yaml:"is_latest,omitempty"`
}

func (t SOllamaTag) Latest() SOllamaTag {
	t.IsLatest = true
	return t
}

type SOllamaModel struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description" yaml:"description"`
	Tags        []SOllamaTag `json:"tags" yaml:"tags"`
}

// SOllamaRegistry 顶层结构，用于生成
// ollama:
//   - name: xxx
//     ...
type SOllamaRegistry struct {
	Ollama []SOllamaModel `json:"ollama" yaml:"ollama"`
}

func NewOllamaTag(name, size, contextLen string, caps []string) SOllamaTag {
	return SOllamaTag{
		Name:          name,
		ModelSize:     size,
		ContextLength: contextLen,
		Capabilities:  caps,
	}
}

func NewOllamaModel(name, desc string, tags ...SOllamaTag) SOllamaModel {
	return SOllamaModel{
		Name:        name,
		Description: desc,
		Tags:        tags,
	}
}

func NewOllamaRegistry(models ...SOllamaModel) SOllamaRegistry {
	return SOllamaRegistry{
		Ollama: models,
	}
}

var (
	CapText   = []string{"Text"}
	CapVision = []string{"Text", "Image"}
)

var OllamaRegistry = NewOllamaRegistry(
	NewOllamaModel(
		"qwen3-vl",
		"Qwen3-vl is the most powerful vision-language model in the Qwen model family to date.",
		NewOllamaTag("2b", "1.9GB", "256K", CapVision),
		NewOllamaTag("4b", "3.3GB", "256K", CapVision),
		NewOllamaTag("8b", "6.1GB", "256K", CapVision).Latest(),
		NewOllamaTag("30b", "20GB", "256K", CapVision),
		NewOllamaTag("32b", "21GB", "256K", CapVision),
	),
	NewOllamaModel(
		"qwen3",
		"Qwen3 is the latest generation of large language models in Qwen series, offering a comprehensive suite of dense and mixture-of-experts (MoE) models.",
		NewOllamaTag("0.6b", "523MB", "40K", CapText),
		NewOllamaTag("1.7b", "1.4GB", "40K", CapText),
		NewOllamaTag("4b", "2.5GB", "256K", CapText),
		NewOllamaTag("8b", "5.2GB", "40K", CapText).Latest(),
		NewOllamaTag("14b", "9.3GB", "40K", CapText),
		NewOllamaTag("30b", "19GB", "256K", CapText),
		NewOllamaTag("32b", "20GB", "40K", CapText),
	),
	NewOllamaModel(
		"qwen2.5-coder",
		"The latest series of Code-Specific Qwen models, with significant improvements in code generation, code reasoning, and code fixing.",
		NewOllamaTag("latest", "4.7GB", "32K", CapText),
		NewOllamaTag("0.5b", "398MB", "32K", CapText),
		NewOllamaTag("1.5b", "986MB", "32K", CapText),
		NewOllamaTag("3b", "1.9GB", "32K", CapText),
		NewOllamaTag("7b", "4.7GB", "32K", CapText).Latest(),
		NewOllamaTag("14b", "9.0GB", "32K", CapText),
		NewOllamaTag("32b", "20GB", "32K", CapText),
	),
)

var OLLAMA_REGISTRY_YAML = jsonutils.Marshal(OllamaRegistry).YAMLString()
