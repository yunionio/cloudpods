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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AiProxyUsageQueryOptions struct {
	Scope     string `help:"resource scope" choices:"system|domain|project" json:"scope"`
	Range     string `help:"preset range: 4h, 8h, 12h, 24h, today, yesterday, 7d, 30d, custom" json:"range"`
	Start     string `help:"range start (RFC3339, datetime, or unix seconds)" json:"start"`
	End       string `help:"range end (RFC3339, datetime, or unix seconds)" json:"end"`
	Timezone  string `help:"timezone for naive timestamps" json:"timezone"`
	Project   string `help:"filter by project id or name" json:"project"`
	Domain    string `help:"filter by domain id or name" json:"domain"`
	ApiKeyId  string `help:"filter by virtual key id" json:"api_key_id"`
	Model     string `help:"filter by model" json:"model"`
	Provider  string `help:"filter by provider" json:"provider"`
	Source    string `help:"filter by source" json:"source"`
	Result    string `help:"filter by result" choices:"success|failed" json:"result"`
	RequestId string `help:"filter by request id" json:"request_id"`
}

func (o *AiProxyUsageQueryOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AiProxyUsageOverviewOptions struct {
	AiProxyUsageQueryOptions
}

func (o *AiProxyUsageOverviewOptions) GetId() string {
	return "overview"
}

func (o *AiProxyUsageOverviewOptions) Params() (jsonutils.JSONObject, error) {
	return o.AiProxyUsageQueryOptions.Params()
}

type AiProxyUsageAnalysisOptions struct {
	AiProxyUsageQueryOptions
}

func (o *AiProxyUsageAnalysisOptions) GetId() string {
	return "analysis"
}

func (o *AiProxyUsageAnalysisOptions) Params() (jsonutils.JSONObject, error) {
	return o.AiProxyUsageQueryOptions.Params()
}

type AiProxyUsageEventListOptions struct {
	AiProxyUsageQueryOptions

	Limit  *int `help:"page limit" json:"limit"`
	Offset *int `help:"page offset" json:"offset"`
}

func (o *AiProxyUsageEventListOptions) GetId() string {
	return "events"
}

func (o *AiProxyUsageEventListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(&o.AiProxyUsageQueryOptions)
	if err != nil {
		return nil, err
	}
	extra, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	params.Update(extra)
	return params, nil
}
