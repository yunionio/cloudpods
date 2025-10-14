package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMListOptions struct {
	options.BaseListOptions

	Host     string `help:"filter by host"`
	LlmModel string `help:"filter by llm model"`
	LlmImage string `help:"filter by llm image"`

	LLMStatus []string `help:"filter by llm status"`

	ListenPort int    `help:"filter by listen port"`
	PublicIp   string `help:"filter by public ip"`
	VolumeId   string `help:"filter by volume id"`
	Unused     *bool  `help:"filter by unused"`
	Used       *bool  `help:"filter by used"`
}

func (o *LLMListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(o)
	if err != nil {
		return nil, err
	}
	if o.Used != nil {
		params.Set("unused", jsonutils.JSONFalse)
	}
	return params, nil
}

type LLMShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMCreateOptions struct {
	options.BaseCreateOptions

	LLM_MODEL_ID string `help:"llm model id or name" json:"llm_model_id"`
	AutoStart    bool
	ProjectId    string
	PreferHost   string

	BandwidthMb int

	Count int `default:"1" help:"batch create count" json:"-"`
}

func (o *LLMCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *LLMCreateOptions) GetCountParam() int {
	return o.Count
}
