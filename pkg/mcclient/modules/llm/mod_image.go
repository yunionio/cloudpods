package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMImageManager struct {
	modulebase.ResourceManager
}

var (
	LLMImage LLMImageManager
)

func init() {
	LLMImage = LLMImageManager{
		ResourceManager: modules.NewLLMManager("llm_image", "llm_images",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&LLMImage)
}
