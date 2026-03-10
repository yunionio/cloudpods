package llm

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifySkuListOptions struct {
	options.BaseListOptions
}

func (o *DifySkuListOptions) Params() (jsonutils.JSONObject, error) {
	dict, err := options.ListStructToParams(o)
	if err != nil {
		return nil, err
	}
	dict.Set("llm_type", jsonutils.NewString(string(api.LLM_CONTAINER_DIFY)))
	return dict, nil
}

type DifySkuShowOptions struct {
	options.BaseShowOptions
}

func (o *DifySkuShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifySkuCreateOptions struct {
	LLMSkuBaseCreateOptions

	POSTGRES_IMAGE_ID      string                   `json:"postgres_image_id"`
	REDIS_IMAGE_ID         string                   `json:"redis_image_id"`
	NGINX_IMAGE_ID         string                   `json:"nginx_image_id"`
	DIFY_API_IMAGE_ID      string                   `json:"dify_api_image_id"`
	DIFY_PLUGIN_IMAGE_ID   string                   `json:"dify_plugin_image_id"`
	DIFY_WEB_IMAGE_ID      string                   `json:"dify_web_image_id"`
	DIFY_SANDBOX_IMAGE_ID  string                   `json:"dify_sandbox_image_id"`
	DIFY_SSRF_IMAGE_ID     string                   `json:"dify_ssrf_image_id"`
	DIFY_WEAVIATE_IMAGE_ID string                   `json:"dify_weaviate_image_id"`
	CustomizedEnvs         []*api.DifyCustomizedEnv `json:"customized_envs,omitempty"`
}

func (o *DifySkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	// Remove image id keys from top level; we put them in llm_spec
	for _, k := range []string{"postgres_image_id", "redis_image_id", "nginx_image_id", "dify_api_image_id", "dify_plugin_image_id", "dify_web_image_id", "dify_sandbox_image_id", "dify_ssrf_image_id", "dify_weaviate_image_id"} {
		dict.Remove(k)
	}
	if err := o.LLMSkuBaseCreateOptions.Params(dict); err != nil {
		return nil, err
	}
	dict.Set("llm_type", jsonutils.NewString(string(api.LLM_CONTAINER_DIFY)))
	spec := &api.LLMSpec{
		Ollama: nil,
		Vllm:   nil,
		Dify: &api.LLMSpecDify{
			PostgresImageId:     o.POSTGRES_IMAGE_ID,
			RedisImageId:        o.REDIS_IMAGE_ID,
			NginxImageId:        o.NGINX_IMAGE_ID,
			DifyApiImageId:      o.DIFY_API_IMAGE_ID,
			DifyPluginImageId:   o.DIFY_PLUGIN_IMAGE_ID,
			DifyWebImageId:      o.DIFY_WEB_IMAGE_ID,
			DifySandboxImageId:  o.DIFY_SANDBOX_IMAGE_ID,
			DifySSRFImageId:     o.DIFY_SSRF_IMAGE_ID,
			DifyWeaviateImageId: o.DIFY_WEAVIATE_IMAGE_ID,
			CustomizedEnvs:      o.CustomizedEnvs,
		},
	}
	dict.Set("llm_spec", jsonutils.Marshal(spec))
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

	PostgresImageId     string                   `json:"postgres_image_id"`
	RedisImageId        string                   `json:"redis_image_id"`
	NginxImageId        string                   `json:"nginx_image_id"`
	DifyApiImageId      string                   `json:"dify_api_image_id"`
	DifyPluginImageId   string                   `json:"dify_plugin_image_id"`
	DifyWebImageId      string                   `json:"dify_web_image_id"`
	DifySandboxImageId  string                   `json:"dify_sandbox_image_id"`
	DifySSRFImageId     string                   `json:"dify_ssrf_image_id"`
	DifyWeaviateImageId string                   `json:"dify_weaviate_image_id"`
	CustomizedEnvs      []*api.DifyCustomizedEnv `json:"customized_envs,omitempty"`
}

func (o *DifySkuUpdateOptions) GetId() string {
	return o.ID
}

func (o *DifySkuUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)

	// Remove image id keys from top level; put them in llm_spec when any is set
	for _, k := range []string{"postgres_image_id", "redis_image_id", "nginx_image_id", "dify_api_image_id", "dify_plugin_image_id", "dify_web_image_id", "dify_sandbox_image_id", "dify_ssrf_image_id", "dify_weaviate_image_id"} {
		dict.Remove(k)
	}
	if err := o.LLMSkuBaseUpdateOptions.Params(dict); err != nil {
		return nil, err
	}
	hasImageId := o.PostgresImageId != "" || o.RedisImageId != "" || o.NginxImageId != "" ||
		o.DifyApiImageId != "" || o.DifyPluginImageId != "" || o.DifyWebImageId != "" ||
		o.DifySandboxImageId != "" || o.DifySSRFImageId != "" || o.DifyWeaviateImageId != ""
	if hasImageId || len(o.CustomizedEnvs) > 0 {
		spec := &api.LLMSpec{
			Ollama: nil,
			Vllm:   nil,
			Dify: &api.LLMSpecDify{
				PostgresImageId:     o.PostgresImageId,
				RedisImageId:        o.RedisImageId,
				NginxImageId:        o.NginxImageId,
				DifyApiImageId:      o.DifyApiImageId,
				DifyPluginImageId:   o.DifyPluginImageId,
				DifyWebImageId:      o.DifyWebImageId,
				DifySandboxImageId:  o.DifySandboxImageId,
				DifySSRFImageId:     o.DifySSRFImageId,
				DifyWeaviateImageId: o.DifyWeaviateImageId,
				CustomizedEnvs:      o.CustomizedEnvs,
			},
		}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	return dict, nil
}
