package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMImageShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMImageShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMImageListOptions struct {
	options.BaseListOptions
}

func (o *LLMImageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMImageCreateOptions struct {
	apis.SharableVirtualResourceCreateInput
	IMAGE_NAME   string
	IMAGE_LABEL  string
	CredentialId string
}

func (o *LLMImageCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMImageUpdateOptions struct {
	apis.SharableVirtualResourceCreateInput

	ID           string
	IMAGE_NAME   string
	IMAGE_LABEL  string
	CredentialId string
}

func (o *LLMImageUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMImageUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMImageDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMImageDeleteOptions) GetId() string {
	return o.ID
}
