package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifySkuListOptions struct {
	options.BaseListOptions
}

func (o *DifySkuListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type DifySkuShowOptions struct {
	options.BaseShowOptions
}

func (o *DifySkuShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifySkuCreateOptions struct {
	LLMSkuBaseCreateOptions

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

func (o *DifySkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMSkuBaseCreateOptions.Params(dict)
	return dict, nil
}

type DifySkuDeleteOptions struct {
	options.BaseIdOptions
}

func (o *DifySkuDeleteOptions) GetId() string {
	return o.ID
}

func (o *DifySkuDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifySkuUpdateOptions struct {
	LLMSkuBaseUpdateOptions

	PostgresImageID     string `json:"postgres_image_id"`
	RedisImageID        string `json:"redis_image_id"`
	NginxImageID        string `json:"nginx_image_id"`
	DifyApiImageID      string `json:"dify_api_image_id"`
	DifyPluginImageID   string `json:"dify_plugin_image_id"`
	DifyWebImageID      string `json:"dify_web_image_id"`
	DifySandboxImageID  string `json:"dify_sandbox_image_id"`
	DifySsrfImageID     string `json:"dify_ssrf_image_id"`
	DifyWeaviateImageID string `json:"dify_weaviate_image_id"`
}

func (o *DifySkuUpdateOptions) GetId() string {
	return o.ID
}

func (o *DifySkuUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	o.LLMSkuBaseUpdateOptions.Params(dict)
	return dict, nil
}
