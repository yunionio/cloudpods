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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AiGatewayListOptions struct {
	options.BaseListOptions
}

func (opts *AiGatewayListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type AiGatewayIdOption struct {
	ID string `help:"Ai gateway Id"`
}

func (opts *AiGatewayIdOption) GetId() string {
	return opts.ID
}

func (opts *AiGatewayIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AiGatewayCreateOptions struct {
	options.BaseCreateOptions
	ManagerId string

	Authentication          bool   `json:"authentication"`
	CacheInvalidateOnUpdate bool   `json:"cache_invalidate_on_update"`
	CacheTTL                int    `json:"cache_ttl"`
	CollectLogs             bool   `json:"collect_logs"`
	RateLimitingInterval    int    `json:"rate_limiting_interval"`
	RateLimitingLimit       int    `json:"rate_limiting_limit"`
	RateLimitingTechnique   string `json:"rate_limiting_technique"`
}

func (opts *AiGatewayCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type AiGatewayChangeConfigOptions struct {
	AiGatewayIdOption
	Authentication          bool   `json:"authentication"`
	CacheInvalidateOnUpdate bool   `json:"cache_invalidate_on_update"`
	CacheTTL                int    `json:"cache_ttl"`
	CollectLogs             bool   `json:"collect_logs"`
	RateLimitingInterval    int    `json:"rate_limiting_interval"`
	RateLimitingLimit       int    `json:"rate_limiting_limit"`
	RateLimitingTechnique   string `json:"rate_limiting_technique"`
}

func (opts *AiGatewayChangeConfigOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
