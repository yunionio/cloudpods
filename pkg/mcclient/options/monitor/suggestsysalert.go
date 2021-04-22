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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SuggestSysAlertListOptions struct {
	options.BaseListOptions
	Type     string `help:"Type of suggest rule"`
	Currency string `json:"currency"`
}

func (o *SuggestSysAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type SSuggestAlertShowOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
}

func (o *SSuggestAlertShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *SSuggestAlertShowOptions) GetId() string {
	return o.ID
}

type SuggestAlertIgnoreOptions struct {
	ID      string `help:"ID or name of the alert" json:"-"`
	Scope   string `help:"Resource scope" choices:"system|domain|project" default:"project"`
	Domain  string `help:"'Owner domain ID or Name" json:"project_domain"`
	Project string `help:"'Owner project ID or Name" json:"project"`
}

func (opt *SuggestAlertIgnoreOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opt)
}

func (o *SuggestAlertIgnoreOptions) GetId() string {
	return o.ID
}

type SuggestAlertCostOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
	options.BaseListOptions
	Currency string `json:"currency"`
}

func (o *SuggestAlertCostOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *SuggestAlertCostOptions) GetId() string {
	return o.ID
}

type SuggestAlertBatchDeleteOptions struct {
	ID []string `help:"ID of models to operate"`
}

func (o SuggestAlertBatchDeleteOptions) GetIds() []string {
	return o.ID
}

func (o SuggestAlertBatchDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
