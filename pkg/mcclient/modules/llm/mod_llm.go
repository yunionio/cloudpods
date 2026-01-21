package llm

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMs LLMManager
)

func init() {
	LLMs = LLMManager{
		modules.NewLLMManager("llm", "llms",
			[]string{},
			[]string{}),
	}
	modules.Register(&LLMs)
}

type LLMManager struct {
	modulebase.ResourceManager
}

func (this *LLMManager) GetAvailableNetwork(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/available-network", this.KeywordPlural)
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return modulebase.Get(this.ResourceManager, session, path, this.Keyword)
}
