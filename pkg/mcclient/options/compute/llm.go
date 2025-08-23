package compute

import (
	"yunion.io/x/jsonutils"
)

type LLMCreateOptions struct {
	PodCreateOptions
	MODEL string `json:"model" help:"Chosen large language model"`
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
