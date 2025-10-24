package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifyModelListOptions struct {
	options.BaseListOptions
}

func (o *DifyModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type DifyModelShowOptions struct {
	options.BaseShowOptions
}

func (o *DifyModelShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifyModelCreateOptions struct {
	LLMModelBaseCreateOptions

	POSTGRES_IMAGE_ID      string `json:"postgres_image_id"`
	REDIS_IMAGE_ID         string `json:"redis_image_id"`
	NGINX_IMAGE_ID         string `json:"nginx_image_id"`
	DIFY_API_IMAGE_ID      string `json:"dify_api_image_id"`
	DIFY_PLUGIN_IMAGE_ID   string `json:"dify_plugin_image_id"`
	DIFY_WEB_IMAGE_ID      string `json:"dify_web_image_id"`
	DIFY_SANDBOX_IMAGE_ID  string `json:"dify_sandbox_image_id"`
	DIFY_SSRF_IMAGE_ID     string `json:"dify_ssrf_image_id"`
	DIFY_WEAVIATE_IMAGE_ID string `json:"dify_weaviate_image_id"`
}

func (o *DifyModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMModelBaseCreateOptions.Params(dict)
	return dict, nil
}

type DifyModelDeleteOptions struct {
	options.BaseIdOptions
}

func (o *DifyModelDeleteOptions) GetId() string {
	return o.ID
}

func (o *DifyModelDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifyModelUpdateOptions struct {
	LLMModelBaseUpdateOptions

	LlmImageId   string
	LlmModelName string
}

func (o *DifyModelUpdateOptions) GetId() string {
	return o.ID
}

func (o *DifyModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMModelBaseUpdateOptions.Params(dict)
	return dict, nil
}
