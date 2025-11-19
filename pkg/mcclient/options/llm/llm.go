package llm

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMBaseListOptions struct {
	options.BaseListOptions
	Host      string   `help:"filter by host"`
	LLMStatus []string `help:"filter by server status"`

	ListenPort int    `help:"filter by listen port"`
	PublicIp   string `help:"filter by public ip"`
	VolumeId   string `help:"filter by volume id"`
	Unused     *bool  `help:"filter by unused"`
	Used       *bool  `help:"filter by used"`
}

type LLMListOptions struct {
	LLMBaseListOptions

	LlmModel string `help:"filter by llm model"`
	LlmImage string `help:"filter by llm image"`
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

type LLMBaseCreateOptions struct {
	options.BaseCreateOptions

	AutoStart  bool
	ProjectId  string
	PreferHost string

	BandwidthMb int

	Count int `default:"1" help:"batch create count" json:"-"`
}

type LLMCreateOptions struct {
	LLMBaseCreateOptions

	LLM_MODEL_ID string `help:"llm model id or name" json:"llm_model_id"`
}

func (o *LLMCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *LLMCreateOptions) GetCountParam() int {
	return o.Count
}

type LLMDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMDeleteOptions) GetId() string {
	return o.ID
}

func (o *LLMDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMStartOptions struct {
	options.BaseIdsOptions
}

func (o *LLMStartOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMStopOptions struct {
	options.BaseIdsOptions
}

func (o *LLMStopOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMIdOptions struct {
	ID string `help:"llm id" json:"-"`
}

func (opts *LLMIdOptions) GetId() string {
	return opts.ID
}

func (opts *LLMIdOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type LLMSaveInstantAppOptions struct {
	LLMIdOptions

	PACKAGE string `help:"llm model id, e.g. 500a1f067a9f"`
	Name    string `help:"instant app name, e.g. qwen3:8b"`

	// AutoRestart bool
}

func (opts *LLMSaveInstantAppOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMSaveInstantAppInput{
		PackageName: opts.PACKAGE,
		ImageName:   opts.Name,
		// AutoRestart: opts.AutoRestart,
	}
	return jsonutils.Marshal(input), nil
}
