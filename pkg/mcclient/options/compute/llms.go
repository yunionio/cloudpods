package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMModelOptions struct {
	MODEL string `json:"model" help:"Chosen large language model"`
}

type LLMCreateOptions struct {
	PodCreateOptions
	LLMModelOptions
}

func (o *LLMCreateOptions) Params() (jsonutils.JSONObject, error) {
	input, err := o.PodCreateOptions.Params()
	if err != nil {
		return nil, err
	}

	params := input.JSON(input)
	params.Set("model", jsonutils.NewString(o.MODEL))

	return params, nil
}

type LLMListOptions struct {
	options.BaseListOptions
	GuestId     string `json:"guest_id" help:"guest(pod) id or name"`
	ContainerId string `json:"container_id" help:"container id or name"`
}

func (o *LLMListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMIdsOptions struct {
	ID []string `help:"ID of llms to operate" metavar:"LLM" json:"-"`
}

func (o *LLMIdsOptions) GetIds() []string {
	return o.ID
}

func (o *LLMIdsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type LLMIdOptions struct {
	ID string `help:"ID or name of the llm" json:"-"`
}

func (o *LLMIdOptions) GetId() string {
	return o.ID
}

func (o *LLMIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type LLMShowOptions struct {
	LLMIdOptions
}

type LLMStopOptions struct {
	LLMIdsOptions
	Timeout int  `help:"Stopping timeout" json:"timeout"`
	Force   bool `help:"Force stop llm" json:"force"`
}

func (o *LLMStopOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMStartOptions struct {
	LLMIdsOptions
}

type LLMChangeModelOptions struct {
	LLMIdOptions
	LLMModelOptions
}

func (o *LLMChangeModelOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
