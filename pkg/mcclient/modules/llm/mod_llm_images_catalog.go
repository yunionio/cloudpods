package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMImagesCatalogs LLMImagesCatalogsManager
)

func init() {
	LLMImagesCatalogs = LLMImagesCatalogsManager{
		modules.NewLLMManager("llm_images_catalog", "llm_images_catalogs",
			[]string{"id", "llm_type", "image", "name", "description"},
			[]string{}),
	}
	modules.Register(&LLMImagesCatalogs)
}

type LLMImagesCatalogsManager struct {
	modulebase.ResourceManager
}
