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

type LoadbalancerClusterCreateOptions struct {
	NAME string

	Zone string `required:"true"`
	Wire string
}

func (opts *LoadbalancerClusterCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type LoadbalancerClusterUpdateOptions struct {
	ID string `json:"-"`

	Wire string
}

func (opts *LoadbalancerClusterUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type LoadbalancerClusterListOptions struct {
	options.BaseListOptions

	Zone string
	Wire string
}

func (opts *LoadbalancerClusterListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type LoadbalancerClusterGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerClusterDeleteOptions struct {
	ID string `json:"-"`
}
