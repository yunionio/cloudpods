package llm

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"

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

	LlmSku   string `help:"filter by llm sku"`
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

	LLM_SKU_ID string `help:"llm sku id or name" json:"llm_sku_id"`
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

type LLMSaveInstantModelOptions struct {
	LLMIdOptions

	MODEL_ID string `help:"llm model id, e.g. 500a1f067a9f"`
	Name     string `help:"instant app name, e.g. qwen3:8b"`

	AutoRestart bool
}

func (opts *LLMSaveInstantModelOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMSaveInstantModelInput{
		ModelId:       opts.MODEL_ID,
		ModelFullName: opts.Name,
		// AutoRestart: opts.AutoRestart,
	}
	return jsonutils.Marshal(input), nil
}

type LLMQuickModelsOptions struct {
	LLMIdOptions

	MODEL []string `help:"model id and optional display name in the format of modelId[@modelName:modelTag], e.g. 6f48b936a09f or 6f48b936a09f@qwen2:0.5b"`

	Method string `help:"install or uninstall" choices:"install|uninstall"`
}

func (opts *LLMQuickModelsOptions) Params() (jsonutils.JSONObject, error) {
	params := api.LLMPerformQuickModelsInput{}
	for _, mdlFul := range opts.MODEL {
		var mdl api.ModelInfo

		var idPart string
		var nameAndTagPart string

		if idx := strings.Index(mdlFul, "@"); idx >= 0 {
			idPart = mdlFul[:idx]
			nameAndTagPart = mdlFul[idx+1:]

			if idxTag := strings.LastIndex(nameAndTagPart, ":"); idxTag >= 0 {
				mdl.DisplayName = nameAndTagPart[:idxTag]
				mdl.Tag = nameAndTagPart[idxTag+1:]
			} else {
				mdl.DisplayName = nameAndTagPart
			}
		} else {
			idPart = mdlFul

			if idxTag := strings.LastIndex(idPart, ":"); idxTag >= 0 {
				mdl.Tag = idPart[idxTag+1:]
				idPart = idPart[:idxTag]
			}
		}

		if regutils.MatchUUID(idPart) {
			mdl.Id = idPart
		} else {
			mdl.ModelId = idPart
		}

		params.Models = append(params.Models, mdl)
	}

	if len(opts.Method) > 0 {
		params.Method = api.TQuickModelMethod(opts.Method)
	}
	return jsonutils.Marshal(params), nil
}
