// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aiproxy

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func mergeJSONStringField(params *jsonutils.JSONDict, key, raw string) error {
	if raw == "" {
		return nil
	}
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		return errors.Wrapf(err, "parse %s", key)
	}
	params.Set(key, obj)
	return nil
}

// --- ai_provider ---

type AiProviderListOptions struct {
	options.BaseListOptions

	ProviderKey     string `help:"filter by provider_key"`
	LlmDeploymentId string `help:"filter by llm_deployment_id" json:"llm_deployment_id"`
	LlmId           string `help:"filter by llm_id" json:"llm_id"`
}

func (o *AiProviderListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiProviderShowOptions struct {
	options.BaseShowOptions
}

type AiProviderCreateOptions struct {
	options.BaseCreateOptions

	ProviderKey     string `help:"provider key (catalog identifier)" json:"provider_key"`
	Config          string `help:"provider config as JSON object string" json:"-"`
	LlmDeploymentId string `help:"source llm_deployment id" json:"llm_deployment_id"`
	LlmId           string `help:"source llm instance id" json:"llm_id"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

func (o *AiProviderCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	params.Remove("config")
	if err := mergeJSONStringField(params, "config", o.Config); err != nil {
		return nil, err
	}
	return params, nil
}

type AiProviderUpdateOptions struct {
	ID              string `help:"ID or name" json:"-"`
	Name            string `json:"name,omitempty"`
	Desc            string `json:"description,omitempty"`
	ProviderKey     string `json:"provider_key,omitempty"`
	Config          string `help:"provider config JSON" json:"-"`
	LlmDeploymentId string `json:"llm_deployment_id,omitempty"`
	LlmId           string `json:"llm_id,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

func (o *AiProviderUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiProviderUpdateOptions) Params() (jsonutils.JSONObject, error) {
	d, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	if err := mergeJSONStringField(d, "config", o.Config); err != nil {
		return nil, err
	}
	return d, nil
}

type AiProviderDeleteOptions struct {
	options.BaseShowOptions
}

// --- ai_model ---

type AiModelListOptions struct {
	options.BaseListOptions

	AiProviderId string `help:"filter by ai_provider_id" json:"ai_provider_id"`
	ModelKey     string `help:"filter by model_key" json:"model_key"`
}

func (o *AiModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiModelShowOptions struct {
	options.BaseShowOptions
}

type AiModelCreateOptions struct {
	options.BaseCreateOptions

	AiProviderId string `help:"ai_provider id or name" json:"ai_provider_id"`
	ModelKey     string `help:"model routing key" json:"model_key"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (o *AiModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiModelUpdateOptions struct {
	ID           string `help:"ID or name" json:"-"`
	Name         string `json:"name,omitempty"`
	Desc         string `json:"description,omitempty"`
	AiProviderId string `json:"ai_provider_id,omitempty"`
	ModelKey     string `json:"model_key,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (o *AiModelUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiModelDeleteOptions struct {
	options.BaseShowOptions
}

// --- ai_key ---

type AiKeyListOptions struct {
	options.BaseListOptions

	AiProviderId string `help:"filter by ai_provider_id" json:"ai_provider_id"`
}

func (o *AiKeyListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiKeyShowOptions struct {
	options.BaseShowOptions
}

type AiKeyCreateOptions struct {
	options.BaseCreateOptions

	AiProviderId string `help:"optional ai_provider id or name" json:"ai_provider_id"`
	Secret       string `help:"API key or secret material" json:"secret"`
	Weight       int    `help:"load-balance weight among matching keys (default 1)" json:"weight,omitzero"`
	Routing      string `help:"routing JSON: allowed_model_keys, blocked_model_keys" json:"-"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (o *AiKeyCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	params.Remove("routing")
	if err := mergeJSONStringField(params, "routing", o.Routing); err != nil {
		return nil, err
	}
	return params, nil
}

type AiKeyUpdateOptions struct {
	ID           string `help:"ID or name" json:"-"`
	Name         string `json:"name,omitempty"`
	Desc         string `json:"description,omitempty"`
	AiProviderId string `json:"ai_provider_id,omitempty"`
	Secret       string `json:"secret,omitempty"`
	Weight       int    `json:"weight,omitzero"`
	Routing      string `json:"-"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (o *AiKeyUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiKeyUpdateOptions) Params() (jsonutils.JSONObject, error) {
	d, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	d.Remove("routing")
	if err := mergeJSONStringField(d, "routing", o.Routing); err != nil {
		return nil, err
	}
	return d, nil
}

type AiKeyDeleteOptions struct {
	options.BaseShowOptions
}

// --- ai_virtual_key ---

type AiVirtualKeyListOptions struct {
	options.BaseListOptions

	VirtualKey string `help:"filter by virtual_key" json:"virtual_key"`
	UserId     string `help:"filter by owner user id or name (admin)" json:"user_id"`
}

func (o *AiVirtualKeyListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiVirtualKeyShowOptions struct {
	options.BaseShowOptions
}

type AiVirtualKeyCreateOptions struct {
	options.BaseCreateOptions

	VirtualKey string `help:"optional client virtual key; auto-generated sk-... when omitted" json:"virtual_key"`
	OwnerId    string `help:"owner user id (admin); default current user" json:"owner_id,omitempty"`
	Limits     string `help:"limits JSON: allowed_ai_provider_ids, max_tokens_per_request, requests_per_minute" json:"-"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

func (o *AiVirtualKeyCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	params.Remove("limits")
	if err := mergeJSONStringField(params, "limits", o.Limits); err != nil {
		return nil, err
	}
	return params, nil
}

type AiVirtualKeyUpdateOptions struct {
	ID         string `help:"ID or name" json:"-"`
	Name       string `json:"name,omitempty"`
	Desc       string `json:"description,omitempty"`
	VirtualKey string `json:"virtual_key,omitempty"`
	OwnerId    string `json:"owner_id,omitempty"`
	Limits     string `help:"limits JSON object" json:"-"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

func (o *AiVirtualKeyUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiVirtualKeyUpdateOptions) Params() (jsonutils.JSONObject, error) {
	d, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	d.Remove("limits")
	if err := mergeJSONStringField(d, "limits", o.Limits); err != nil {
		return nil, err
	}
	return d, nil
}

type AiVirtualKeyDeleteOptions struct {
	options.BaseShowOptions
}

// --- ai_routing ---

type AiRoutingListOptions struct {
	options.BaseListOptions

	ModelPattern    string `json:"model_pattern"`
	AiProxyNodeId   string `json:"ai_proxy_node_id"`
	LlmDeploymentId string `help:"filter by llm_deployment_id" json:"llm_deployment_id"`
	Enabled         *bool  `help:"filter by enabled flag" json:"enabled"`
}

func (o *AiRoutingListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiRoutingShowOptions struct {
	options.BaseShowOptions
}

type AiRoutingCreateOptions struct {
	apis.SharableVirtualResourceCreateInput

	Priority        int    `json:"priority,omitzero"`
	ModelPattern    string `json:"model_pattern,omitempty"`
	AiProxyNodeId   string `json:"ai_proxy_node_id,omitempty"`
	LlmDeploymentId string `help:"source llm_deployment id" json:"llm_deployment_id"`
	Models          string `help:"routing models JSON array: [{ai_provider_id,ai_model_id,priority|weight,model_pattern?,llm_id?}]" json:"-"`
	Enabled         *bool  `help:"turn on enabled flag" json:"enabled,omitempty"`
	Disabled        *bool  `help:"turn off enabled flag" json:"disabled,omitempty"`
}

func (o *AiRoutingCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	if err := mergeJSONStringField(params, "models", o.Models); err != nil {
		return nil, err
	}
	return params, nil
}

type AiRoutingUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ID              string `help:"ID or name" json:"-"`
	Priority        int    `json:"priority,omitzero"`
	ModelPattern    string `json:"model_pattern,omitempty"`
	AiProxyNodeId   string `json:"ai_proxy_node_id,omitempty"`
	LlmDeploymentId string `json:"llm_deployment_id,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

func (o *AiRoutingUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiRoutingUpdateOptions) Params() (jsonutils.JSONObject, error) {
	d, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	baseParams, err := options.StructToParams(&o.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, err
	}
	if baseParams != nil {
		d.Update(baseParams)
	}
	return d, nil
}

type AiRoutingDeleteOptions struct {
	options.BaseShowOptions
}

type AiRoutingSetModelsOptions struct {
	options.BaseIdOptions

	Models string `help:"routing models JSON array: [{ai_provider_id,ai_model_id,priority|weight,model_pattern?}]" json:"-"`
}

func (o *AiRoutingSetModelsOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if err := mergeJSONStringField(params, "models", o.Models); err != nil {
		return nil, err
	}
	return params, nil
}

// --- ai_routing_model ---

type AiRoutingModelListOptions struct {
	options.BaseListOptions

	AiRoutingId  string `help:"filter by ai_routing id or name" json:"ai_routing_id"`
	AiProviderId string `json:"ai_provider_id"`
	AiModelId    string `json:"ai_model_id"`
	LlmId        string `help:"filter by llm_id" json:"llm_id"`
}

func (o *AiRoutingModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiRoutingModelShowOptions struct {
	options.BaseShowOptions
}

type AiRoutingModelCreateOptions struct {
	options.BaseCreateOptions

	AiRoutingId  string `help:"parent ai_routing id or name" json:"ai_routing_id"`
	AiProviderId string `help:"ai_provider id or name" json:"ai_provider_id"`
	AiModelId    string `help:"ai_model id or name" json:"ai_model_id"`
	Priority     int    `help:"lower value = higher priority within routing" json:"priority,omitzero"`
	ModelPattern string `help:"optional client model glob/prefix" json:"model_pattern,omitempty"`
	LlmId        string `help:"source llm instance id" json:"llm_id"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (o *AiRoutingModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiRoutingModelUpdateOptions struct {
	ID           string `help:"ID or name" json:"-"`
	Name         string `json:"name,omitempty"`
	Desc         string `json:"description,omitempty"`
	AiRoutingId  string `json:"ai_routing_id,omitempty"`
	AiProviderId string `json:"ai_provider_id,omitempty"`
	AiModelId    string `json:"ai_model_id,omitempty"`
	Priority     int    `json:"priority,omitzero"`
	ModelPattern string `json:"model_pattern,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (o *AiRoutingModelUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiRoutingModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiRoutingModelDeleteOptions struct {
	options.BaseShowOptions
}

// --- ai_proxy_node ---

type AiProxyNodeListOptions struct {
	options.BaseListOptions

	Address string `help:"filter by address" json:"address"`
	Domain  string `help:"filter by domain" json:"domain"`
}

func (o *AiProxyNodeListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AiProxyNodeShowOptions struct {
	options.BaseShowOptions
}

type AiProxyNodeCreateOptions struct {
	options.BaseCreateOptions

	Address   string `help:"reachable base URL (https://host:port or host:port)" json:"address"`
	Domain    string `help:"optional hostname without scheme or port" json:"domain,omitempty"`
	HbTimeout int    `help:"heartbeat timeout in seconds (default 120)" json:"hb_timeout,omitzero"`
	Enabled   *bool  `help:"turn on enabled flag" json:"enabled,omitempty"`
}

func (o *AiProxyNodeCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiProxyNodeUpdateOptions struct {
	ID        string `help:"ID or name" json:"-"`
	Name      string `json:"name,omitempty"`
	Desc      string `json:"description,omitempty"`
	Address   string `help:"reachable base URL" json:"address,omitempty"`
	Domain    string `help:"hostname without scheme or port; empty string clears" json:"domain,omitempty"`
	HbTimeout int    `help:"heartbeat timeout in seconds" json:"hb_timeout,omitzero"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

func (o *AiProxyNodeUpdateOptions) GetId() string {
	return o.ID
}

func (o *AiProxyNodeUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiProxyNodeDeleteOptions struct {
	options.BaseShowOptions
}

// AiProxyNodeRegisterOptions is used by standby instances (PerformClass register).
type AiProxyNodeRegisterOptions struct {
	Address   string `help:"instance address (https://host:port or host:port)" json:"address"`
	HbTimeout int    `help:"heartbeat timeout in seconds (default 120)" json:"hb_timeout,omitzero"`
}

func (o *AiProxyNodeRegisterOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
