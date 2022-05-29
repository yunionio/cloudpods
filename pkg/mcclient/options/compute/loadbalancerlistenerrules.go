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

type LoadbalancerListenerRuleCreateOptions struct {
	NAME         string
	Listener     string `required:"true"`
	BackendGroup string
	Domain       string
	Path         string

	HTTPRequestRate       *int
	HTTPRequestRatePerSrc *int

	Redirect       *string `choices:"off|raw"`
	RedirectCode   *int    `choices:"301|302|307"`
	RedirectScheme *string `json:",allowempty" choices:"http|https|"`
	RedirectHost   *string `json:",allowempty"`
	RedirectPath   *string `json:",allowempty"`
}

type LoadbalancerListenerRuleListOptions struct {
	options.BaseListOptions

	BackendGroup string
	Listener     string
	Domain       string
	Path         string

	Redirect       *string `choices:"off|raw"`
	RedirectCode   *int    `choices:"301|302|307"`
	RedirectScheme *string `choices:"http|https|" json:",allowempty"`
	RedirectHost   *string `json:",allowempty"`
	RedirectPath   *string `json:",allowempty"`
}

func (opts *LoadbalancerListenerRuleListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type LoadbalancerListenerRuleUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	BackendGroup string

	HTTPRequestRate       *int
	HTTPRequestRatePerSrc *int

	Redirect       *string `choices:"off|raw"`
	RedirectCode   *int    `choices:"301|302|307"`
	RedirectScheme *string `choices:"http|https|" json:",allowempty"`
	RedirectHost   *string `json:",allowempty"`
	RedirectPath   *string `json:",allowempty"`
}

func (opts *LoadbalancerListenerRuleUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type LoadbalancerListenerRuleGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerRuleDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerRuleGetBackendStatusOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerRuleActionStatusOptions struct {
	ID     string `json:"-"`
	Status string `choices:"enabled|disabled"`
}

func (opts *LoadbalancerListenerRuleActionStatusOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}
