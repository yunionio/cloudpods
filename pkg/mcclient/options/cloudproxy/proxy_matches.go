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

package cloudproxy

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ProxyMatchCreateOptions struct {
	NAME string

	ProxyEndpointId string `required:"true"`
	MatchScope      string `required:"true" choices:"vpc|network"`
	MatchValue      string `required:"true"`
}

func (opts *ProxyMatchCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ProxyMatchShowOptions struct {
	options.BaseShowOptions
}

type ProxyMatchUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	ProxyEndpointId string
	MatchScope      string `choices:"vpc|network"`
	MatchValue      string
}

func (opts *ProxyMatchUpdateOptions) GetId() string {
	return opts.ID
}

func (opts *ProxyMatchUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ProxyMatchDeleteOptions struct {
	options.BaseShowOptions
}

type ProxyMatchListOptions struct {
	options.BaseListOptions

	ProxyEndpointId string
	MatchScope      string `choices:"vpc|network"`
	MatchValue      string
}

func (opts *ProxyMatchListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}
