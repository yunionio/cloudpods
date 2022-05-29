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

type LoadbalancerBackendCreateOptions struct {
	BackendGroup string `required:"true"`
	Backend      string `required:"true"`
	BackendType  string `default:"guest"`
	Port         *int   `required:"true"`
	Weight       *int   `default:"1"`

	SendProxy string `choices:"off|v1|v2|v2-ssl|v2-ssl-on"`
	Ssl       string `choices:"on|off"`
}

func (opts *LoadbalancerBackendCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type LoadbalancerBackendListOptions struct {
	options.BaseListOptions
	BackendGroup string
	Backend      string
	BackendType  string
	Weight       *int
	Address      string
	Port         *int

	SendProxy string `choices:"off|v1|v2|v2-ssl|v2-ssl-on"`
	Ssl       string `choices:"on|off"`
}

func (opts *LoadbalancerBackendListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type LoadbalancerBackendUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	Weight *int
	Port   *int

	SendProxy string `choices:"off|v1|v2|v2-ssl|v2-ssl-on"`
	Ssl       string `choices:"on|off"`
}

func (opts *LoadbalancerBackendUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type LoadbalancerBackendGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerBackendDeleteOptions struct {
	ID string `json:"-"`
}
