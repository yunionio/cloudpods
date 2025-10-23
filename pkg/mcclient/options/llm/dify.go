package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifyListOptions struct {
	LLMBaseListOptions

	DifyModel string `help:"filter by dify model"`
}

func (o *DifyListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(o)
	if err != nil {
		return nil, err
	}
	if o.Used != nil {
		params.Set("unused", jsonutils.JSONFalse)
	}
	return params, nil
}

type DifyShowOptions struct {
	options.BaseShowOptions
}

func (o *DifyShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifyCreateOptions struct {
	LLMBaseCreateOptions

	DIFY_MODEL_ID string `help:"dify model id or name" json:"dify_model_id"`
}

func (o *DifyCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *DifyCreateOptions) GetCountParam() int {
	return o.Count
}

type DifyDeleteOptions struct {
	options.BaseIdOptions
}

func (o *DifyDeleteOptions) GetId() string {
	return o.ID
}

func (o *DifyDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
